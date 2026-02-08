package main

import (
	"os"

	"github.com/cadre-oss/cadre/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
