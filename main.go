package main

import (
	"os"

	"github.com/daniel-munoz/code-review-assistant/cmd"
)

func main() {
	os.Exit(cmd.ExitCode(cmd.Execute()))
}
