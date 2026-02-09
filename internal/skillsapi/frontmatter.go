package skillsapi

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// parseSkillNameFromFrontmatter extracts the `name` field from a YAML frontmatter
// block in a SKILL.md file.
func parseSkillNameFromFrontmatter(md string) (string, bool) {
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
