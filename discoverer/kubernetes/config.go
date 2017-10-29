package kubernetes

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

type kubeDiscovererConfig struct {
	InCluster          bool                    `config:"in_cluster"`
	KubeConfig         string                  `config:"kube_config"`
	Host               string                  `config:"host"`
	Namespace          string                  `config:"namespace"`
	SyncPeriod         time.Duration           `config:"sync_period"`
	Builders           PluginConfig            `config:"builders"`
	DefaultBuilders    Enabled                 `config:"default_builders"`
	Appenders          PluginConfig            `config:"appenders"`
	DefaultAppenders   Enabled                 `config:"default_appenders"`
	Indexers           kubernetes.PluginConfig `config:"indexers"`
	DefaultIndexers    Enabled                 `config:"default_indexers"`
	IncludeLabels      []string                `config:"include_labels"`
	ExcludeLabels      []string                `config:"exclude_labels"`
	IncludeAnnotations []string                `config:"include_annotations"`
}

type Enabled struct {
	Enabled bool `config:"enabled"`
}

type PluginConfig []map[string]*common.Config

func defaultKuberentesDiscovererConfig() kubeDiscovererConfig {
	return kubeDiscovererConfig{
		InCluster:        true,
		SyncPeriod:       1 * time.Second,
		Namespace:        "kube-system",
		DefaultBuilders:  Enabled{true},
		DefaultAppenders: Enabled{true},
		DefaultIndexers:  Enabled{true},
	}
}

func (k kubeDiscovererConfig) Validate() error {
	if !k.InCluster && k.KubeConfig == "" {
		return fmt.Errorf("`kube_config` path can't be empty when in_cluster is set to false")
	}
	return nil
}
