package metricbeat

import (
	"fmt"
	"sync"

	"github.com/ebay/collectbeat/discoverer"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	mbeater "github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/mb/module"

	"github.com/pkg/errors"

	//Add collectbeat specific discoverers
	factory "github.com/ebay/collectbeat/discoverer/common/factory"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	_ "github.com/ebay/collectbeat/discoverer/kubernetes"
	"github.com/ebay/collectbeat/discoverer/kubernetes/common/builder/metrics_annotations"
	"github.com/ebay/collectbeat/discoverer/kubernetes/common/builder/metrics_secret"
)

// Collectbeat implements the Beater interface.
type Collectbeat struct {
	done        chan struct{} // Channel used to initiate shutdown.
	discoverers []*discoverer.DiscovererPlugin
	metricbeat  beat.Beater
	config      Config
}

// New creates and returns a new Collectbeat instance.
func New(b *beat.Beat, rawConfig *common.Config) (beat.Beater, error) {
	// Initialize metricbeat using metricbeat's beater
	metricbeat, err := mbeater.New(b, rawConfig)
	if err != nil {
		return nil, fmt.Errorf("error initializing metricbeat modules: %v", err)
	}

	config := defaultConfig
	err = rawConfig.Unpack(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error reading configuration file")
	}

	registerDefaultBuilderConfigs()
	discoverers, err := discoverer.InitDiscoverers(config.Discoverers)

	if err != nil {
		return nil, fmt.Errorf("error initializing discoverer: %v", err)
	}

	cb := &Collectbeat{
		done:        make(chan struct{}),
		metricbeat:  metricbeat,
		config:      config,
		discoverers: discoverers,
	}
	return cb, nil
}

// Run starts the workers for Collectbeat and blocks until Stop is called
// and the workers complete. Each host associated with a MetricSet is given its
// own goroutine for fetching data. The ensures that each host is isolated so
// that a single unresponsive host cannot inadvertently block other hosts
// within the same Module and MetricSet from collection.
func (bt *Collectbeat) Run(b *beat.Beat) error {
	var wg sync.WaitGroup

	// Start up all discoverers
	f := module.NewFactory(bt.config.MaxStartDelay, b.Publisher)
	factoryRawConf := map[string]interface{}{
		"name": "runner",
	}

	factoryCfg, err := common.NewConfigFrom(factoryRawConf)
	if err != nil {
		return fmt.Errorf("Factory config creation failed with error: %v", err)
	}

	runner, err := factory.InitFactory(factoryCfg, f)
	if err != nil {
		return err
	}

	for _, disc := range bt.discoverers {
		d := disc
		go d.Discoverer.Start(runner.Factory)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-bt.done
			d.Discoverer.Stop()
		}()
	}

	// Start up metricbeat modules
	bt.metricbeat.Run(b)

	wg.Wait()
	return nil
}

// Stop signals to Collectbeat that it should stop. It closes the "done" channel
// and closes the publisher client associated with each Module.
//
// Stop should only be called a single time. Calling it more than once may
// result in undefined behavior.
func (bt *Collectbeat) Stop() {
	bt.metricbeat.Stop()
	close(bt.done)
}

func registerDefaultBuilderConfigs() {
	cfg := common.NewConfig()
	// Register default builders
	registry.BuilderRegistry.AddDefaultBuilderConfig(metrics_annotations.AnnotationsBuilder, *cfg)
	registry.BuilderRegistry.AddDefaultBuilderConfig(metrics_secret.SecretsBuilder, *cfg)
}
