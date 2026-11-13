//go:build linux

package capture

import (
	"fmt"
	"image"
	_ "image/png"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// desktop uses the XDG Desktop Portal on Wayland (direct capture is blocked there)
// and plain X11 capture otherwise.
func desktop() (image.Image, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return portalScreenshot()
	}
	return grabNative()
}

func portalScreenshot() (image.Image, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("session bus: %w", err)
	}
	defer conn.Close()

	// Request handle path is predictable from our unique name + token, so we can
	// subscribe to the Response signal before issuing the call (avoids a race).
	token := fmt.Sprintf("ocrtools%d", time.Now().UnixNano())
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	reqPath := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath(reqPath),
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return nil, err
	}
	sigs := make(chan *dbus.Signal, 4)
	conn.Signal(sigs)

	opts := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"interactive":  dbus.MakeVariant(false),
	}
	portal := conn.Object("org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop")
	if call := portal.Call("org.freedesktop.portal.Screenshot.Screenshot", 0, "", opts); call.Err != nil {
		return nil, fmt.Errorf("portal screenshot: %w", call.Err)
	}

	timeout := time.After(60 * time.Second)
	for {
		select {
		case sig := <-sigs:
			if sig.Path != reqPath || len(sig.Body) < 2 {
				continue
			}
			if code, _ := sig.Body[0].(uint32); code != 0 {
				return nil, fmt.Errorf("screenshot denied or cancelled (code %d)", code)
			}
			res, _ := sig.Body[1].(map[string]dbus.Variant)
			uri, _ := res["uri"].Value().(string)
			u, err := url.Parse(uri)
			if err != nil || u.Path == "" {
				return nil, fmt.Errorf("portal returned bad uri %q", uri)
			}
			f, err := os.Open(u.Path)
			if err != nil {
				return nil, err
			}
			defer func() { f.Close(); os.Remove(u.Path) }()
			img, _, err := image.Decode(f)
			return img, err
		case <-timeout:
			return nil, fmt.Errorf("portal screenshot timed out")
		}
	}
}
