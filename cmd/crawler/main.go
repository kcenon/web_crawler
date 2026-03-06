package main

import (
	"os"

	"github.com/kcenon/web_crawler/cmd/crawler/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
