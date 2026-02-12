package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdoutStderr(t *testing.T) (*bytes.Buffer, *bytes.Buffer, func()) {
	t.Helper()

	oldOut := os.Stdout
	oldErr := os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stdout = outW
	os.Stderr = errW

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(outBuf, outR)
		close(done)
	}()
	doneErr := make(chan struct{})
	go func() {
		_, _ = io.Copy(errBuf, errR)
		close(doneErr)
	}()

	return outBuf, errBuf, func() {
		_ = outW.Close()
		_ = errW.Close()
		<-done
		<-doneErr
		os.Stdout = oldOut
		os.Stderr = oldErr
	}
}

func useTestConfigPath(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	p := filepath.Join(tmp, "config.json")
	orig := configPathFunc
	configPathFunc = func() (string, error) { return p, nil }
	t.Cleanup(func() { configPathFunc = orig })
}

func TestRunAdd_WhenSkillIDOmitted_UsesSelectorAndInstallUIWithOverwrite(t *testing.T) {
	origSelect := runAddSelectFromSource
	origInstallUI := runAddInstallUI
	t.Cleanup(func() {
		runAddSelectFromSource = origSelect
		runAddInstallUI = origInstallUI
	})

	runAddSelectFromSource = func(source string) (tuiSearchResult, error) {
		if source != "owner/repo" {
			t.Fatalf("selector source=%q", source)
		}
		return tuiSearchResult{Selected: true, Skill: tuiSkill{Source: source, SkillID: "chosen-skill"}}, nil
	}

	var gotTarget string
	var gotForce bool
	var gotSkill tuiSkill
	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		gotTarget = targetDir
		gotForce = force
		gotSkill = skill
		return tuiInstallResult{InstalledPath: "/tmp/installed"}, nil
	}

	outBuf, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "owner/repo", "--dir", "/tmp/skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%s", exit, errBuf.String())
	}
	if gotTarget != "/tmp/skills" {
		t.Fatalf("target=%q", gotTarget)
	}
	if !gotForce {
		t.Fatalf("expected overwrite behavior (force=true)")
	}
	if gotSkill.Source != "owner/repo" || gotSkill.SkillID != "chosen-skill" {
		t.Fatalf("skill=%+v", gotSkill)
	}
	if !strings.Contains(outBuf.String(), "Installed chosen-skill") {
		t.Fatalf("unexpected stdout: %q", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Source: owner/repo") {
		t.Fatalf("unexpected stdout: %q", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "Path: /tmp/installed") {
		t.Fatalf("unexpected stdout: %q", outBuf.String())
	}
}

func TestRunAdd_WhenSelectorCancelled_DoesNotInstall(t *testing.T) {
	origSelect := runAddSelectFromSource
	origInstallUI := runAddInstallUI
	t.Cleanup(func() {
		runAddSelectFromSource = origSelect
		runAddInstallUI = origInstallUI
	})

	runAddSelectFromSource = func(source string) (tuiSearchResult, error) {
		return tuiSearchResult{Selected: false}, nil
	}

	calledInstall := false
	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		calledInstall = true
		return tuiInstallResult{}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "owner/repo", "--dir", "/tmp/skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%s", exit, errBuf.String())
	}
	if calledInstall {
		t.Fatalf("install should not have been called")
	}
}

func TestRunAdd_WhenDirMissing_UsesConfiguredDir(t *testing.T) {
	useTestConfigPath(t)

	if err := setInstallDir("/tmp/saved-skills"); err != nil {
		t.Fatal(err)
	}

	origSelect := runAddSelectFromSource
	origInstallUI := runAddInstallUI
	t.Cleanup(func() {
		runAddSelectFromSource = origSelect
		runAddInstallUI = origInstallUI
	})

	runAddSelectFromSource = func(source string) (tuiSearchResult, error) {
		return tuiSearchResult{Selected: true, Skill: tuiSkill{Source: source, SkillID: "chosen-skill"}}, nil
	}

	gotTarget := ""
	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		gotTarget = targetDir
		return tuiInstallResult{InstalledPath: "/tmp/installed"}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "owner/repo"})
	restore()

	if exit != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", exit, errBuf.String())
	}
	if gotTarget != "/tmp/saved-skills" {
		t.Fatalf("target=%q", gotTarget)
	}
}

func TestRunAdd_WhenSourceUsesAtShorthand_InstallsDirectlyWithoutSelector(t *testing.T) {
	origSelect := runAddSelectFromSource
	origInstallUI := runAddInstallUI
	t.Cleanup(func() {
		runAddSelectFromSource = origSelect
		runAddInstallUI = origInstallUI
	})

	calledSelect := false
	runAddSelectFromSource = func(source string) (tuiSearchResult, error) {
		calledSelect = true
		return tuiSearchResult{}, nil
	}

	var gotSkill tuiSkill
	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		gotSkill = skill
		return tuiInstallResult{InstalledPath: "/tmp/installed"}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "owner/repo@test-driven-development", "--dir", "/tmp/skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%s", exit, errBuf.String())
	}
	if calledSelect {
		t.Fatalf("selector should not have been called")
	}
	if gotSkill.Source != "owner/repo" {
		t.Fatalf("got source=%q", gotSkill.Source)
	}
	if gotSkill.SkillID != "test-driven-development" {
		t.Fatalf("got skill=%q", gotSkill.SkillID)
	}
}

func TestRunAdd_WhenSourceIsGitSSH_DoesNotUseAtShorthand(t *testing.T) {
	origSelect := runAddSelectFromSource
	origInstallUI := runAddInstallUI
	t.Cleanup(func() {
		runAddSelectFromSource = origSelect
		runAddInstallUI = origInstallUI
	})

	calledSelect := false
	runAddSelectFromSource = func(source string) (tuiSearchResult, error) {
		calledSelect = true
		if source != "git@github.com:owner/repo" {
			t.Fatalf("selector source=%q", source)
		}
		return tuiSearchResult{Selected: false}, nil
	}

	calledInstall := false
	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		calledInstall = true
		return tuiInstallResult{}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "git@github.com:owner/repo", "--dir", "/tmp/skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%s", exit, errBuf.String())
	}
	if !calledSelect {
		t.Fatalf("selector should have been called")
	}
	if calledInstall {
		t.Fatalf("install should not be called when selector cancels")
	}
}

func TestRunAdd_WhenInstallUICannotOpenTTY_FallsBackToPlainInstall(t *testing.T) {
	origInstallUI := runAddInstallUI
	origInstallPlain := runAddInstallPlain
	t.Cleanup(func() {
		runAddInstallUI = origInstallUI
		runAddInstallPlain = origInstallPlain
	})

	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		return tuiInstallResult{}, errNoTTYForTest{}
	}

	calledPlain := false
	runAddInstallPlain = func(source string, skillID string, targetDir string, force bool) (string, error) {
		calledPlain = true
		if source != "owner/repo" || skillID != "my-skill" || targetDir != "/tmp/skills" || !force {
			t.Fatalf("unexpected plain args: source=%q skill=%q dir=%q force=%v", source, skillID, targetDir, force)
		}
		return "/tmp/installed", nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "owner/repo", "my-skill", "--dir", "/tmp/skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%s", exit, errBuf.String())
	}
	if !calledPlain {
		t.Fatalf("expected plain installer fallback")
	}
}

type errNoTTYForTest struct{}

func (errNoTTYForTest) Error() string {
	return "could not open a new TTY: open /dev/tty: device not configured"
}

func TestRunAdd_WhenSkillIDProvided_StillInstallsDirectly(t *testing.T) {
	origSelect := runAddSelectFromSource
	origInstallUI := runAddInstallUI
	t.Cleanup(func() {
		runAddSelectFromSource = origSelect
		runAddInstallUI = origInstallUI
	})

	calledSelect := false
	runAddSelectFromSource = func(source string) (tuiSearchResult, error) {
		calledSelect = true
		return tuiSearchResult{}, nil
	}

	var gotSkill tuiSkill
	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		gotSkill = skill
		return tuiInstallResult{InstalledPath: "/tmp/installed"}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "owner/repo", "my-skill", "--dir", "/tmp/skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d, stderr=%s", exit, errBuf.String())
	}
	if calledSelect {
		t.Fatalf("selector should not have been called")
	}
	if gotSkill.SkillID != "my-skill" {
		t.Fatalf("got skill=%q", gotSkill.SkillID)
	}
}

func TestRun_ConfigSetAndGetDir(t *testing.T) {
	useTestConfigPath(t)

	_, errBufSet, restoreSet := captureStdoutStderr(t)
	exitSet := Run([]string{"config", "set", "dir", "/tmp/skills"})
	restoreSet()
	if exitSet != 0 {
		t.Fatalf("set exit=%d stderr=%s", exitSet, errBufSet.String())
	}

	outBufGet, errBufGet, restoreGet := captureStdoutStderr(t)
	exitGet := Run([]string{"config", "get"})
	restoreGet()
	if exitGet != 0 {
		t.Fatalf("get exit=%d stderr=%s", exitGet, errBufGet.String())
	}
	if !strings.Contains(outBufGet.String(), "/tmp/skills") {
		t.Fatalf("expected configured dir in output, got: %q", outBufGet.String())
	}
}

func TestRun_NoArgs_WhenDirNotConfigured_PromptsAndPersists(t *testing.T) {
	useTestConfigPath(t)

	origPrompt := promptForInstallDir
	origSearch := runSearchUI
	t.Cleanup(func() {
		promptForInstallDir = origPrompt
		runSearchUI = origSearch
	})

	promptForInstallDir = func() (string, error) {
		return "/tmp/prompted-skills", nil
	}
	runSearchUI = func() (tuiSearchResult, error) {
		return tuiSearchResult{Selected: false}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, errBuf.String())
	}

	dir, ok, err := getInstallDir()
	if err != nil {
		t.Fatal(err)
	}
	if !ok || dir != "/tmp/prompted-skills" {
		t.Fatalf("got dir=%q ok=%v", dir, ok)
	}
}

func TestRunAdd_DirFlagOverridesConfiguredDir(t *testing.T) {
	useTestConfigPath(t)

	if err := setInstallDir("/tmp/saved-skills"); err != nil {
		t.Fatal(err)
	}

	origInstallUI := runAddInstallUI
	t.Cleanup(func() { runAddInstallUI = origInstallUI })

	gotTarget := ""
	runAddInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		gotTarget = targetDir
		return tuiInstallResult{InstalledPath: "/tmp/installed"}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"add", "owner/repo", "my-skill", "--dir", "/tmp/flag-skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, errBuf.String())
	}
	if gotTarget != "/tmp/flag-skills" {
		t.Fatalf("target=%q", gotTarget)
	}
}

func TestRunSearch_DirFlagOverridesConfiguredDir(t *testing.T) {
	useTestConfigPath(t)

	if err := setInstallDir("/tmp/saved-skills"); err != nil {
		t.Fatal(err)
	}

	origSearch := runSearchUI
	origInstall := runSearchInstallUI
	t.Cleanup(func() {
		runSearchUI = origSearch
		runSearchInstallUI = origInstall
	})

	runSearchUI = func() (tuiSearchResult, error) {
		return tuiSearchResult{Selected: true, Skill: tuiSkill{Source: "owner/repo", SkillID: "chosen-skill"}}, nil
	}

	gotTarget := ""
	runSearchInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		gotTarget = targetDir
		return tuiInstallResult{InstalledPath: "/tmp/installed"}, nil
	}

	_, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"--dir", "/tmp/flag-skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, errBuf.String())
	}
	if gotTarget != "/tmp/flag-skills" {
		t.Fatalf("target=%q", gotTarget)
	}
}

func TestRunSearch_FinalSuccessLine_ShowsSourceAndHomeShortenedPath(t *testing.T) {
	origSearch := runSearchUI
	origInstall := runSearchInstallUI
	t.Cleanup(func() {
		runSearchUI = origSearch
		runSearchInstallUI = origInstall
	})

	home := t.TempDir()
	t.Setenv("HOME", home)

	runSearchUI = func() (tuiSearchResult, error) {
		return tuiSearchResult{Selected: true, Skill: tuiSkill{Source: "owner/repo", SkillID: "chosen-skill"}}, nil
	}
	runSearchInstallUI = func(targetDir string, force bool, skill tuiSkill) (tuiInstallResult, error) {
		return tuiInstallResult{InstalledPath: filepath.Join(home, ".pi/agent/skills/chosen-skill")}, nil
	}

	outBuf, errBuf, restore := captureStdoutStderr(t)
	exit := Run([]string{"--dir", "/tmp/skills"})
	restore()
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, errBuf.String())
	}
	out := outBuf.String()
	if !strings.Contains(out, "Installed chosen-skill") {
		t.Fatalf("missing install title line: %q", out)
	}
	if !strings.Contains(out, "Source: owner/repo") {
		t.Fatalf("missing source in success output: %q", out)
	}
	if !strings.Contains(out, "Path: ~/.pi/agent/skills/chosen-skill") {
		t.Fatalf("missing home-shortened path in success output: %q", out)
	}
}
