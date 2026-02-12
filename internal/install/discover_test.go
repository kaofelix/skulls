package install

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDiscoverSkills_RootSkillEarlyReturnByDefault(t *testing.T) {
	repo := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("SKILL.md", "---\nname: root-skill\ndescription: root\n---\n")
	mustWrite("skills/nested/SKILL.md", "---\nname: nested-skill\ndescription: nested\n---\n")

	skills, err := discoverSkillsInRepo(repo, discoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("got %d skills", len(skills))
	}
	if skills[0].Name != "root-skill" {
		t.Fatalf("first skill=%q", skills[0].Name)
	}
}

func TestDiscoverSkills_FullDepthIncludesRootAndNested(t *testing.T) {
	repo := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("SKILL.md", "---\nname: root-skill\ndescription: root\n---\n")
	mustWrite("skills/nested/SKILL.md", "---\nname: nested-skill\ndescription: nested\n---\n")

	skills, err := discoverSkillsInRepo(repo, discoverOptions{FullDepth: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 {
		t.Fatalf("got %d skills", len(skills))
	}
	names := []string{skills[0].Name, skills[1].Name}
	sort.Strings(names)
	if names[0] != "nested-skill" || names[1] != "root-skill" {
		t.Fatalf("names=%v", names)
	}
}

func TestDiscoverSkills_FindsPriorityAgentDirs(t *testing.T) {
	repo := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite(".claude/skills/alpha/SKILL.md", "---\nname: alpha\ndescription: a\n---\n")

	skills, err := discoverSkillsInRepo(repo, discoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "alpha" {
		t.Fatalf("skills=%+v", skills)
	}
}

func TestDiscoverSkills_RecursiveFallbackFindsNonStandardLayout(t *testing.T) {
	repo := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("custom/catalog/my-skill/SKILL.md", "---\nname: my-skill\ndescription: d\n---\n")

	skills, err := discoverSkillsInRepo(repo, discoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "my-skill" {
		t.Fatalf("skills=%+v", skills)
	}
}

func TestDiscoverSkills_StrictFrontmatterRequiresDescription(t *testing.T) {
	repo := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("skills/one/SKILL.md", "---\nname: one\n---\n")

	_, err := discoverSkillsInRepo(repo, discoverOptions{})
	if err == nil {
		t.Fatalf("expected no skills found error")
	}
}
