package common

import "github.com/elastic/beats/libbeat/common"

type Meta common.MapStr

type ConfigHolder struct {
	Config *common.Config
	Meta   Meta
}
