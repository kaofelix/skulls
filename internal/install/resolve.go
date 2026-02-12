package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveSkillDir returns the directory containing the skill files.
//
// Fast path: repo/skills/<skillID>/SKILL.md with matching strict frontmatter.
// Fallback: faithful repository discovery and frontmatter name matching.
func resolveSkillDir(repoDir string, skillID string) (string, error) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return "", fmt.Errorf("skill directory not found in repo: %s", filepath.ToSlash(filepath.Join("skills", skillID)))
	}

	expected := filepath.Join(repoDir, "skills", skillID, "SKILL.md")
	if fi, err := os.Stat(expected); err == nil && !fi.IsDir() {
		b, err := os.ReadFile(expected)
		if err == nil {
			if fm, ok := parseSkillFrontmatter(string(b)); ok && fm.Name == skillID {
				return filepath.Dir(expected), nil
			}
		}
	}

	skills, err := discoverSkillsInRepo(repoDir, discoverOptions{})
	if err != nil {
		return "", fmt.Errorf("skill directory not found in repo: %s", filepath.ToSlash(filepath.Join("skills", skillID)))
	}

	for _, s := range skills {
		if s.Name == skillID {
			return s.SkillDirPath, nil
		}
	}

	return "", fmt.Errorf("skill directory not found in repo: %s", filepath.ToSlash(filepath.Join("skills", skillID)))
}
