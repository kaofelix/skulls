package skillsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Skill struct {
	ID       string `json:"id"`
	SkillID  string `json:"skillId"`
	Name     string `json:"name"`
	Installs int    `json:"installs"`
	Source   string `json:"source"`
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type searchResponse struct {
	Skills []Skill `json:"skills"`
}

func (c Client) Search(ctx context.Context, query string, limit int) ([]Skill, error) {
	base := strings.TrimSpace(c.BaseURL)
	if base == "" {
		base = "https://skills.sh"
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/search"
	q := u.Query()
	q.Set("q", query)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search failed: %s", resp.Status)
	}

	var decoded searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded.Skills, nil
}
