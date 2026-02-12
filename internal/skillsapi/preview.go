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
	"sort"
	"strings"
	"time"

	"github.com/kaofelix/skulls/internal/skilllayout"
)

// ErrPreviewUnavailable is returned (possibly wrapped) when a SKILL.md preview
// can't be fetched (unsupported source, missing file, network error, etc.).
var ErrPreviewUnavailable = errors.New("preview unavailable")

var shorthandRepoRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

const maxPreviewCandidates = 250

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
//  2. Fallback: on 404, list repo tree via GitHub API, rank SKILL.md candidates,
//     fetch candidates, parse strict frontmatter, and return matching name.
func (c Client) FetchSkillMarkdown(ctx context.Context, skill Skill) (string, error) {
	owner, repo, ok := parseGitHubRepo(skill.Source)
	if !ok {
		return "", ErrPreviewUnavailable
	}
	skillID := strings.TrimSpace(skill.SkillID)
	if skillID == "" {
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

	primaryPath := path.Join("skills", skillID, "SKILL.md")
	md, status, err := fetchGitHubRaw(ctx, httpClient, rawBase, owner, repo, primaryPath)
	if err == nil {
		return md, nil
	}
	if status != http.StatusNotFound {
		return "", err
	}

	paths, treeErr := fetchGitHubTreeSkillMdPaths(ctx, httpClient, apiBase, owner, repo, skillID)
	if treeErr != nil {
		return "", fmt.Errorf("%w: %w", ErrPreviewUnavailable, treeErr)
	}

	for _, p := range paths {
		candidate, _, rawErr := fetchGitHubRaw(ctx, httpClient, rawBase, owner, repo, p)
		if rawErr != nil {
			continue
		}
		name, ok := parseSkillNameFromFrontmatter(candidate)
		if ok && name == skillID {
			return candidate, nil
		}
	}

	return "", ErrPreviewUnavailable
}

func fetchGitHubRaw(ctx context.Context, httpClient *http.Client, rawBase, owner, repo, relPath string) (string, int, error) {
	u, err := url.Parse(rawBase)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %w", ErrPreviewUnavailable, err)
	}

	u.Path = path.Join(u.Path, owner, repo, "HEAD", relPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %w", ErrPreviewUnavailable, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %w", ErrPreviewUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", resp.StatusCode, fmt.Errorf("%w: %s", ErrPreviewUnavailable, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, fmt.Errorf("%w: %w", ErrPreviewUnavailable, err)
	}
	return string(b), resp.StatusCode, nil
}

func fetchGitHubTreeSkillMdPaths(ctx context.Context, httpClient *http.Client, apiBase, owner, repo, skillID string) ([]string, error) {
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

	all := make([]string, 0, 64)
	for _, it := range decoded.Tree {
		if it.Type != "blob" {
			continue
		}
		if strings.EqualFold(path.Base(it.Path), "SKILL.md") {
			all = append(all, it.Path)
		}
	}

	skillID = strings.TrimSpace(skillID)
	exact := make([]string, 0, len(all))
	priority := make([]string, 0, len(all))
	others := make([]string, 0, len(all))

	exactSuffix := "/" + skillID + "/SKILL.md"
	for _, p := range all {
		if skillID != "" && (p == path.Join(skillID, "SKILL.md") || strings.HasSuffix(p, exactSuffix)) {
			exact = append(exact, p)
			continue
		}
		if skilllayout.IsPrioritySkillPath(p) {
			priority = append(priority, p)
			continue
		}
		others = append(others, p)
	}

	sort.Strings(exact)
	sort.Strings(priority)
	sort.Strings(others)

	combined := make([]string, 0, len(all))
	combined = append(combined, exact...)
	combined = append(combined, priority...)
	combined = append(combined, others...)
	if len(combined) > maxPreviewCandidates {
		combined = combined[:maxPreviewCandidates]
	}
	return combined, nil
}

func parseGitHubRepo(source string) (owner, repo string, ok bool) {
	s := strings.TrimSpace(source)
	if s == "" {
		return "", "", false
	}

	if shorthandRepoRe.MatchString(s) {
		parts := strings.SplitN(s, "/", 2)
		return parts[0], parts[1], true
	}

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
	repo = strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}
