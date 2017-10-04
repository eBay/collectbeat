package builder

import (
	dcommon "github.com/ebay/collectbeat/discoverer/common"

	"github.com/elastic/beats/libbeat/common"
)

// Builders take pods and generate module runners for them
type Builder interface {
	// Name returns the name of the builder
	Name() string
}

type PollerBuilder interface {
	Builder
	// BuildModuleRunner generates module runners for the given configs, then returns the
	// list of indexes to create, with the metadata to put on them
	BuildModuleConfigs(obj interface{}) []*dcommon.ConfigHolder
}

type PushBuilder interface {
	Builder
	// AddModuleConfig adds a configuration to the current configuration of the push metricset
	AddModuleConfig(obj interface{}) *dcommon.ConfigHolder

	// RemoveModuleConfig removes a configuration to the current configuration of the push metricset
	RemoveModuleConfig(obj interface{}) *dcommon.ConfigHolder

	// ModuleConfig returns current module config
	ModuleConfig() *dcommon.ConfigHolder
}

type ClientInfo common.MapStr

type BuilderConstructor func(config *common.Config, client ClientInfo) (Builder, error)
