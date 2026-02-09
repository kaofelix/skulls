package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kaofelix/skulls/internal/install"
	"github.com/kaofelix/skulls/internal/tui"
)

const helpText = `skulls — dead simple skills

Usage:
  skulls --dir <target-dir> [--force]          # interactive search
  skulls add <source> <skill-id> --dir <target-dir> [--force]

Source:
  - GitHub shorthand: owner/repo
  - Any git URL: https://..., git@..., file:///...
  - Local path to a git repo: ./path/to/repo

Examples:
  skulls add obra/superpowers using-git-worktrees --dir ~/.pi/agent/skills
`

func Run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, "--dir is required for search mode for now. Try: skulls --dir <target-dir>\n\n")
		fmt.Fprint(os.Stderr, helpText)
		return 2
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(helpText)
		return 0
	case "add":
		return runAdd(args[1:])
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
		fmt.Fprint(os.Stderr, "Usage: skulls add <source> <skill-id> --dir <target-dir> [--force]\n")
		return 2
	}
	if parsed.Help {
		fmt.Fprint(os.Stderr, "Usage: skulls add <source> <skill-id> --dir <target-dir> [--force]\n")
		return 0
	}
	if len(parsed.Position) != 2 {
		fmt.Fprint(os.Stderr, "Usage: skulls add <source> <skill-id> --dir <target-dir> [--force]\n")
		return 2
	}

	source := strings.TrimSpace(parsed.Position[0])
	skillID := strings.TrimSpace(parsed.Position[1])
	if source == "" || skillID == "" {
		fmt.Fprint(os.Stderr, "source and skill-id must be non-empty\n")
		return 2
	}
	if strings.TrimSpace(parsed.TargetDir) == "" {
		fmt.Fprint(os.Stderr, "--dir is required for now\n")
		return 2
	}

	installedPath, err := install.InstallSkill(source, skillID, install.Options{
		TargetDir: parsed.TargetDir,
		Force:     parsed.Force,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("Installed %s to %s\n", skillID, installedPath)
	return 0
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

func runSearch(args []string) int {
	parsed, err := parseSearchArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprint(os.Stderr, "Usage: skulls --dir <target-dir> [--force]\n")
		return 2
	}
	if parsed.Help {
		fmt.Fprint(os.Stderr, "Usage: skulls --dir <target-dir> [--force]\n")
		return 0
	}
	if strings.TrimSpace(parsed.TargetDir) == "" {
		fmt.Fprint(os.Stderr, "--dir is required for now\n")
		return 2
	}

	searchRes, err := tui.RunSearch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if !searchRes.Selected {
		return 0
	}

	installRes, err := tui.RunInstall(parsed.TargetDir, parsed.Force, searchRes.Skill)
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
