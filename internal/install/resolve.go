package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// resolveSkillDir returns the directory containing the skill files.
//
// Fast path: repo/skills/<skillID>/SKILL.md
// Fallback: scan for SKILL.md under repo/skills and match YAML frontmatter name.
func resolveSkillDir(repoDir string, skillID string) (string, error) {
	// Fast path
	expected := filepath.Join(repoDir, "skills", skillID)
	if fi, err := os.Stat(expected); err == nil && fi.IsDir() {
		skillMd := filepath.Join(expected, "SKILL.md")
		if fi, err := os.Stat(skillMd); err == nil && !fi.IsDir() {
			return expected, nil
		}
		return "", fmt.Errorf("SKILL.md not found at %s", filepath.ToSlash(filepath.Join("skills", skillID, "SKILL.md")))
	}

	// Fallback: scan repo/skills/**/SKILL.md
	root := filepath.Join(repoDir, "skills")
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("skill directory not found in repo: %s", filepath.ToSlash(filepath.Join("skills", skillID)))
	}

	var matches []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "SKILL.md") {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			name, ok := parseSkillNameFromFrontmatter(string(b))
			if ok && name == skillID {
				matches = append(matches, filepath.Dir(p))
			}
		}
		return nil
	})

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("skill directory not found in repo: %s", filepath.ToSlash(filepath.Join("skills", skillID)))
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple skills matched %q: %s", skillID, strings.Join(matches, ", "))
	}
}

func parseSkillNameFromFrontmatter(md string) (string, bool) {
	// Accept both \n and \r\n.
	md = strings.ReplaceAll(md, "\r\n", "\n")
	lines := strings.Split(md, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", false
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", false
	}
	yamlText := strings.Join(lines[1:end], "\n")

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(yamlText), &fm); err != nil {
		return "", false
	}
	v, ok := fm["name"]
	if !ok {
		return "", false
	}
	name, ok := v.(string)
	if !ok {
		return "", false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	return name, true
}
