package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInstallSkill_FromLocalRepo(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "skills", "hello-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "skills", "hello-skill", "SKILL.md"), []byte("---\nname: hello-skill\ndescription: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, string(out))
		}
	}

	run("git", "init")
	run("git", "add", ".")
	run("git", "commit", "-m", "init")

	target := filepath.Join(tmp, "target")
	installed, err := InstallSkill(repo, "hello-skill", Options{TargetDir: target})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(installed, "SKILL.md")); err != nil {
		t.Fatalf("expected SKILL.md to exist: %v", err)
	}
}

func TestInstallSkill_FallbackByFrontmatterName(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

	// Folder name doesn't match skill id, but SKILL.md frontmatter name does.
	if err := os.MkdirAll(filepath.Join(repo, "skills", "composition-patterns"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "skills", "composition-patterns", "SKILL.md"), []byte("---\nname: vercel-composition-patterns\ndescription: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "skills", "composition-patterns", "extra.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, string(out))
		}
	}

	run("git", "init")
	run("git", "add", ".")
	run("git", "commit", "-m", "init")

	target := filepath.Join(tmp, "target")
	installed, err := InstallSkill(repo, "vercel-composition-patterns", Options{TargetDir: target})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(installed, "SKILL.md")); err != nil {
		t.Fatalf("expected SKILL.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installed, "extra.txt")); err != nil {
		t.Fatalf("expected extra.txt to exist: %v", err)
	}
}

func TestInstallSkill_FromPriorityAgentDir(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

	if err := os.MkdirAll(filepath.Join(repo, ".claude", "skills", "agent-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".claude", "skills", "agent-skill", "SKILL.md"), []byte("---\nname: agent-skill\ndescription: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".claude", "skills", "agent-skill", "extra.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, string(out))
		}
	}

	run("git", "init")
	run("git", "add", ".")
	run("git", "commit", "-m", "init")

	target := filepath.Join(tmp, "target")
	installed, err := InstallSkill(repo, "agent-skill", Options{TargetDir: target})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(installed, "SKILL.md")); err != nil {
		t.Fatalf("expected SKILL.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installed, "extra.txt")); err != nil {
		t.Fatalf("expected extra.txt to exist: %v", err)
	}
}
