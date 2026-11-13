//go:build linux

package hotkey

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/thiagozs/ocrtools/internal/launcher"
)

const (
	portalDest  = "org.freedesktop.portal.Desktop"
	portalPath  = "/org/freedesktop/portal/desktop"
	shortcutsIf = "org.freedesktop.portal.GlobalShortcuts"
	shortcutID  = "capture"
)

// portal is a minimal GlobalShortcuts portal client. Wails' own Wayland backend
// never registers the app id, which GNOME requires for apps not started from a
// launcher, so we talk to the portal ourselves.
type portal struct {
	icon []byte

	opMu sync.Mutex // serializes apply() runs
	mu   sync.Mutex // guards the fields below
	conn *dbus.Conn
	pend map[dbus.ObjectPath]chan portalResp
	sess dbus.ObjectPath
	cb   func()
	gen  int
	cur  string
}

type portalResp struct {
	code    uint32
	results map[string]dbus.Variant
}

func newPortal(icon []byte) *portal {
	return &portal{icon: icon, pend: map[dbus.ObjectPath]chan portalResp{}}
}

// Set validates accel synchronously and binds it in the background: the first
// BindShortcuts may block on a GNOME confirmation dialog.
func (p *portal) Set(accel string, cb func()) error {
	p.mu.Lock()
	if accel == p.cur {
		p.mu.Unlock()
		return nil
	}
	trigger := ""
	if accel != "" {
		var err error
		if trigger, err = portalTrigger(accel); err != nil {
			p.mu.Unlock()
			return err
		}
	}
	p.cur = accel
	p.cb = cb
	p.gen++
	gen := p.gen
	p.mu.Unlock()

	go func() {
		if err := p.apply(gen, trigger); err != nil {
			log.Printf("atalho global (portal): %v", err)
		}
	}()
	return nil
}

func (p *portal) apply(gen int, trigger string) error {
	p.opMu.Lock()
	defer p.opMu.Unlock()
	if p.stale(gen) {
		return nil
	}

	if err := p.connect(); err != nil {
		return err
	}
	p.closeSession()
	if trigger == "" {
		return nil
	}

	resp, err := p.request("CreateSession", map[string]dbus.Variant{
		"session_handle_token": dbus.MakeVariant(fmt.Sprintf("s%d", time.Now().UnixNano())),
	})
	if err != nil {
		return err
	}
	if resp.code != 0 {
		return fmt.Errorf("CreateSession recusado (código %d)", resp.code)
	}
	h, _ := resp.results["session_handle"].Value().(string)
	if h == "" {
		return fmt.Errorf("portal não devolveu a sessão")
	}
	sess := dbus.ObjectPath(h)
	p.mu.Lock()
	p.sess = sess
	p.mu.Unlock()

	type entry struct {
		ID    string
		Props map[string]dbus.Variant
	}
	list := []entry{{shortcutID, map[string]dbus.Variant{
		"description":       dbus.MakeVariant("Capturar texto da tela (OCR)"),
		"preferred_trigger": dbus.MakeVariant(trigger),
	}}}
	resp, err = p.request("BindShortcuts", map[string]dbus.Variant{}, sess, list, "")
	if err != nil {
		return fmt.Errorf("BindShortcuts: %w", err)
	}
	if resp.code != 0 {
		return fmt.Errorf("atalho não concedido (código %d; 1 = cancelado no diálogo)", resp.code)
	}
	log.Printf("atalho global ativo (pedido %q): %v", trigger, resp.results["shortcuts"].Value())
	return nil
}

func (p *portal) stale(gen int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return gen != p.gen
}

func (p *portal) closeSession() {
	p.mu.Lock()
	sess, conn := p.sess, p.conn
	p.sess = ""
	p.mu.Unlock()
	if sess != "" && conn != nil {
		_ = conn.Object(portalDest, sess).Call("org.freedesktop.portal.Session.Close", 0).Err
	}
}

// connect opens the bus, registers the app id and starts the signal loop (once).
func (p *portal) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		return nil
	}
	if err := launcher.Ensure(p.icon); err != nil {
		return fmt.Errorf("criar launcher .desktop: %w", err)
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("session bus: %w", err)
	}
	reg := conn.Object(portalDest, portalPath).Call("org.freedesktop.host.portal.Registry.Register", 0, launcher.AppID, map[string]dbus.Variant{})
	if reg.Err != nil {
		conn.Close()
		return fmt.Errorf("registrar app id %s: %w", launcher.AppID, reg.Err)
	}
	for _, m := range [][]dbus.MatchOption{
		{dbus.WithMatchInterface("org.freedesktop.portal.Request"), dbus.WithMatchMember("Response")},
		{dbus.WithMatchInterface(shortcutsIf), dbus.WithMatchMember("Activated")},
	} {
		if err := conn.AddMatchSignal(m...); err != nil {
			conn.Close()
			return err
		}
	}
	sigs := make(chan *dbus.Signal, 16)
	conn.Signal(sigs)
	p.conn = conn
	go p.loop(sigs)
	return nil
}

func (p *portal) loop(sigs chan *dbus.Signal) {
	for sig := range sigs {
		switch {
		case strings.HasSuffix(sig.Name, ".Response"):
			var r portalResp
			if len(sig.Body) >= 2 {
				r.code, _ = sig.Body[0].(uint32)
				r.results, _ = sig.Body[1].(map[string]dbus.Variant)
			}
			p.mu.Lock()
			ch := p.pend[sig.Path]
			p.mu.Unlock()
			if ch != nil {
				select {
				case ch <- r:
				default:
				}
			}
		case strings.HasSuffix(sig.Name, ".Activated"):
			if len(sig.Body) < 2 {
				continue
			}
			sess, _ := sig.Body[0].(dbus.ObjectPath)
			id, _ := sig.Body[1].(string)
			p.mu.Lock()
			cb, ok := p.cb, sess == p.sess && id == shortcutID
			p.mu.Unlock()
			if ok && cb != nil {
				go cb()
			}
		}
	}
}

// request calls a portal method that answers through a Request/Response signal.
func (p *portal) request(method string, opts map[string]dbus.Variant, args ...any) (portalResp, error) {
	p.mu.Lock()
	conn := p.conn
	p.mu.Unlock()
	token := fmt.Sprintf("r%d", time.Now().UnixNano())
	opts["handle_token"] = dbus.MakeVariant(token)
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	path := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)

	ch := make(chan portalResp, 1)
	p.mu.Lock()
	p.pend[path] = ch
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.pend, path)
		p.mu.Unlock()
	}()

	call := conn.Object(portalDest, portalPath).Call(shortcutsIf+"."+method, 0, append(args, opts)...)
	if call.Err != nil {
		return portalResp{}, call.Err
	}
	select {
	case r := <-ch:
		return r, nil
	case <-time.After(5 * time.Minute): // user may take a while on the GNOME dialog
		return portalResp{}, fmt.Errorf("tempo esgotado esperando o portal (%s)", method)
	}
}
