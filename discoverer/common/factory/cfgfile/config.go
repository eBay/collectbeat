package cfgfile

import "github.com/elastic/beats/libbeat/common"

type CfgfileConfig struct {
	Prefix         string         `config:"prefix"`
	ReloaderConfig *common.Config `config:"reloader_config"`
}

func defaultConfig() CfgfileConfig {
	return CfgfileConfig{
		Prefix: "collectbeat-",
	}
}
