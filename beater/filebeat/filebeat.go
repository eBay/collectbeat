package filebeat

import (
	"fmt"
	"sync"

	"github.com/ebay/collectbeat/discoverer"
	"github.com/ebay/collectbeat/discoverer/common/factory"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	"github.com/ebay/collectbeat/discoverer/kubernetes/common/builder/log_annotations"

	fbeater "github.com/elastic/beats/filebeat/beater"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/pkg/errors"

	_ "github.com/elastic/beats/filebeat/processor/add_kubernetes_metadata"
)

// Collectbeat implements the Beater interface.
type Collectbeat struct {
	done        chan struct{} // Channel used to initiate shutdown.
	discoverers []*discoverer.DiscovererPlugin
	staticbeat  beat.Beater
	config      Config
}

// New creates and returns a new Collectbeat instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	// Initialize staticbeat using metricbeat's beater
	filebeat, err := fbeater.New(b, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("error initializing staticbeat modules: %v", err)
	}

	config := defaultConfig
	err = rawConfig.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	// Register default configs for builders
	registerDefaultBuilderConfigs()
	discoverers, err := discoverer.InitDiscoverers(config.Discoverers)

	if err != nil {
		return nil, fmt.Errorf("error initializing discoverer: %v", err)
	}

	cb := &Collectbeat{
		done:        make(chan struct{}),
		staticbeat:  filebeat,
		config:      config,
		discoverers: discoverers,
	}
	return cb, nil
}

func (bt *Collectbeat) Run(b *beat.Beat) error {
	var wg sync.WaitGroup

	if bt.config.ConfigProspector == nil {
		rawProspectorConfig := map[string]interface{}{
			"enabled": true,
			"path":    "./prospectors.d/*.yml",
			"reload": map[string]interface{}{
				"enabled": true,
				"period":  "5s",
			},
		}

		conf, err := common.NewConfigFrom(rawProspectorConfig)
		if err != nil {
			return fmt.Errorf("Unable to create prospectors config")
		}
		bt.config.ConfigProspector = conf
	}
	if len(bt.discoverers) != 0 {
		factoryRawConf := map[string]interface{}{
			"name":            "cfgfile",
			"reloader_config": bt.config.ConfigProspector,
		}

		factoryCfg, err := common.NewConfigFrom(factoryRawConf)
		if err != nil {
			return fmt.Errorf("Factory config creation failed with error: %v", err)
		}

		runner, err := factory.InitFactory(factoryCfg, nil)
		if err != nil {
			return err
		}

		builder := &discoverer.Builders{}
		builder.SetFactory(runner.Factory)

		for _, disc := range bt.discoverers {
			d := disc
			go d.Discoverer.Start(builder)
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-bt.done
				d.Discoverer.Stop()
			}()
		}
	}

	// Start up staticbeat modules
	bt.staticbeat.Run(b)

	return nil
}

// Stop signals to Collectbeat that it should stop. It closes the "done" channel
// and closes the publisher client associated with each Module.
//
// Stop should only be called a single time. Calling it more than once may
// result in undefined behavior.
func (bt *Collectbeat) Stop() {
	bt.staticbeat.Stop()
}

func registerDefaultBuilderConfigs() {
	cfg := common.NewConfig()
	// Register default builders
	registry.BuilderRegistry.AddDefaultBuilderConfig(log_annotations.LogAnnotationsBuilder, *cfg)
}
