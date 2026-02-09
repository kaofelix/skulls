package skillsapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Search_BuildsURLAndParsesResults(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotLimit string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("q")
		gotLimit = r.URL.Query().Get("limit")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"query": "git",
			"skills": [
				{"id":"owner/repo/s1","skillId":"s1","name":"s1","installs":12,"source":"owner/repo"}
			]
		}`))
	}))
	defer srv.Close()

	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	skills, err := c.Search(ctx, "git", 10)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if gotPath != "/api/search" {
		t.Fatalf("expected path /api/search, got %q", gotPath)
	}
	if gotQuery != "git" {
		t.Fatalf("expected q=git, got %q", gotQuery)
	}
	if gotLimit != "10" {
		t.Fatalf("expected limit=10, got %q", gotLimit)
	}

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].SkillID != "s1" || skills[0].Source != "owner/repo" || skills[0].Installs != 12 {
		t.Fatalf("unexpected skill: %#v", skills[0])
	}
}

func TestClient_Search_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	c := Client{BaseURL: srv.URL, HTTP: srv.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.Search(ctx, "git", 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}
