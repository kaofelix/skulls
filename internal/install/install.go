package install

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaofelix/skulls/internal/fsutil"
	"github.com/kaofelix/skulls/internal/gitutil"
)

type Step string

const (
	StepNormalize Step = "normalize"
	StepClone     Step = "clone"
	StepVerify    Step = "verify"
	StepRemove    Step = "remove"
	StepCopy      Step = "copy"
)

type Event struct {
	Step    Step
	Message string
	Done    bool
}

type ProgressFunc func(Event)

type Options struct {
	TargetDir string
	Force     bool

	// Progress, if set, is called as the installer advances.
	Progress ProgressFunc

	// GitStdout/GitStderr control where `git clone` output goes.
	// If nil, defaults to os.Stdout/os.Stderr.
	GitStdout io.Writer
	GitStderr io.Writer
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

	if opts.Progress != nil {
		opts.Progress(Event{Step: StepNormalize, Message: "Normalizing source…"})
	}
	cloneURL, err := gitutil.NormalizeSourceToGitURL(source)
	if err != nil {
		return "", err
	}
	if opts.Progress != nil {
		opts.Progress(Event{Step: StepNormalize, Message: "Normalizing source…", Done: true})
	}

	stdout := opts.GitStdout
	stderr := opts.GitStderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	if opts.Progress != nil {
		opts.Progress(Event{Step: StepClone, Message: "Downloading repository…"})
	}
	if err := gitutil.CloneShallowTo(cloneURL, repoDir, stdout, stderr); err != nil {
		return "", err
	}
	if opts.Progress != nil {
		opts.Progress(Event{Step: StepClone, Message: "Downloading repository…", Done: true})
	}

	if opts.Progress != nil {
		opts.Progress(Event{Step: StepVerify, Message: "Verifying skill layout…"})
	}

	skillDir, err := resolveSkillDir(repoDir, skillID)
	if err != nil {
		return "", err
	}

	if opts.Progress != nil {
		opts.Progress(Event{Step: StepVerify, Message: "Verifying skill layout…", Done: true})
	}

	folderName := sanitizeName(skillID)
	installPath := filepath.Join(targetBase, folderName)

	if _, statErr := os.Stat(installPath); statErr == nil {
		if !opts.Force {
			return "", fmt.Errorf("target already exists: %s (use --force to overwrite)", installPath)
		}
		if opts.Progress != nil {
			opts.Progress(Event{Step: StepRemove, Message: "Removing existing installation…"})
		}
		if err := os.RemoveAll(installPath); err != nil {
			return "", err
		}
		if opts.Progress != nil {
			opts.Progress(Event{Step: StepRemove, Message: "Removing existing installation…", Done: true})
		}
	}

	if opts.Progress != nil {
		opts.Progress(Event{Step: StepCopy, Message: "Installing skill files…"})
	}
	if err := fsutil.CopyDir(skillDir, installPath); err != nil {
		return "", err
	}
	if opts.Progress != nil {
		opts.Progress(Event{Step: StepCopy, Message: "Installing skill files…", Done: true})
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
