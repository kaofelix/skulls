package skillsapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Popular_ParsesInitialSkillsAndSorts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><body>
<script>self.__next_f.push([1,"something ... \"initialSkills\":[{\"source\":\"a/b\",\"skillId\":\"one\",\"name\":\"one\",\"installs\":2},{\"source\":\"c/d\",\"skillId\":\"two\",\"name\":\"two\",\"installs\":5}],\"totalSkills\":123 ..."])</script>
</body></html>`))
	}))
	defer srv.Close()

	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	skills, err := c.Popular(ctx, 10)
	if err != nil {
		t.Fatalf("Popular returned error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
	// sorted desc by installs
	if skills[0].SkillID != "two" || skills[0].Installs != 5 {
		t.Fatalf("unexpected first skill: %#v", skills[0])
	}
	if skills[1].SkillID != "one" || skills[1].Installs != 2 {
		t.Fatalf("unexpected second skill: %#v", skills[1])
	}
}

func TestClient_Popular_Limit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`\"initialSkills\":[{\"source\":\"a/b\",\"skillId\":\"one\",\"name\":\"one\",\"installs\":2},{\"source\":\"c/d\",\"skillId\":\"two\",\"name\":\"two\",\"installs\":5}],\"totalSkills\":123`))
	}))
	defer srv.Close()

	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	skills, err := c.Popular(ctx, 1)
	if err != nil {
		t.Fatalf("Popular returned error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].SkillID != "two" {
		t.Fatalf("unexpected skill: %#v", skills[0])
	}
}

func TestClient_Popular_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`no initial skills here`))
	}))
	defer srv.Close()

	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.Popular(ctx, 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}
