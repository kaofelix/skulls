package skillsapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

// ErrPreviewUnavailable is returned (possibly wrapped) when a SKILL.md preview
// can't be fetched (unsupported source, missing file, network error, etc.).
var ErrPreviewUnavailable = errors.New("preview unavailable")

var shorthandRepoRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// FetchSkillMarkdown fetches the raw SKILL.md contents for a skill as best-effort.
//
// Currently GitHub-only. For GitHub repos, it uses raw.githubusercontent.com with
// the special ref "HEAD" to resolve the default branch.
func (c Client) FetchSkillMarkdown(ctx context.Context, skill Skill) (string, error) {
	owner, repo, ok := parseGitHubRepo(skill.Source)
	if !ok {
		return "", ErrPreviewUnavailable
	}
	if strings.TrimSpace(skill.SkillID) == "" {
		return "", fmt.Errorf("%w: empty skill id", ErrPreviewUnavailable)
	}

	rawBase := strings.TrimRight(strings.TrimSpace(c.GitHubRawBase), "/")
	if rawBase == "" {
		rawBase = "https://raw.githubusercontent.com"
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	u, err := url.Parse(rawBase)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}

	// /<owner>/<repo>/HEAD/skills/<skill-id>/SKILL.md
	u.Path = path.Join(u.Path, owner, repo, "HEAD", "skills", skill.SkillID, "SKILL.md")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: %s", ErrPreviewUnavailable, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}
	return string(b), nil
}

func parseGitHubRepo(source string) (owner, repo string, ok bool) {
	s := strings.TrimSpace(source)
	if s == "" {
		return "", "", false
	}

	// owner/repo shorthand
	if shorthandRepoRe.MatchString(s) {
		parts := strings.SplitN(s, "/", 2)
		return parts[0], parts[1], true
	}

	// SSH form: git@github.com:owner/repo(.git)
	if strings.HasPrefix(s, "git@github.com:") {
		p := strings.TrimPrefix(s, "git@github.com:")
		p = strings.TrimSuffix(p, ".git")
		p = strings.Trim(p, "/")
		parts := strings.Split(p, "/")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], true
		}
		return "", "", false
	}

	// URL-ish forms.
	if strings.HasPrefix(s, "github.com/") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", "", false
	}

	host := strings.ToLower(u.Host)
	if host != "github.com" && host != "www.github.com" {
		return "", "", false
	}

	p := strings.Trim(u.Path, "/")
	if p == "" {
		return "", "", false
	}
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return "", "", false
	}

	owner = parts[0]
	repo = parts[1]
	repo = strings.TrimSuffix(repo, ".git")
	if owner == "" || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}
