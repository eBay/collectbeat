package discoverer

import (
	"sync"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/appender"
	"github.com/ebay/collectbeat/discoverer/common/builder"
	"github.com/ebay/collectbeat/discoverer/common/factory"

	"github.com/elastic/beats/libbeat/logp"
)

type Builders struct {
	runnerFactory factory.Factory
	sync.RWMutex
	builders  []builder.Builder
	appenders []appender.Appender
}

func NewBuilder(builders []builder.Builder, appenders []appender.Appender) *Builders {
	return &Builders{
		builders:  builders,
		appenders: appenders,
	}
}

func (b *Builders) AddBuilder(builder builder.Builder) {
	b.builders = append(b.builders, builder)
}

func (b *Builders) AddAppender(a appender.Appender) {
	b.appenders = append(b.appenders, a)
}

// AppendConfigs appends additional configs to a metricbeat config
func (b *Builders) appendConfigs(configs []*dcommon.ConfigHolder) {
	for _, config := range configs {
		b.appendConfig(config)
	}
}

func (b *Builders) appendConfig(config *dcommon.ConfigHolder) {
	b.RLock()
	defer b.RUnlock()

	for _, appender := range b.appenders {
		appender.Append(config)
	}
}

func (b *Builders) StartModuleRunners(obj interface{}) {
	b.RLock()
	defer b.RUnlock()

	for _, build := range b.builders {
		switch bType := build.(type) {
		case builder.PollerBuilder:
			configs := bType.BuildModuleConfigs(obj)
			b.appendConfigs(configs)

			err := b.runnerFactory.Start(configs)
			if err != nil {
				logp.Err("Module start up failed due to error %v", err)
			}
		case builder.PushBuilder:
			// Stop the older push metricset before starting an added configuration
			oldCfg := bType.ModuleConfig()
			b.appendConfig(oldCfg)

			config := bType.AddModuleConfig(obj)
			b.appendConfig(config)

			err := b.runnerFactory.Restart(oldCfg, config)
			if err != nil {
				logp.Err("Unable to restart module due to error %s", err)
			}
		default:
			logp.Err("Unsupported builder type %v", bType)
		}
	}
}

func (b *Builders) StopModuleRunners(obj interface{}) {
	b.RLock()
	defer b.RUnlock()

	for _, build := range b.builders {
		switch bType := build.(type) {
		case builder.PollerBuilder:
			configs := bType.BuildModuleConfigs(obj)
			b.appendConfigs(configs)

			err := b.runnerFactory.Stop(configs)
			if err != nil {
				logp.Err("Module stop failed due to error %v", err)
			}
		case builder.PushBuilder:
			// Stop the older push metricset before starting a metricset with removed configuration
			oldCfg := bType.ModuleConfig()
			b.appendConfig(oldCfg)

			config := bType.RemoveModuleConfig(obj)
			b.appendConfig(config)

			err := b.runnerFactory.Restart(oldCfg, config)
			if err != nil {
				logp.Err("Unable to restart module due to error %s", err)
			}
		default:
			logp.Err("Unsupported builder type %v", bType)
		}
	}
}

func (b *Builders) SetFactory(factory factory.Factory) {
	b.runnerFactory = factory
}
