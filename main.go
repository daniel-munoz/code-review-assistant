package main

import (
	"os"

	"github.com/daniel-munoz/code-review-assistant/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
