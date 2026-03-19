package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaofelix/skulls/internal/skillsapi"
)

func TestSearchModel_OpenSelectedSkillWithShortcut(t *testing.T) {
	oldOpenExternalTarget := openExternalTarget
	defer func() { openExternalTarget = oldOpenExternalTarget }()

	var opened string
	openExternalTarget = func(target string) error {
		opened = target
		return nil
	}

	m := newSearchModelWithOptions(SearchOptions{
		InitialSkills: []skillsapi.Skill{{Source: "owner/repo", SkillID: "test-skill", Name: "Test Skill"}},
		OpenTargetFunc: func(_ context.Context, skill skillsapi.Skill) (string, error) {
			return "https://example.com/" + skill.SkillID, nil
		},
	})

	if m.results.SelectedItem() == nil {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		m = m2.(searchModel)
	}

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m = m2.(searchModel)
	if cmd == nil {
		t.Fatal("expected open command")
	}

	msg := cmd()
	openMsg, ok := msg.(openResultMsg)
	if !ok {
		t.Fatalf("expected openResultMsg, got %T", msg)
	}
	if openMsg.err != nil {
		t.Fatalf("open failed: %v", openMsg.err)
	}
	if opened != "https://example.com/test-skill" {
		t.Fatalf("opened target: got %q", opened)
	}

	m3, _ := m.Update(openMsg)
	m = m3.(searchModel)
	if !strings.Contains(m.statusMessage, "Opened:") {
		t.Fatalf("expected opened status message, got %q", m.statusMessage)
	}
}

func TestSearchModel_ViewIncludesOpenPathHint(t *testing.T) {
	m := newSearchModelWithOptions(SearchOptions{
		InitialSkills: []skillsapi.Skill{{Source: "owner/repo", SkillID: "test-skill", Name: "Test Skill"}},
	})

	view := m.View()
	if !strings.Contains(view, "Ctrl+O to open path") {
		t.Fatalf("expected open path hint in view, got %q", view)
	}
}
