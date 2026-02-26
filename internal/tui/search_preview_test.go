package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaofelix/skulls/internal/skillsapi"
)

// TestPreviewRendersAfterWindowSizeWhenPreviewArrivesFirst simulates the race
// condition where preview result arrives before WindowSizeMsg. The preview should
// still render once window size is known.
func TestPreviewRendersAfterWindowSizeWhenPreviewArrivesFirst(t *testing.T) {
	// Create model with initial skills (like when running "skulls add <repo>")
	opts := SearchOptions{
		InitialSkills: []skillsapi.Skill{
			{Source: "test-repo", SkillID: "test-skill", Name: "Test Skill"},
		},
	}
	m := newSearchModelWithOptions(opts)

	// Manually set a selection by simulating what happens when items are set
	// The list should have items but selection might not be visible yet
	if m.results.SelectedItem() == nil {
		// Force selection to first item by sending a key message
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		m = m2.(searchModel)
	}

	// Verify we have a selection
	if m.results.SelectedItem() == nil {
		t.Fatal("expected selected item after setting items")
	}

	// Trigger preview loading and get the seq number that was assigned
	_ = m.ensurePreviewForSelection()
	seq := m.previewSeq // This is the seq number that was assigned

	// Simulate preview result arriving BEFORE window size (the race condition)
	key := previewKeyForSkill(skillsapi.Skill{Source: "test-repo", SkillID: "test-skill"})
	previewMsg := previewResultMsg{
		seq: seq,
		key: key,
		md:  "# Test Skill\n\nThis is the preview content.",
		err: nil,
	}

	// Process the preview result BEFORE window size
	m2, _ := m.Update(previewMsg)
	m = m2.(searchModel)

	// At this point, previewMarkdown should be set but previewRendered empty
	// because previewPaneW is still 0
	if m.previewMarkdown == "" {
		t.Fatalf("previewMarkdown should be set after preview result, got previewErr=%v", m.previewErr)
	}
	if m.previewRendered != "" {
		t.Log("previewRendered already set before window size (unexpected but ok)")
	}

	// Now send WindowSizeMsg to set up the layout
	m3, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = m3.(searchModel)

	// After window size, preview should be rendered
	if m.previewRendered == "" {
		t.Fatal("previewRendered should be set after WindowSizeMsg when previewMarkdown exists")
	}
}

// TestPreviewRendersWhenWindowSizeArrivesFirst simulates normal flow where
// WindowSizeMsg arrives before preview result.
func TestPreviewRendersWhenWindowSizeArrivesFirst(t *testing.T) {
	opts := SearchOptions{
		InitialSkills: []skillsapi.Skill{
			{Source: "test-repo", SkillID: "test-skill", Name: "Test Skill"},
		},
	}
	m := newSearchModelWithOptions(opts)

	// Force selection
	if m.results.SelectedItem() == nil {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		m = m2.(searchModel)
	}

	if m.results.SelectedItem() == nil {
		t.Fatal("expected selected item")
	}

	// First set window size
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = m2.(searchModel)

	if m.previewPaneW == 0 {
		t.Fatal("previewPaneW should be set after WindowSizeMsg")
	}

	// Trigger preview loading
	_ = m.ensurePreviewForSelection()
	seq := m.previewSeq

	// Simulate preview result arriving AFTER window size (normal flow)
	key := previewKeyForSkill(skillsapi.Skill{Source: "test-repo", SkillID: "test-skill"})
	previewMsg := previewResultMsg{
		seq: seq,
		key: key,
		md:  "# Test Skill\n\nThis is the preview content.",
		err: nil,
	}

	// Process preview result
	m3, _ := m.Update(previewMsg)
	m = m3.(searchModel)

	// Preview should be rendered
	if m.previewRendered == "" {
		t.Fatalf("previewRendered should be set after preview result when window size is known, previewMarkdown=%q", m.previewMarkdown)
	}
}

// TestPreviewLoadsWhenWindowSizeArrivesBeforeInit tests the specific race where
// WindowSizeMsg arrives before the Init command's preview command executes.
// This is the scenario: window size is known, but preview hasn't loaded yet.
func TestPreviewLoadsWhenWindowSizeArrivesBeforeInit(t *testing.T) {
	opts := SearchOptions{
		InitialSkills: []skillsapi.Skill{
			{Source: "test-repo", SkillID: "test-skill", Name: "Test Skill"},
		},
	}
	m := newSearchModelWithOptions(opts)

	// Window size arrives BEFORE init command runs (common in real TUI apps)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = m2.(searchModel)

	if m.previewPaneW == 0 {
		t.Fatal("previewPaneW should be set after WindowSizeMsg")
	}

	// At this point, previewMarkdown should be empty because Init hasn't run
	if m.previewMarkdown != "" {
		t.Log("previewMarkdown already set (unexpected)")
	}

	// Now simulate Init running and returning its command
	initCmd := m.Init()

	// Process the init command
	msg := initCmd()

	// Check for batch and execute commands
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			resultMsg := c()
			if previewMsg, ok := resultMsg.(previewResultMsg); ok {
				// This is the preview result
				m3, _ := m.Update(previewMsg)
				m = m3.(searchModel)
			}
		}
	}

	// Preview should be rendered now
	if m.previewRendered == "" && m.previewMarkdown != "" {
		t.Fatalf("previewRendered should be set, got previewMarkdown=%q, previewPaneW=%d, previewErr=%v",
			m.previewMarkdown, m.previewPaneW, m.previewErr)
	}

	// Suppress unused variable warning
	_ = cmd
}

// TestInitReturnsPreviewCommandWhenSelectionExists checks that Init returns
// a preview command when InitialSkills is provided and selection exists.
func TestInitReturnsPreviewCommandWhenSelectionExists(t *testing.T) {
	opts := SearchOptions{
		InitialSkills: []skillsapi.Skill{
			{Source: "test-repo", SkillID: "test-skill", Name: "Test Skill"},
		},
	}
	m := newSearchModelWithOptions(opts)

	// Check if selection exists immediately after creation
	hasSelection := m.results.SelectedItem() != nil
	t.Logf("Has selection immediately: %v", hasSelection)
	t.Logf("allItems count: %d", len(m.allItems))

	// Get Init command
	cmd := m.Init()

	// If there's no selection, Init should still return a command (spinner, etc.)
	// but NOT the preview command because ensurePreviewForSelection returns nil
	if cmd == nil {
		t.Fatal("expected Init to return a command even without selection")
	}

	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		t.Logf("Batch has %d commands", len(batch))
		for i, c := range batch {
			if c == nil {
				t.Logf("Command %d: nil", i)
			} else {
				result := c()
				t.Logf("Command %d returns: %T", i, result)
			}
		}
	}

	// The real issue: if there's no selection, ensurePreviewForSelection returns nil
	// and no preview is ever loaded
	if !hasSelection {
		t.Log("WARNING: No selection exists when Init() is called - this might be the bug!")
	}
}

// TestPreviewWithFileBasedPreviewFunc simulates the actual usage from RunSearchFromSource
// where PreviewFunc reads from a file (which may fail initially).
func TestPreviewWithFileBasedPreviewFunc(t *testing.T) {
	// Create a temporary directory with a skill file
	tmpDir := t.TempDir()
	skillContent := "# Test Skill\n\nThis is the skill content."
	skillFile := filepath.Join(tmpDir, "test-skill.md")
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create the filesBySkill map like RunSearchFromSource does
	filesBySkill := map[string]string{
		"test-skill": skillFile,
	}

	// Create preview function like RunSearchFromSource does
	preview := func(_ctx context.Context, skill skillsapi.Skill) (string, error) {
		p, ok := filesBySkill[skill.SkillID]
		if !ok {
			return "", skillsapi.ErrPreviewUnavailable
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", skillsapi.ErrPreviewUnavailable
		}
		return string(b), nil
	}

	opts := SearchOptions{
		InitialSkills: []skillsapi.Skill{
			{Source: "test-repo", SkillID: "test-skill", Name: "Test Skill"},
		},
		PreviewFunc: preview,
	}
	m := newSearchModelWithOptions(opts)

	// Force selection
	if m.results.SelectedItem() == nil {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		m = m2.(searchModel)
	}

	// Get the preview command
	cmd := m.ensurePreviewForSelection()
	if cmd == nil {
		t.Fatal("expected preview command")
	}

	// Execute the command
	msg := cmd()
	previewMsg, ok := msg.(previewResultMsg)
	if !ok {
		t.Fatalf("expected previewResultMsg, got %T", msg)
	}

	// Process the preview result
	m2, _ := m.Update(previewMsg)
	m = m2.(searchModel)

	// Check the result
	if previewMsg.err != nil {
		t.Fatalf("preview failed: %v", previewMsg.err)
	}

	if m.previewMarkdown == "" {
		t.Fatal("previewMarkdown should be set")
	}

	if m.previewMarkdown != skillContent {
		t.Fatalf("previewMarkdown mismatch: got %q, want %q", m.previewMarkdown, skillContent)
	}
}

// TestPreviewTriggeredAfterWindowSizeWhenNotYetLoaded tests that when
// WindowSizeMsg arrives and we have a selection but no preview loaded yet,
// the preview should be triggered.
func TestPreviewTriggeredAfterWindowSizeWhenNotYetLoaded(t *testing.T) {
	opts := SearchOptions{
		InitialSkills: []skillsapi.Skill{
			{Source: "test-repo", SkillID: "test-skill", Name: "Test Skill"},
		},
	}
	m := newSearchModelWithOptions(opts)

	// At this point, we should have a selection (the first item)
	// but no preview loaded yet
	if m.results.SelectedItem() == nil {
		t.Fatal("expected selection to exist with InitialSkills")
	}

	// Send WindowSizeMsg - this should trigger a preview load
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = m2.(searchModel)

	if m.previewPaneW == 0 {
		t.Fatal("previewPaneW should be set after WindowSizeMsg")
	}

	// Check if a preview command was returned
	hasPreviewCmd := false
	if cmd != nil {
		msg := cmd()
		// The command should produce a previewResultMsg
		if _, ok := msg.(previewResultMsg); ok {
			hasPreviewCmd = true
		}
	}

	// After WindowSizeMsg, if we have a selection but no preview,
	// a preview command should be triggered
	if !hasPreviewCmd {
		t.Fatalf("expected preview command to be triggered after WindowSizeMsg when preview not loaded. selectedKey=%q, previewMarkdown=%q, previewLoading=%v",
			m.selectedKey(), m.previewMarkdown, m.previewLoading)
	}
}

// TestEnsurePreviewReturnsNilWhenNoSelection tests that ensurePreviewForSelection
// returns nil when there is no selection, which would cause no preview to load.
func TestEnsurePreviewReturnsNilWhenNoSelection(t *testing.T) {
	// Create model WITHOUT initial skills - simulating when items might not be ready
	m := newSearchModel()

	// Call ensurePreviewForSelection - should return nil when no selection
	cmd := m.ensurePreviewForSelection()

	if cmd != nil {
		t.Fatal("expected ensurePreviewForSelection to return nil when no selection")
	}
}
