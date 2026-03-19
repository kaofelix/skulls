package skillsapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_BrowserURLForSkill_GitHubPrimaryPath(t *testing.T) {
	rawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/obra/superpowers/HEAD/skills/using-git-worktrees/SKILL.md" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Hello\n"))
	}))
	defer rawSrv.Close()

	c := Client{GitHubRawBase: rawSrv.URL, HTTP: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := c.BrowserURLForSkill(ctx, Skill{SkillID: "using-git-worktrees", Source: "obra/superpowers"})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}

	want := "https://github.com/obra/superpowers/blob/HEAD/skills/using-git-worktrees/SKILL.md"
	if got != want {
		t.Fatalf("url: got %q want %q", got, want)
	}
}

func TestClient_BrowserURLForSkill_FallsBackToClosestKnownPath(t *testing.T) {
	rawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/owner/repo/HEAD/skills/test-skill/SKILL.md":
			w.WriteHeader(http.StatusNotFound)
			return
		case "/owner/repo/HEAD/custom/catalog/containing-folder/SKILL.md":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("---\nname: some-other-skill\ndescription: x\n---\n# Hi\n"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer rawSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/git/trees/HEAD" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tree":[{"path":"custom/catalog/containing-folder/SKILL.md","type":"blob"}]}`))
	}))
	defer apiSrv.Close()

	c := Client{GitHubRawBase: rawSrv.URL, GitHubAPIBase: apiSrv.URL, HTTP: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := c.BrowserURLForSkill(ctx, Skill{SkillID: "test-skill", Source: "owner/repo"})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}

	want := "https://github.com/owner/repo/blob/HEAD/custom/catalog/containing-folder/SKILL.md"
	if got != want {
		t.Fatalf("url: got %q want %q", got, want)
	}
}

func TestClient_BrowserURLForSkill_FallsBackToRepoRootWhenNoKnownSkillPath(t *testing.T) {
	rawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer rawSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer apiSrv.Close()

	c := Client{GitHubRawBase: rawSrv.URL, GitHubAPIBase: apiSrv.URL, HTTP: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := c.BrowserURLForSkill(ctx, Skill{SkillID: "test-skill", Source: "owner/repo"})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}

	want := "https://github.com/owner/repo"
	if got != want {
		t.Fatalf("url: got %q want %q", got, want)
	}
}
