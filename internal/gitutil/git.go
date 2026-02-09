package gitutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NormalizeSourceToGitURL accepts:
// - owner/repo (GitHub shorthand)
// - any URL-like git remote (https://..., git@..., file:///...)
// - local path to a git repo
func NormalizeSourceToGitURL(source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", errors.New("source is required")
	}

	// URL-ish
	if strings.Contains(source, "://") || strings.HasPrefix(source, "git@") {
		return source, nil
	}

	// Local path
	if looksLikePath(source) {
		abs, err := filepath.Abs(source)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}

	// owner/repo shorthand
	parts := strings.Split(source, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1]), nil
	}

	return "", fmt.Errorf("unsupported source format: %q", source)
}

func looksLikePath(s string) bool {
	return strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "/") || s == "." || s == ".."
}

func CloneShallow(url string, dest string) error {
	return CloneShallowTo(url, dest, os.Stdout, os.Stderr)
}

func CloneShallowTo(url string, dest string, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.Command("git", "clone", "--depth", "1", url, dest)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
