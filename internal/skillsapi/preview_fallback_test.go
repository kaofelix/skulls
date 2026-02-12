package skillsapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_FetchSkillMarkdown_FallbackViaGitHubTree(t *testing.T) {
	// Fake GitHub raw host.
	rawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/vercel-labs/agent-skills/HEAD/skills/vercel-composition-patterns/SKILL.md":
			w.WriteHeader(http.StatusNotFound)
			return
		case "/vercel-labs/agent-skills/HEAD/skills/composition-patterns/SKILL.md":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("---\nname: vercel-composition-patterns\ndescription: x\n---\n# Hi\n"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer rawSrv.Close()

	// Fake GitHub API tree.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/repos/vercel-labs/agent-skills/git/trees/HEAD" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Minimal trees response.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tree":[
			{"path":"README.md","type":"blob"},
			{"path":"skills/composition-patterns/SKILL.md","type":"blob"}
		]}`))
	}))
	defer apiSrv.Close()

	c := Client{
		GitHubRawBase: rawSrv.URL,
		GitHubAPIBase: apiSrv.URL,
		HTTP:          &http.Client{Timeout: 2 * time.Second},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	md, err := c.FetchSkillMarkdown(ctx, Skill{SkillID: "vercel-composition-patterns", Source: "vercel-labs/agent-skills"})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if md == "" {
		t.Fatalf("expected markdown")
	}
}

func TestClient_FetchSkillMarkdown_FallbackViaNonStandardPriorityPath(t *testing.T) {
	rawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/owner/repo/HEAD/skills/agent-skill/SKILL.md":
			w.WriteHeader(http.StatusNotFound)
			return
		case "/owner/repo/HEAD/.claude/skills/agent-skill/SKILL.md":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("---\nname: agent-skill\ndescription: x\n---\n# Hi\n"))
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
		_, _ = w.Write([]byte(`{"tree":[
			{"path":".claude/skills/agent-skill/SKILL.md","type":"blob"}
		]}`))
	}))
	defer apiSrv.Close()

	c := Client{GitHubRawBase: rawSrv.URL, GitHubAPIBase: apiSrv.URL, HTTP: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	md, err := c.FetchSkillMarkdown(ctx, Skill{SkillID: "agent-skill", Source: "owner/repo"})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if md == "" {
		t.Fatalf("expected markdown")
	}
}
