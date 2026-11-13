package main

import (
	"image"
	"testing"
)

// Mirrors the real setup: 3 monitors of 1920x1080, the last one offset by 6px.
func threeMonitors() []monitor {
	return []monitor{
		{Rect: image.Rect(0, 0, 1920, 1080)},
		{Rect: image.Rect(1920, 0, 3840, 1080)},
		{Rect: image.Rect(3840, 6, 5760, 1086)},
	}
}

func TestMonitorPixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 5760, 1086))
	want := []image.Rectangle{
		image.Rect(0, 0, 1920, 1080),
		image.Rect(1920, 0, 3840, 1080),
		image.Rect(3840, 6, 5760, 1086),
	}
	for i, w := range want {
		got, ok := monitorPixels(img, threeMonitors(), i)
		if !ok || got != w {
			t.Errorf("monitor %d = %v ok=%v, want %v", i, got, ok, w)
		}
	}
	if _, ok := monitorPixels(img, threeMonitors(), 3); ok {
		t.Error("out-of-range monitor should fail")
	}
}

func TestMonitorPixelsHiDPIScreenshot(t *testing.T) {
	// Screenshot at 2x the compositor layout.
	img := image.NewRGBA(image.Rect(0, 0, 3840, 1080))
	mons := []monitor{{Rect: image.Rect(0, 0, 960, 540)}, {Rect: image.Rect(960, 0, 1920, 540)}}
	got, ok := monitorPixels(img, mons, 1)
	if want := image.Rect(1920, 0, 3840, 1080); !ok || got != want {
		t.Fatalf("got %v ok=%v, want %v", got, ok, want)
	}
}
