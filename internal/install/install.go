package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaofelix/skulls/internal/fsutil"
	"github.com/kaofelix/skulls/internal/gitutil"
)

type Options struct {
	TargetDir string
	Force     bool
}

func InstallSkill(source string, skillID string, opts Options) (string, error) {
	skillID = strings.TrimSpace(skillID)
	if skillID == "" {
		return "", errors.New("skill-id is required")
	}

	targetBase, err := filepath.Abs(expandHome(opts.TargetDir))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(targetBase, 0o755); err != nil {
		return "", err
	}

	// Clone to temp.
	tmp, err := os.MkdirTemp("", "skulls-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	repoDir := filepath.Join(tmp, "repo")
	cloneURL, err := gitutil.NormalizeSourceToGitURL(source)
	if err != nil {
		return "", err
	}
	if err := gitutil.CloneShallow(cloneURL, repoDir); err != nil {
		return "", err
	}

	// For now we follow the skills.sh convention: skills/<skillID>/SKILL.md
	skillDir := filepath.Join(repoDir, "skills", skillID)
	if fi, err := os.Stat(skillDir); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("skill directory not found in repo: %s", filepath.ToSlash(filepath.Join("skills", skillID)))
	}

	skillMd := filepath.Join(skillDir, "SKILL.md")
	if fi, err := os.Stat(skillMd); err != nil || fi.IsDir() {
		return "", fmt.Errorf("SKILL.md not found at %s", filepath.ToSlash(filepath.Join("skills", skillID, "SKILL.md")))
	}

	folderName := sanitizeName(skillID)
	installPath := filepath.Join(targetBase, folderName)

	if _, err := os.Stat(installPath); err == nil {
		if !opts.Force {
			return "", fmt.Errorf("target already exists: %s (use --force to overwrite)", installPath)
		}
		if err := os.RemoveAll(installPath); err != nil {
			return "", err
		}
	}

	if err := fsutil.CopyDir(skillDir, installPath); err != nil {
		return "", err
	}

	return installPath, nil
}

func expandHome(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		h, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(h, p[2:])
	}
	return p
}

// sanitizeName makes a safe directory name.
// Matches the spirit of vercel-labs/skills: kebab-ish, no traversal.
func sanitizeName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, string(filepath.Separator), "-")

	// Keep [a-z0-9._-], replace everything else with '-'
	b := strings.Builder{}
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), ".-")
	if out == "" {
		out = "unnamed-skill"
	}
	if len(out) > 255 {
		out = out[:255]
	}
	return out
}
