package tui

import "testing"

func TestWrapWidthForPreview(t *testing.T) {
	if got := wrapWidthForPreview(200); got != 80 {
		t.Fatalf("got %d want 80", got)
	}
	// subtracts 2
	if got := wrapWidthForPreview(50); got != 48 {
		t.Fatalf("got %d want 48", got)
	}
	// very small widths shouldn't underflow
	if got := wrapWidthForPreview(2); got != 1 {
		t.Fatalf("got %d want 1", got)
	}
}
