package install

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name        string
	Description string
}

func parseSkillFrontmatter(md string) (skillFrontmatter, bool) {
	md = strings.ReplaceAll(md, "\r\n", "\n")
	lines := strings.Split(md, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return skillFrontmatter{}, false
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return skillFrontmatter{}, false
	}
	yamlText := strings.Join(lines[1:end], "\n")

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(yamlText), &fm); err != nil {
		return skillFrontmatter{}, false
	}

	name, ok := fm["name"].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return skillFrontmatter{}, false
	}
	description, ok := fm["description"].(string)
	if !ok || strings.TrimSpace(description) == "" {
		return skillFrontmatter{}, false
	}

	return skillFrontmatter{Name: strings.TrimSpace(name), Description: strings.TrimSpace(description)}, true
}
