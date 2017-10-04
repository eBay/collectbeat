package factory

import (
	"fmt"

	dcommon "github.com/ebay/collectbeat/discoverer/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var factoryPlugins = make(map[string]FactoryConstructor)

type Meta interface{}

type Factory interface {
	Start(config *dcommon.ConfigHolder) error
	Stop(config *dcommon.ConfigHolder) error
	Restart(old, new *dcommon.ConfigHolder) error
}

type FactoryConstructor func(config *common.Config, meta Meta) (Factory, error)

func RegisterFactoryPlugin(name string, factory FactoryConstructor) {
	factoryPlugins[name] = factory
}

type FactoryPlugin struct {
	Name    string
	Config  *common.Config
	Factory Factory
}

func InitFactory(
	config *common.Config,
	meta Meta,
) (*FactoryPlugin, error) {
	if config == nil {
		return nil, fmt.Errorf("`factory` config needs to be defined")
	}

	conf := struct {
		Name string `config:"name"`
	}{}

	err := config.Unpack(&conf)
	if err != nil {
		return nil, err
	}

	if plugin, ok := factoryPlugins[conf.Name]; ok {
		config.PrintDebugf("Configure factory plugin '%v' with:", conf.Name)
		if config.Enabled() == false {
			return nil, fmt.Errorf("Plugin %s not enabled", conf.Name)
		}

		factory, err := plugin(config, meta)
		if err != nil {
			logp.Err("failed to initialize %s plugin as factory: %s", conf.Name, err)
			return nil, err
		}

		plugin := &FactoryPlugin{Name: conf.Name, Config: config, Factory: factory}
		return plugin, nil

	} else {
		return nil, fmt.Errorf("Factory %s does not exist", conf.Name)
	}
}
