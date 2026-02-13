package main

import (
	"os"

	"github.com/kaofelix/skulls/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
