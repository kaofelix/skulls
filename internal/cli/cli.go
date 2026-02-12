package cli

import (
	"fmt"
	"os"
	"strings"

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

	targetDir, err := resolveInstallDir(parsed.TargetDir)
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
			fmt.Printf("Installed %s to %s\n", skillID, installedPath)
			return 0
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if installRes.Err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", installRes.Err)
		return 1
	}

	fmt.Printf("Installed %s to %s\n", skillID, installRes.InstalledPath)
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
	targetDir, err := resolveInstallDir(parsed.TargetDir)
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

	fmt.Printf("\n💀 Installed %s to %s\n", searchRes.Skill.SkillID, installRes.InstalledPath)
	return 0
}
