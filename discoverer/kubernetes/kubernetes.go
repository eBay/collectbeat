package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ebay/collectbeat/discoverer"
	"github.com/ebay/collectbeat/discoverer/common/appender"
	"github.com/ebay/collectbeat/discoverer/common/builder"
	"github.com/ebay/collectbeat/discoverer/common/factory"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	kubecommon "github.com/ebay/collectbeat/discoverer/kubernetes/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/ericchiang/k8s"
	"github.com/ghodss/yaml"
)

const (
	ready_timeout = time.Second * 5
)

var (
	fatalError = errors.New("Unable to create kubernetes processor")
	debug      = logp.MakeDebug("kubernetes")
)

type kubernetesDiscoverer struct {
	podWatcher *PodWatcher
}

func init() {
	discoverer.RegisterDiscovererPlugin("kubernetes", newKubernetesDiscoverer)
}

func newKubernetesDiscoverer(cfg *common.Config) (discoverer.Discoverer, error) {
	config := defaultKuberentesDiscovererConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes configuration: %s", err)
	}

	//Load default builder configs
	if config.DefaultBuilders.Enabled == true {
		registry.BuilderRegistry.RLock()
		for key, cfg := range registry.BuilderRegistry.GetDefaultBuilderConfigs() {
			config.Builders = append(config.Builders, map[string]*common.Config{key: &cfg})
		}
		registry.BuilderRegistry.RUnlock()
	}

	//Load default builder configs
	if config.DefaultAppenders.Enabled == true {
		registry.BuilderRegistry.RLock()
		for key, cfg := range registry.BuilderRegistry.GetDefaultAppenderConfigs() {
			config.Appenders = append(config.Appenders, map[string]*common.Config{key: &cfg})
		}
		registry.BuilderRegistry.RUnlock()
	}

	builders := []builder.Builder{}
	appenders := []appender.Appender{}

	var client *k8s.Client
	if config.InCluster == true {
		client, err = k8s.NewInClusterClient()
		if err != nil {
			return nil, fmt.Errorf("Unable to get in cluster configuration")
		}
	} else {
		data, err := ioutil.ReadFile(config.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("read kubeconfig: %v", err)
		}

		// Unmarshal YAML into a Kubernetes config object.
		var config k8s.Config
		if err = yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("unmarshal kubeconfig: %v", err)
		}
		client, err = k8s.NewClient(&config)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	if config.Host == "" {
		podName := os.Getenv("HOSTNAME")
		logp.Info("Using pod name %s and namespace %s", podName, config.Namespace)
		if podName == "localhost" {
			config.Host = "localhost"
		} else {
			pod, error := client.CoreV1().GetPod(ctx, podName, config.Namespace)
			if error != nil {
				logp.Err("Querying for pod failed with error: ", error.Error())
				logp.Info("Unable to find pod, setting host to localhost")
				config.Host = "localhost"
			} else {
				config.Host = pod.Spec.GetNodeName()
			}

		}
	}

	debug("kubernetes", "Using host ", config.Host)
	debug("kubernetes", "Initializing watcher")
	if client != nil {

		clientInfo := builder.ClientInfo{
			kubecommon.ClientKey: client,
		}
		//Create all configured builders
		for _, pluginConfigs := range config.Builders {
			for name, pluginConfig := range pluginConfigs {
				indexFunc := registry.BuilderRegistry.GetBuilder(name)
				if indexFunc == nil {
					logp.Warn("Unable to find builder plugin %s", name)
					continue
				}

				builder, err := indexFunc(pluginConfig, clientInfo)
				if err != nil {
					logp.Warn("Unable to initialize indexing plugin %s due to error %v", name, err)
					continue
				}

				if builder != nil {
					builders = append(builders, builder)
				}

			}
		}

		//Create all configured appenders
		for _, pluginConfigs := range config.Appenders {
			for name, pluginConfig := range pluginConfigs {
				indexFunc := registry.BuilderRegistry.GetAppender(name)
				if indexFunc == nil {
					logp.Warn("Unable to find appender plugin %s", name)
					continue
				}

				appender, err := indexFunc(pluginConfig)
				if err != nil {
					logp.Warn("Unable to initialize appender plugin %s due to error %v", name, err)
				}

				appenders = append(appenders, appender)

			}
		}

		if len(builders) == 0 {
			return nil, fmt.Errorf("Can not initialize kubernetes plugin with zero builder plugins")
		}

		builder := discoverer.NewBuilder(builders, appenders)
		watcher := NewPodWatcher(client, builder, config.SyncPeriod, config.Host)
		return kubernetesDiscoverer{podWatcher: watcher}, nil
	}

	return nil, fatalError
}

func (k kubernetesDiscoverer) Start(factory factory.Factory) {
	k.podWatcher.builders.SetFactory(factory)
	k.podWatcher.Run()
}

func (k kubernetesDiscoverer) Stop() {
	k.podWatcher.Stop()
}

func (k kubernetesDiscoverer) String() string { return "kubernetes" }
