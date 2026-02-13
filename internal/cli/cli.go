package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kaofelix/skulls/internal/install"
	"github.com/kaofelix/skulls/internal/skillsapi"
	"github.com/kaofelix/skulls/internal/tui"
)

type tuiSearchResult = tui.SearchResult
type tuiInstallResult = tui.InstallResult
type tuiSkill = skillsapi.Skill

var runAddInstallUI = tui.RunInstall
var runAddInstallPlain = func(source string, skillID string, targetDir string, force bool) (string, error) {
	return install.InstallSkill(source, skillID, install.Options{TargetDir: targetDir, Force: force})
}
var runAddSelectFromSource = tui.RunSearchFromSource
var runSearchUI = tui.RunSearch
var runSearchInstallUI = tui.RunInstall

const helpText = `skulls — dead simple skills

Usage:
  skulls [--dir <target-dir>] [--force]          # interactive search
  skulls add <source> [skill-id] [--dir <target-dir>]
  skulls config set dir <path>
  skulls config get

Source:
  - GitHub shorthand: owner/repo
  - Any git URL: https://..., git@..., file:///...
  - Local path to a git repo: ./path/to/repo

Examples:
  skulls add obra/superpowers using-git-worktrees --dir ~/.pi/agent/skills
  skulls add obra/superpowers@test-driven-development --dir ~/.pi/agent/skills
`

func Run(args []string) int {
	if len(args) == 0 {
		return runSearch(args)
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(helpText)
		return 0
	case "add":
		return runAdd(args[1:])
	case "config":
		return runConfig(args[1:])
	default:
		if strings.HasPrefix(args[0], "-") {
			return runSearch(args)
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", args[0])
		fmt.Fprint(os.Stderr, helpText)
		return 2
	}
}

type addArgs struct {
	TargetDir string
	Force     bool
	Help      bool
	Position  []string
}

func parseAddArgs(args []string) (addArgs, error) {
	var out addArgs

	flagMode := true
	for i := 0; i < len(args); i++ {
		a := args[i]

		if flagMode && a == "--" {
			flagMode = false
			continue
		}

		if flagMode && (a == "-h" || a == "--help") {
			out.Help = true
			continue
		}
		if flagMode && (a == "-f" || a == "--force") {
			out.Force = true
			continue
		}

		if flagMode && (a == "-d" || a == "--dir") {
			i++
			if i >= len(args) {
				return out, fmt.Errorf("%s requires a value", a)
			}
			out.TargetDir = args[i]
			continue
		}
		if flagMode && strings.HasPrefix(a, "--dir=") {
			out.TargetDir = strings.TrimPrefix(a, "--dir=")
			continue
		}

		if flagMode && strings.HasPrefix(a, "-") {
			return out, fmt.Errorf("unknown flag: %s", a)
		}

		out.Position = append(out.Position, a)
	}

	return out, nil
}

func runAdd(args []string) int {
	parsed, err := parseAddArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprint(os.Stderr, "Usage: skulls add <source> [skill-id] [--dir <target-dir>]\n")
		return 2
	}
	if parsed.Help {
		fmt.Fprint(os.Stderr, "Usage: skulls add <source> [skill-id] [--dir <target-dir>]\n")
		return 0
	}
	if len(parsed.Position) < 1 || len(parsed.Position) > 2 {
		fmt.Fprint(os.Stderr, "Usage: skulls add <source> [skill-id] [--dir <target-dir>]\n")
		return 2
	}

	source := strings.TrimSpace(parsed.Position[0])
	if source == "" {
		fmt.Fprint(os.Stderr, "source must be non-empty\n")
		return 2
	}

	targetDir, dirCtx, err := resolveInstallDirForRun(parsed.TargetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	var skillID string
	if len(parsed.Position) == 2 {
		skillID = strings.TrimSpace(parsed.Position[1])
		if skillID == "" {
			fmt.Fprint(os.Stderr, "skill-id must be non-empty\n")
			return 2
		}
	} else {
		if shorthandSource, shorthandSkill, ok := splitSourceSkillShorthand(source); ok {
			source = shorthandSource
			skillID = shorthandSkill
		} else {
			selection, err := runAddSelectFromSource(source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
			if !selection.Selected {
				return 0
			}
			skillID = strings.TrimSpace(selection.Skill.SkillID)
			if skillID == "" {
				fmt.Fprint(os.Stderr, "Error: selected skill is empty\n")
				return 1
			}
		}
	}

	installRes, err := runAddInstallUI(targetDir, true, skillsapi.Skill{Source: source, SkillID: skillID})
	if err != nil {
		if isNoTTYError(err) {
			installedPath, plainErr := runAddInstallPlain(source, skillID, targetDir, true)
			if plainErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", plainErr)
				return 1
			}
			printInstallSuccess(skillID, source, installedPath)
			printInstallTip(dirCtx, targetDir)
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if installRes.Err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", installRes.Err)
		return 1
	}

	printInstallSuccess(skillID, source, installRes.InstalledPath)
	printInstallTip(dirCtx, targetDir)
	return 0
}

func isNoTTYError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "open /dev/tty") || strings.Contains(msg, "could not open a new tty")
}

type searchArgs struct {
	TargetDir string
	Force     bool
	Help      bool
}

func parseSearchArgs(args []string) (searchArgs, error) {
	var out searchArgs

	for i := 0; i < len(args); i++ {
		a := args[i]

		switch {
		case a == "-h" || a == "--help":
			out.Help = true
			continue
		case a == "-f" || a == "--force":
			out.Force = true
			continue
		case a == "-d" || a == "--dir":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("%s requires a value", a)
			}
			out.TargetDir = args[i]
			continue
		case strings.HasPrefix(a, "--dir="):
			out.TargetDir = strings.TrimPrefix(a, "--dir=")
			continue
		default:
			if strings.HasPrefix(a, "-") {
				return out, fmt.Errorf("unknown flag: %s", a)
			}
			return out, fmt.Errorf("unexpected argument: %s", a)
		}
	}

	return out, nil
}

func splitSourceSkillShorthand(source string) (string, string, bool) {
	s := strings.TrimSpace(source)
	if strings.Count(s, "@") != 1 {
		return "", "", false
	}
	parts := strings.SplitN(s, "@", 2)
	repo := strings.TrimSpace(parts[0])
	skill := strings.TrimSpace(parts[1])
	if repo == "" || skill == "" {
		return "", "", false
	}
	if strings.Contains(repo, "://") || strings.HasPrefix(repo, "git@") || strings.HasPrefix(repo, "/") || strings.HasPrefix(repo, "./") || strings.HasPrefix(repo, "../") {
		return "", "", false
	}
	repoParts := strings.Split(repo, "/")
	if len(repoParts) != 2 || strings.TrimSpace(repoParts[0]) == "" || strings.TrimSpace(repoParts[1]) == "" {
		return "", "", false
	}
	return repo, skill, true
}

func runSearch(args []string) int {
	parsed, err := parseSearchArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprint(os.Stderr, "Usage: skulls [--dir <target-dir>] [--force]\n")
		return 2
	}
	if parsed.Help {
		fmt.Fprint(os.Stderr, "Usage: skulls [--dir <target-dir>] [--force]\n")
		return 0
	}
	targetDir, dirCtx, err := resolveInstallDirForRun(parsed.TargetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	searchRes, err := runSearchUI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if !searchRes.Selected {
		return 0
	}

	installRes, err := runSearchInstallUI(targetDir, parsed.Force, searchRes.Skill)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if installRes.Err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", installRes.Err)
		return 1
	}

	printInstallSuccess(searchRes.Skill.SkillID, searchRes.Skill.Source, installRes.InstalledPath)
	printInstallTip(dirCtx, targetDir)
	return 0
}

type installDirContext struct {
	UsedFlag      bool
	HasConfigured bool
	ConfiguredDir string
}

func resolveInstallDirForRun(flagValue string) (string, installDirContext, error) {
	ctx := installDirContext{}

	if configured, ok, err := getInstallDir(); err != nil {
		return "", ctx, err
	} else if ok {
		ctx.HasConfigured = true
		ctx.ConfiguredDir = configured
	}

	if v := strings.TrimSpace(flagValue); v != "" {
		ctx.UsedFlag = true
		return v, ctx, nil
	}
	if ctx.HasConfigured {
		return ctx.ConfiguredDir, ctx, nil
	}

	return "", ctx, fmt.Errorf("install dir is not configured yet ☠️\nUse --dir <target-dir> for this run, or set a default:\n  skulls config set dir <path>")
}

func printInstallTip(ctx installDirContext, targetDir string) {
	if !ctx.UsedFlag {
		return
	}
	cmd := fmt.Sprintf("skulls config set dir %s", strconv.Quote(strings.TrimSpace(targetDir)))
	if !ctx.HasConfigured {
		printTipBox(
			[]string{
				fmt.Sprintf("Installed to %s for this run.", compactPath(targetDir)),
				"Want to make it your default install dir?",
			},
			cmd,
		)
		return
	}
	if samePath(ctx.ConfiguredDir, targetDir) {
		return
	}
	printTipBox(
		[]string{
			fmt.Sprintf("Default dir is %s.", compactPath(ctx.ConfiguredDir)),
			fmt.Sprintf("This install used %s.", compactPath(targetDir)),
			"To make this your new default:",
		},
		cmd,
	)
}

func printTipBox(lines []string, command string) {
	if len(lines) == 0 && strings.TrimSpace(command) == "" {
		return
	}
	color := shouldUseTipColor()

	titleStyle := lipgloss.NewStyle().Bold(true)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	cmdStyle := lipgloss.NewStyle().Bold(true)

	if color {
		titleStyle = titleStyle.Foreground(lipgloss.Color("6"))
		boxStyle = boxStyle.BorderForeground(lipgloss.Color("8"))
		cmdStyle = cmdStyle.Foreground(lipgloss.Color("6"))
	}

	body := make([]string, 0, len(lines)+2)
	body = append(body, titleStyle.Render("☠️ Tip"))
	body = append(body, lines...)
	if strings.TrimSpace(command) != "" {
		body = append(body, "  "+cmdStyle.Render(command))
	}

	fmt.Println()
	fmt.Println(boxStyle.Render(strings.Join(body, "\n")))
}

func shouldUseTipColor() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func samePath(a string, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return a == b
	}
	aAbs, aErr := filepath.Abs(a)
	bAbs, bErr := filepath.Abs(b)
	if aErr != nil || bErr != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(aAbs) == filepath.Clean(bAbs)
}

func printInstallSuccess(skillID string, source string, installedPath string) {
	fmt.Printf("\n💀 Installed %s\n", strings.TrimSpace(skillID))
	if strings.TrimSpace(source) != "" {
		fmt.Printf("   Source: %s\n", strings.TrimSpace(source))
	}
	fmt.Printf("   Path: %s\n", compactPath(installedPath))
}

func compactPath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return abs
	}
	home = filepath.Clean(home)
	if abs == home {
		return "~"
	}
	prefix := home + string(filepath.Separator)
	if strings.HasPrefix(abs, prefix) {
		rel := strings.TrimPrefix(abs, prefix)
		if rel == "" {
			return "~"
		}
		return "~" + string(filepath.Separator) + rel
	}
	return abs
}
