package skilllayout

import "strings"

// PrioritySearchDirs mirrors the discovery order used by vercel-labs/skills.
// Entries are repository-relative directories that contain per-skill folders.
var PrioritySearchDirs = []string{
	"skills",
	"skills/.curated",
	"skills/.experimental",
	"skills/.system",
	".agent/skills",
	".agents/skills",
	".claude/skills",
	".cline/skills",
	".codebuddy/skills",
	".codex/skills",
	".commandcode/skills",
	".continue/skills",
	".cursor/skills",
	".github/skills",
	".goose/skills",
	".iflow/skills",
	".junie/skills",
	".kilocode/skills",
	".kiro/skills",
	".mux/skills",
	".neovate/skills",
	".opencode/skills",
	".openhands/skills",
	".pi/skills",
	".qoder/skills",
	".roo/skills",
	".trae/skills",
	".windsurf/skills",
	".zencoder/skills",
}

const MaxRecursiveDepth = 5

var skipDirs = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	"dist":         {},
	"build":        {},
	"__pycache__":  {},
}

func ShouldSkipDir(name string) bool {
	_, ok := skipDirs[name]
	return ok
}

func IsPrioritySkillPath(relPath string) bool {
	rel := strings.Trim(strings.TrimSpace(relPath), "/")
	if rel == "SKILL.md" {
		return true
	}
	for _, root := range PrioritySearchDirs {
		prefix := strings.Trim(root, "/") + "/"
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}
