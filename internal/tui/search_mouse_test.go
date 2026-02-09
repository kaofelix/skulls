package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSearchModel_IsInPreviewPane(t *testing.T) {
	m := newSearchModel()
	m.listW = 48
	m.previewPaneW = 40
	m.bodyH = 20

	// Y is below the header (2 lines) and X is in the preview pane.
	if !m.isInPreviewPane(tea.MouseMsg{X: 60, Y: 5}) {
		t.Fatalf("expected mouse to be in preview pane")
	}

	// In list column.
	if m.isInPreviewPane(tea.MouseMsg{X: 10, Y: 5}) {
		t.Fatalf("expected mouse to be in list pane")
	}

	// In header area.
	if m.isInPreviewPane(tea.MouseMsg{X: 60, Y: 0}) {
		t.Fatalf("expected header area to not count as preview pane")
	}

	// Preview hidden.
	m.previewPaneW = 0
	if m.isInPreviewPane(tea.MouseMsg{X: 60, Y: 5}) {
		t.Fatalf("expected preview pane to be disabled")
	}
}
