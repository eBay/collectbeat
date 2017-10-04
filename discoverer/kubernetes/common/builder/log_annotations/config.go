package log_annotations

import "github.com/elastic/beats/libbeat/common"

type LogPathConfig struct {
	Prefix               string         `config:"prefix"`
	BaseProspectorConfig *common.Config `config:"base_prospector_config"`
	LogsPath             string         `config:"logs_path"`
	DefaultNamespace     string         `config:"default_namespace"`
	CustomPath           CustomPath     `config:"custom_path"`
}

type CustomPath struct {
	Enabled bool `config:"enabled"`
}

func DefaultLogPathConfig() LogPathConfig {
	return LogPathConfig{
		Prefix:               default_prefix,
		BaseProspectorConfig: defaultBaseProspectorConfig(),
		LogsPath:             "/var/lib/docker/containers/",
		CustomPath: CustomPath{
			Enabled: false,
		},
	}
}
