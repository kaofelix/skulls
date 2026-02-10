package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPreviewIndicatorLine_UsesPreviewWidth(t *testing.T) {
	m := newSearchModel()
	m.previewPaneW = 120
	m.previewVP.Width = wrapWidthForPreview(m.previewPaneW)
	m.previewVP.Height = 5
	m.previewVP.SetContent("a\nb\nc\nd\ne\nf\n")

	line := m.previewIndicatorLine()
	if got, want := lipgloss.Width(line), m.previewVP.Width; got != want {
		t.Fatalf("indicator width = %d, want %d", got, want)
	}
}
