//go:build linux && cgo && !gtk3 && !android && !server

package main

/*
#cgo pkg-config: gtk4
#include <gtk/gtk.h>

static int monitor_count(void) {
	GdkDisplay *d = gdk_display_get_default();
	if (!d) return 0;
	GListModel *l = gdk_display_get_monitors(d);
	return (int)g_list_model_get_n_items(l);
}

static int monitor_geometry(int i, int *x, int *y, int *w, int *h) {
	GdkDisplay *d = gdk_display_get_default();
	if (!d) return 0;
	GListModel *l = gdk_display_get_monitors(d);
	GdkMonitor *m = g_list_model_get_item(l, i);
	if (!m) return 0;
	GdkRectangle r;
	gdk_monitor_get_geometry(m, &r);
	*x = r.x; *y = r.y; *w = r.width; *h = r.height;
	g_object_unref(m);
	return 1;
}

static void fullscreen_on_monitor(void *win, int i) {
	GdkDisplay *d = gdk_display_get_default();
	if (!d || !win) return;
	GListModel *l = gdk_display_get_monitors(d);
	GdkMonitor *m = g_list_model_get_item(l, i);
	if (!m) return;
	gtk_window_fullscreen_on_monitor(GTK_WINDOW(win), m);
	g_object_unref(m);
}
*/
import "C"

import (
	"image"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// listMonitors asks GDK directly: on Wayland it is the only source that matches
// the compositor layout used by the portal screenshot.
func listMonitors(app *application.App) []monitor {
	var out []monitor
	application.InvokeSync(func() {
		n := int(C.monitor_count())
		for i := 0; i < n; i++ {
			var x, y, w, h C.int
			if C.monitor_geometry(C.int(i), &x, &y, &w, &h) == 1 {
				out = append(out, monitor{Rect: image.Rect(int(x), int(y), int(x+w), int(y+h))})
			}
		}
	})
	return out
}

func overlayOptions(o application.WebviewWindowOptions, _ monitor) application.WebviewWindowOptions {
	// Fullscreen is requested per monitor in afterCreate; Wails cannot target a monitor here.
	return o
}

// afterCreate fullscreens the window on monitor idx (index in GDK's monitor list).
func afterCreate(w *application.WebviewWindow, idx int) {
	application.InvokeSync(func() {
		if p := w.NativeWindow(); p != nil {
			C.fullscreen_on_monitor(p, C.int(idx))
		}
	})
}
