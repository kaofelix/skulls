package skillsapi

import (
	"context"
	"net/http"
	"path"
	"strings"
	"time"
)

const gitHubWebBase = "https://github.com"

// BrowserURLForSkill returns the best known browser URL for a skill's SKILL.md.
//
// Resolution strategy:
//   - exact default path when it exists
//   - matching path discovered from the GitHub tree
//   - closest known discovered SKILL.md path
//   - repository root as the final known fallback
func (c Client) BrowserURLForSkill(ctx context.Context, skill Skill) (string, error) {
	owner, repo, ok := parseGitHubRepo(skill.Source)
	if !ok {
		return "", ErrPreviewUnavailable
	}

	skillID := strings.TrimSpace(skill.SkillID)
	if skillID == "" {
		return repoWebURL(owner, repo), nil
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
	exists, status := githubRawPathExists(ctx, httpClient, rawBase, owner, repo, primaryPath)
	if exists {
		return repoBlobURL(owner, repo, primaryPath), nil
	}
	if status != http.StatusNotFound {
		return repoWebURL(owner, repo), nil
	}

	var paths []string
	if fetchedPaths, err := fetchGitHubTreeSkillMdPaths(ctx, httpClient, apiBase, owner, repo, skillID); err == nil {
		paths = fetchedPaths
	}
	if len(paths) == 0 {
		return repoWebURL(owner, repo), nil
	}

	for _, p := range paths {
		candidate, _, rawErr := fetchGitHubRaw(ctx, httpClient, rawBase, owner, repo, p)
		if rawErr != nil {
			continue
		}
		name, ok := parseSkillNameFromFrontmatter(candidate)
		if ok && name == skillID {
			return repoBlobURL(owner, repo, p), nil
		}
	}

	return repoBlobURL(owner, repo, paths[0]), nil
}

func githubRawPathExists(ctx context.Context, httpClient *http.Client, rawBase, owner, repo, relPath string) (bool, int) {
	_, status, err := fetchGitHubRaw(ctx, httpClient, rawBase, owner, repo, relPath)
	return err == nil, status
}

func repoWebURL(owner, repo string) string {
	return gitHubWebBase + "/" + owner + "/" + repo
}

func repoBlobURL(owner, repo, relPath string) string {
	return repoWebURL(owner, repo) + "/blob/HEAD/" + strings.TrimLeft(relPath, "/")
}
