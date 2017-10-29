package metricbeat

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Config is the root of the Collecbeat configuration hierarchy.
type Config struct {
	// Discoverers is a list of discoverer specific configurationd data.
	Discoverers map[string]*common.Config `config:"discovery"`
	// Upper bound on the random startup delay for metricsets (use 0 to disable startup delay).
	MaxStartDelay time.Duration  `config:"max_start_delay"`
	ConfigModules *common.Config `config:"config.modules"`
}

var defaultConfig = Config{
	MaxStartDelay: 10 * time.Second,
}
