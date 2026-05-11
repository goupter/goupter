package main

import (
	"os"

	"github.com/goupter/goupter/cmd/goupter/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
