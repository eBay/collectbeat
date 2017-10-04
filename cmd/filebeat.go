package cmd

import (
	"flag"

	"github.com/ebay/collectbeat/beater/filebeat"

	cmd "github.com/elastic/beats/libbeat/cmd"
)

var Filebeat = "filebeat"

func getFilebeat() *cmd.BeatsRootCmd {
	rootCmd := cmd.GenRootCmd(Filebeat, "", filebeat.New)

	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("M"))
	rootCmd.TestCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	rootCmd.SetupCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	rootCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("once"))
	rootCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	rootCmd.AddCommand(cmd.GenModulesCmd(Filebeat, "", buildModulesManager))

	return rootCmd
}
