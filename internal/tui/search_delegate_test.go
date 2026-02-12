package tui

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewTerminalListDelegate_UsesTerminalANSIAndNoAdaptiveColors(t *testing.T) {
	d := newTerminalListDelegate()

	if got := d.Styles.SelectedTitle.GetForeground(); !reflect.DeepEqual(got, lipgloss.Color("6")) {
		t.Fatalf("selected title foreground=%v, want ANSI 6", got)
	}
	if got := d.Styles.SelectedTitle.GetBorderLeftForeground(); !reflect.DeepEqual(got, lipgloss.Color("6")) {
		t.Fatalf("selected title border foreground=%v, want ANSI 6", got)
	}
	if got := d.Styles.SelectedDesc.GetForeground(); !reflect.DeepEqual(got, lipgloss.Color("6")) {
		t.Fatalf("selected desc foreground=%v, want ANSI 6", got)
	}

	if _, ok := d.Styles.SelectedTitle.GetForeground().(lipgloss.AdaptiveColor); ok {
		t.Fatalf("selected title should not use adaptive color")
	}
	if _, ok := d.Styles.SelectedDesc.GetForeground().(lipgloss.AdaptiveColor); ok {
		t.Fatalf("selected desc should not use adaptive color")
	}
}
