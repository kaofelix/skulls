package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/kaofelix/skulls/internal/gitutil"
	"github.com/kaofelix/skulls/internal/skilllayout"
)

type DiscoveredSkill struct {
	Name          string
	SkillDirPath  string
	SkillFilePath string
}

type discoverOptions struct {
	FullDepth bool
}

// DiscoverSkills discovers skills in a source repository using the same spirit
// as vercel-labs/skills:
//   - root SKILL.md (early return by default)
//   - priority skill directories
//   - recursive fallback
func DiscoverSkills(source string) ([]DiscoveredSkill, func(), error) {
	cloneURL, err := gitutil.NormalizeSourceToGitURL(source)
	if err != nil {
		return nil, nil, err
	}

	repoDir := cloneURL
	cleanup := func() {}

	if fi, err := os.Stat(cloneURL); err != nil || !fi.IsDir() {
		tmp, err := os.MkdirTemp("", "skulls-discover-*")
		if err != nil {
			return nil, nil, err
		}
		repoDir = filepath.Join(tmp, "repo")
		if err := gitutil.CloneShallowTo(cloneURL, repoDir, io.Discard, io.Discard); err != nil {
			_ = os.RemoveAll(tmp)
			return nil, nil, err
		}
		cleanup = func() { _ = os.RemoveAll(tmp) }
	}

	skills, err := discoverSkillsInRepo(repoDir, discoverOptions{})
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return skills, cleanup, nil
}

func discoverSkillsInRepo(repoDir string, opts discoverOptions) ([]DiscoveredSkill, error) {
	out := make([]DiscoveredSkill, 0, 16)
	seen := map[string]struct{}{}

	addSkill := func(skillFilePath string) error {
		b, err := os.ReadFile(skillFilePath)
		if err != nil {
			return err
		}
		fm, ok := parseSkillFrontmatter(string(b))
		if !ok {
			return nil
		}
		if _, exists := seen[fm.Name]; exists {
			return nil
		}
		seen[fm.Name] = struct{}{}
		out = append(out, DiscoveredSkill{
			Name:          fm.Name,
			SkillDirPath:  filepath.Dir(skillFilePath),
			SkillFilePath: skillFilePath,
		})
		return nil
	}

	rootSkill := filepath.Join(repoDir, "SKILL.md")
	if fi, err := os.Stat(rootSkill); err == nil && !fi.IsDir() {
		if err := addSkill(rootSkill); err != nil {
			return nil, err
		}
		if len(out) > 0 && !opts.FullDepth {
			sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
			return out, nil
		}
	}

	for _, relDir := range skilllayout.PrioritySearchDirs {
		dir := filepath.Join(repoDir, filepath.FromSlash(relDir))
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
			if fi, err := os.Stat(skillFile); err == nil && !fi.IsDir() {
				if err := addSkill(skillFile); err != nil {
					return nil, err
				}
			}
		}
	}

	if len(out) == 0 || opts.FullDepth {
		if err := walkSkillDirsRecursive(repoDir, 0, addSkill); err != nil {
			return nil, err
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no skills found")
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out, nil
}

func walkSkillDirsRecursive(dir string, depth int, addSkill func(string) error) error {
	if depth > skilllayout.MaxRecursiveDepth {
		return nil
	}

	skillFile := filepath.Join(dir, "SKILL.md")
	if fi, err := os.Stat(skillFile); err == nil && !fi.IsDir() {
		if err := addSkill(skillFile); err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if skilllayout.ShouldSkipDir(e.Name()) {
			continue
		}
		if err := walkSkillDirsRecursive(filepath.Join(dir, e.Name()), depth+1, addSkill); err != nil {
			return err
		}
	}
	return nil
}
