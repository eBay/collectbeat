package cmd

import (
	cmd "github.com/elastic/beats/libbeat/cmd"
)

func genRootCmd() *cmd.BeatsRootCmd {
	rootCmd := &cmd.BeatsRootCmd{}
	rootCmd.Use = "run"
	rootCmd.Short = "Run " + Name

	return rootCmd
}
