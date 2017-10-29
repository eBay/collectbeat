package common

import (
	"github.com/elastic/beats/libbeat/common"
)

type Meta common.MapStr

type ConfigHolder struct {
	Config common.MapStr
	Meta   Meta
}

func (c *ConfigHolder) GetConfigFromHolder() *common.Config {
	cfg, _ := common.NewConfigFrom(c.Config)
	return cfg
}

func GetMapFromConfig(config *common.Config) common.MapStr {
	out := common.MapStr{}
	err := config.Unpack(&out)

	if err != nil {
		return nil
	}
	return out
}
