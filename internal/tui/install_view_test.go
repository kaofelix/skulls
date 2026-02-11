package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"

	"github.com/kaofelix/skulls/internal/install"
	"github.com/kaofelix/skulls/internal/skillsapi"
)

func TestFormatStepLine_IncludesDotLeaderAndStatus(t *testing.T) {
	line := formatStepLine("Download repository", "✓", 40)
	if line == "" {
		t.Fatalf("line should not be empty")
	}
	if !strings.Contains(line, "Download repository") {
		t.Fatalf("missing label: %q", line)
	}
	if !strings.Contains(line, "✓") {
		t.Fatalf("missing status: %q", line)
	}
	if !strings.Contains(line, ".") {
		t.Fatalf("expected dot leader: %q", line)
	}
}

func TestInstallView_UsesDotLeadersForSteps(t *testing.T) {
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := installModel{
		skill: skillsapi.Skill{SkillID: "demo"},
		spin:  s,
		steps: map[install.Step]install.Event{
			install.StepNormalize: {Step: install.StepNormalize, Done: true},
			install.StepClone:     {Step: install.StepClone, Done: false},
		},
		order: []install.Step{install.StepNormalize, install.StepClone, install.StepVerify},
	}

	view := m.View()
	if !strings.Contains(view, "Normalize source") {
		t.Fatalf("missing normalize step: %q", view)
	}
	if !strings.Contains(view, "Download repository") {
		t.Fatalf("missing clone step: %q", view)
	}
	if !strings.Contains(view, ".") {
		t.Fatalf("expected dot leaders in view: %q", view)
	}
}
