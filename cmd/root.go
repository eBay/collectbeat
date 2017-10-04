package cmd

var (
	Name    = "collectbeat"
	RootCmd = genRootCmd()
)

func init() {
	// Add metricbeat as a collectbeat subcommand
	metricbeatCmd := getMetricBeat().Command
	RootCmd.AddCommand(&metricbeatCmd)

	// Add filebeat as a collectbeat subcommand
	filebeatCmd := getFilebeat().Command
	RootCmd.AddCommand(&filebeatCmd)
}
