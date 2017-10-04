package cmd

import (
	"flag"

	"github.com/ebay/collectbeat/beater/metricbeat"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/metricbeat/cmd/test"

	// Include all official metricbeat modules and metricsets
	_ "github.com/elastic/beats/metricbeat/include"

	// Add metricbeat specific processors
	_ "github.com/elastic/beats/metricbeat/processor/add_kubernetes_metadata"
)

const Metricbeat = "metricbeat"

func getMetricBeat() *cmd.BeatsRootCmd {
	rootCmd := cmd.GenRootCmd(Metricbeat, "", metricbeat.New)

	rootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, ""))
	rootCmd.AddCommand(cmd.GenModulesCmd(Name, "", buildModulesManager))
	rootCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))

	return rootCmd
}
