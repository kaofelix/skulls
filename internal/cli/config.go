package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type configFile struct {
	Dir string `json:"dir"`
}

var configPathFunc = defaultConfigPath

func runConfig(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, "Usage: skulls config set dir <path> | skulls config get\n")
		return 2
	}

	switch args[0] {
	case "get":
		dir, ok, err := getInstallDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		if !ok {
			fmt.Println("dir: <not set>")
			return 0
		}
		fmt.Printf("dir: %s\n", dir)
		return 0
	case "set":
		if len(args) != 3 || args[1] != "dir" {
			fmt.Fprint(os.Stderr, "Usage: skulls config set dir <path>\n")
			return 2
		}
		if err := setInstallDir(args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Printf("Saved dir: %s\n", strings.TrimSpace(args[2]))
		return 0
	default:
		fmt.Fprint(os.Stderr, "Usage: skulls config set dir <path> | skulls config get\n")
		return 2
	}
}

func getInstallDir() (string, bool, error) {
	p, err := configPath()
	if err != nil {
		return "", false, err
	}

	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}

	var cfg configFile
	if err := json.Unmarshal(b, &cfg); err != nil {
		return "", false, err
	}
	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		return "", false, nil
	}
	return dir, true, nil
}

func setInstallDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("dir must be non-empty")
	}
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(configFile{Dir: dir}, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(p, b, 0o644)
}

func configPath() (string, error) {
	return configPathFunc()
}

func defaultConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "skulls", "config.json"), nil
}
