package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSkills_LocalPathFindsSkillsAndSkillFiles(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

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

	mustWrite("skills/alpha/SKILL.md", "---\nname: alpha\ndescription: a\n---\n")
	mustWrite("skills/nested/beta/SKILL.md", "---\nname: beta\ndescription: b\n---\n")

	skills, cleanup, err := DiscoverSkills(repo)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatal(err)
	}

	if len(skills) != 2 {
		t.Fatalf("got %d skills", len(skills))
	}
	if skills[0].Name != "alpha" {
		t.Fatalf("first skill=%q", skills[0].Name)
	}
	if filepath.Base(skills[0].SkillFilePath) != "SKILL.md" {
		t.Fatalf("expected SKILL.md path, got %q", skills[0].SkillFilePath)
	}
	if skills[1].Name != "beta" {
		t.Fatalf("second skill=%q", skills[1].Name)
	}
}

func TestDiscoverSkills_DuplicateFrontmatterNames_Error(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

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

	mustWrite("skills/one/SKILL.md", "---\nname: dup\n---\n")
	mustWrite("skills/two/SKILL.md", "---\nname: dup\n---\n")

	_, cleanup, err := DiscoverSkills(repo)
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Fatalf("expected duplicate name error")
	}
}
