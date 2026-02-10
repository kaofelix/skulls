package skillsapi

import (
	"context"
	"encoding/json"
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

type githubTreeResponse struct {
	Truncated bool `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

// FetchSkillMarkdown fetches the raw SKILL.md contents for a skill as best-effort.
//
// GitHub-only. Strategy:
//  1. Fast path: try /skills/<skillID>/SKILL.md
//  2. Fallback: if 404, list repo tree via GitHub API, fetch candidate SKILL.md files,
//     parse frontmatter name, and return the one matching skillID.
func (c Client) FetchSkillMarkdown(ctx context.Context, skill Skill) (string, error) {
	owner, repo, ok := parseGitHubRepo(skill.Source)
	if !ok {
		return "", ErrPreviewUnavailable
	}
	if strings.TrimSpace(skill.SkillID) == "" {
		return "", fmt.Errorf("%w: empty skill id", ErrPreviewUnavailable)
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	rawBase := strings.TrimRight(strings.TrimSpace(c.GitHubRawBase), "/")
	if rawBase == "" {
		rawBase = "https://raw.githubusercontent.com"
	}

	apiBase := strings.TrimRight(strings.TrimSpace(c.GitHubAPIBase), "/")
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}

	// Fast path.
	primaryPath := path.Join("skills", skill.SkillID, "SKILL.md")
	md, status, err := fetchGitHubRaw(ctx, httpClient, rawBase, owner, repo, primaryPath)
	if err == nil {
		return md, nil
	}

	// Only attempt fallback on 404. Other errors should just surface.
	if status != http.StatusNotFound {
		return "", err
	}

	paths, treeErr := fetchGitHubTreeSkillMdPaths(ctx, httpClient, apiBase, owner, repo)
	if treeErr != nil {
		return "", fmt.Errorf("%w: %v", ErrPreviewUnavailable, treeErr)
	}

	for _, p := range paths {
		candidate, _, rawErr := fetchGitHubRaw(ctx, httpClient, rawBase, owner, repo, p)
		if rawErr != nil {
			continue
		}
		name, ok := parseSkillNameFromFrontmatter(candidate)
		if ok && name == skill.SkillID {
			return candidate, nil
		}
	}

	return "", ErrPreviewUnavailable
}

func fetchGitHubRaw(ctx context.Context, httpClient *http.Client, rawBase, owner, repo, relPath string) (string, int, error) {
	u, err := url.Parse(rawBase)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}

	// /<owner>/<repo>/HEAD/<relPath>
	u.Path = path.Join(u.Path, owner, repo, "HEAD", relPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", resp.StatusCode, fmt.Errorf("%w: %s", ErrPreviewUnavailable, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("%w: %v", ErrPreviewUnavailable, err)
	}
	return string(b), resp.StatusCode, nil
}

func fetchGitHubTreeSkillMdPaths(ctx context.Context, httpClient *http.Client, apiBase, owner, repo string) ([]string, error) {
	u, err := url.Parse(apiBase)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "repos", owner, repo, "git", "trees", "HEAD")
	q := u.Query()
	q.Set("recursive", "1")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tree listing failed: %s", resp.Status)
	}

	var decoded githubTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	paths := make([]string, 0, 32)
	for _, it := range decoded.Tree {
		if it.Type != "blob" {
			continue
		}
		if !strings.HasSuffix(it.Path, "SKILL.md") {
			continue
		}
		// Prefer skills/* and root SKILL.md.
		if it.Path == "SKILL.md" || strings.HasPrefix(it.Path, "skills/") {
			paths = append(paths, it.Path)
		}
	}
	return paths, nil
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
