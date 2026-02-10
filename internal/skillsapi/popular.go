package skillsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var errInitialSkillsNotFound = errors.New("initialSkills not found")

// Popular fetches a list of popular skills from the skills.sh homepage.
//
// Note: skills.sh doesn't currently expose a public "popular" API. The homepage
// embeds an `initialSkills` list inside an escaped payload; we parse it.
func (c Client) Popular(ctx context.Context, limit int) ([]Skill, error) {
	base := strings.TrimSpace(c.BaseURL)
	if base == "" {
		base = "https://skills.sh"
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("popular fetch failed: %s", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	skills, err := parseInitialSkills(string(b))
	if err != nil {
		return nil, err
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Installs > skills[j].Installs
	})

	if limit > 0 && len(skills) > limit {
		skills = skills[:limit]
	}
	return skills, nil
}

var initialSkillsRe = regexp.MustCompile(`(?s)\\+"initialSkills\\+"\s*:\s*\[(.*?)\]\s*,\s*\\+"totalSkills\\+"\s*:`)

func parseInitialSkills(html string) ([]Skill, error) {
	m := initialSkillsRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return nil, errInitialSkillsNotFound
	}

	// m[1] is the JSON array content but with escaped quotes (e.g. {\"a\":1}).
	// Wrap it as a quoted string and unescape it.
	unescaped, err := strconv.Unquote("\"" + m[1] + "\"")
	if err != nil {
		return nil, err
	}

	payload := "[" + unescaped + "]"
	var skills []Skill
	if err := json.Unmarshal([]byte(payload), &skills); err != nil {
		return nil, err
	}
	return skills, nil
}
