package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"

	"github.com/kaofelix/skulls/internal/install"
	"github.com/kaofelix/skulls/internal/skillsapi"
)

func TestInstallView_ShowsBannerAndContext(t *testing.T) {
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := installModel{
		targetDir: "~/agent/skills",
		skill:     skillsapi.Skill{SkillID: "demo-skill", Source: "owner/repo"},
		spin:      s,
		steps: map[install.Step]install.Event{
			install.StepNormalize: {Step: install.StepNormalize, Message: "Source normalized", Done: true},
		},
		order: []install.Step{install.StepNormalize},
	}

	view := m.View()
	if !strings.Contains(view, "Source:") {
		t.Fatalf("expected Source context line in view: %q", view)
	}
	if !strings.Contains(view, "Skill:") {
		t.Fatalf("expected Skill context line in view: %q", view)
	}
	if !strings.Contains(view, "Install dir:") {
		t.Fatalf("expected install dir context line in view: %q", view)
	}
}

func TestNewInstallModel_HidesNormalizeStepFromTimeline(t *testing.T) {
	m := newInstallModel("/tmp/skills", false, skillsapi.Skill{SkillID: "demo", Source: "owner/repo"})
	for _, step := range m.order {
		if step == install.StepNormalize {
			t.Fatalf("normalize step should not be shown in timeline order: %+v", m.order)
		}
	}
}

func TestInstallView_UsesConnectedStepTimeline(t *testing.T) {
	s := spinner.New()
	s.Spinner = spinner.Dot

	m := installModel{
		skill: skillsapi.Skill{SkillID: "demo"},
		spin:  s,
		steps: map[install.Step]install.Event{
			install.StepClone: {Step: install.StepClone, Message: "Cloned: https://github.com/obra/superpowers.git", Done: true},
		},
		order: []install.Step{install.StepClone, install.StepVerify},
	}

	view := m.View()
	if !strings.Contains(view, "│") {
		t.Fatalf("expected connected timeline bars in view: %q", view)
	}
	if strings.Contains(view, "...") {
		t.Fatalf("expected connected timeline style, got dot leader style: %q", view)
	}
	if !strings.Contains(view, "Cloned: https://github.com/obra/superpowers.git") {
		t.Fatalf("expected merged cloned message in timeline: %q", view)
	}
	if strings.Contains(view, "Repository cloned") {
		t.Fatalf("expected separate repository-cloned line to be gone: %q", view)
	}
}
