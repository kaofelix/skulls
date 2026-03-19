package tui

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaofelix/skulls/internal/install"
	"github.com/kaofelix/skulls/internal/skillsapi"
)

// RunSearchFromSource opens the interactive selector using skills discovered
// from a repository source (local path or git source).
func RunSearchFromSource(source string) (SearchResult, error) {
	discovered, cleanup, err := install.DiscoverSkills(source)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return SearchResult{}, err
	}

	skills := make([]skillsapi.Skill, 0, len(discovered))
	filesBySkill := make(map[string]string, len(discovered))
	for _, d := range discovered {
		skills = append(skills, skillsapi.Skill{Source: source, SkillID: d.Name, Name: d.Name})
		filesBySkill[d.Name] = d.SkillFilePath
	}

	preview := func(_ context.Context, skill skillsapi.Skill) (string, error) {
		p, ok := filesBySkill[strings.TrimSpace(skill.SkillID)]
		if !ok {
			return "", skillsapi.ErrPreviewUnavailable
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return "", skillsapi.ErrPreviewUnavailable
		}
		return string(b), nil
	}

	openTarget := func(_ context.Context, skill skillsapi.Skill) (string, error) {
		p, ok := filesBySkill[strings.TrimSpace(skill.SkillID)]
		if !ok {
			return "", skillsapi.ErrPreviewUnavailable
		}
		absPath, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		return (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}).String(), nil
	}

	return RunSearchWithOptions(SearchOptions{
		InitialSkills:  skills,
		Placeholder:    "Filter skills…",
		StatusHint:     "Filter repository skills • Enter to install • Ctrl+O to open path • Esc to quit",
		PreviewFunc:    preview,
		OpenTargetFunc: openTarget,
	})
}
