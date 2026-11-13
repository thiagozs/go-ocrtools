package hotkey

import "testing"

func TestPortalTrigger(t *testing.T) {
	cases := map[string]string{
		"CmdOrCtrl+Alt+O":      "CTRL+ALT+o",
		"Ctrl+Shift+Space":     "CTRL+SHIFT+space",
		"Super+F5":             "LOGO+F5",
		"CmdOrCtrl+ctrl+Alt+1": "CTRL+ALT+1",
	}
	for in, want := range cases {
		got, err := portalTrigger(in)
		if err != nil || got != want {
			t.Errorf("%q = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"O", "Foo+O", "Ctrl+", ""} {
		if _, err := portalTrigger(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
