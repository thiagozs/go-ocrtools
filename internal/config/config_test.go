package config

import (
	"path/filepath"
	"testing"
)

func TestStoreDefaultsAndRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "settings.json")
	s, err := NewStoreAt(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Get(); got != Defaults() {
		t.Fatalf("defaults = %+v", got)
	}
	if err := s.Save(Settings{Language: "", AutoCopy: false}); err != nil {
		t.Fatal(err)
	}
	s2, err := NewStoreAt(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get(); got.Language != "eng" || got.AutoCopy {
		t.Fatalf("reloaded = %+v", got)
	}
}

func TestHotkeyDefaultAndExplicitEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	s, _ := NewStoreAt(p)
	if s.Get().Hotkey == "" {
		t.Fatal("default hotkey should be set")
	}
	if err := s.Save(Settings{Language: "eng", Hotkey: ""}); err != nil {
		t.Fatal(err)
	}
	s2, _ := NewStoreAt(p)
	if s2.Get().Hotkey != "" {
		t.Fatal("explicitly disabled hotkey must stay disabled")
	}
}
