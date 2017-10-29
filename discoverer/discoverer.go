package discoverer

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	_ "github.com/ebay/collectbeat/discoverer/include"
)

var discovererPlugins = make(map[string]Constructor)

type Discoverer interface {
	Start(builder *Builders)
	Stop()
	String() string
}

type Constructor func(config *common.Config) (Discoverer, error)

func RegisterDiscovererPlugin(name string, discoverer Constructor) {
	discovererPlugins[name] = discoverer
}

type DiscovererPlugin struct {
	Name       string
	Config     *common.Config
	Discoverer Discoverer
}

func InitDiscoverers(
	configs map[string]*common.Config,
) ([]*DiscovererPlugin, error) {
	var plugins []*DiscovererPlugin
	for name, plugin := range discovererPlugins {
		config, exists := configs[name]
		if !exists {
			continue
		}

		config.PrintDebugf("Configure discovery plugin '%v' with:", name)
		if !config.Enabled() {
			continue
		}

		discoverer, err := plugin(config)
		if err != nil {
			logp.Err("failed to initialize %s plugin as discovery: %s", name, err)
			return nil, err
		}

		plugin := &DiscovererPlugin{Name: name, Config: config, Discoverer: discoverer}
		plugins = append(plugins, plugin)
		logp.Info("Activated %s as discovery plugin.", name)
	}
	return plugins, nil
}
