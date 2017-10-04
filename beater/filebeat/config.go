package filebeat

import (
	"github.com/elastic/beats/libbeat/common"
)

type Config struct {
	// Discoverers is a list of discoverer specific configurationd data.
	Discoverers      map[string]*common.Config `config:"discovery"`
	ConfigProspector *common.Config            `config:"config.prospectors"`
}

var defaultConfig = Config{}
