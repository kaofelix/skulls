package openutil

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

var command = exec.Command

// Open opens a URL or file target with the platform-default external handler.
func Open(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("empty open target")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = command("open", target)
	case "windows":
		cmd = command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = command("xdg-open", target)
	}
	return cmd.Start()
}
