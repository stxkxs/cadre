package main

import (
	"os"

	"github.com/stxkxs/cadre/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
