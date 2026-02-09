package skillsapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name   string
		source string
		ok     bool
		owner  string
		repo   string
	}{
		{"shorthand", "obra/superpowers", true, "obra", "superpowers"},
		{"https", "https://github.com/obra/superpowers", true, "obra", "superpowers"},
		{"https_dot_git", "https://github.com/obra/superpowers.git", true, "obra", "superpowers"},
		{"ssh", "git@github.com:obra/superpowers.git", true, "obra", "superpowers"},
		{"non_github", "https://gitlab.com/obra/superpowers", false, "", ""},
		{"weird", "", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, ok := parseGitHubRepo(tt.source)
			if ok != tt.ok {
				t.Fatalf("ok: got %v want %v", ok, tt.ok)
			}
			if owner != tt.owner {
				t.Fatalf("owner: got %q want %q", owner, tt.owner)
			}
			if repo != tt.repo {
				t.Fatalf("repo: got %q want %q", repo, tt.repo)
			}
		})
	}
}

func TestClient_FetchSkillMarkdown_GitHubRawHEAD(t *testing.T) {
	// Fake raw.githubusercontent.com.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/obra/superpowers/HEAD/skills/using-git-worktrees/SKILL.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Hello\n"))
	}))
	defer srv.Close()

	c := Client{GitHubRawBase: srv.URL, HTTP: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	md, err := c.FetchSkillMarkdown(ctx, Skill{SkillID: "using-git-worktrees", Source: "obra/superpowers"})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if md != "# Hello\n" {
		t.Fatalf("unexpected md: %q", md)
	}
}

func TestClient_FetchSkillMarkdown_UnsupportedSource(t *testing.T) {
	c := Client{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.FetchSkillMarkdown(ctx, Skill{SkillID: "x", Source: "https://example.com/foo/bar"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrPreviewUnavailable) {
		t.Fatalf("expected ErrPreviewUnavailable, got %v", err)
	}
}
