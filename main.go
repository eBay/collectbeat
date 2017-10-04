package main

import (
	"os"

	"github.com/ebay/collectbeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
