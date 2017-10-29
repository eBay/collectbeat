package graphite_annotations

import (
	"fmt"
	"strings"
	"sync"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/builder"
	"github.com/ebay/collectbeat/discoverer/common/metagen"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	kubecommon "github.com/ebay/collectbeat/discoverer/kubernetes/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
	"github.com/elastic/beats/metricbeat/module/graphite/server"
)

const (
	delimiter = "delimiter"
	filter    = "filter"
	namespace = "namespace"
	tags      = "tags"
	template  = "template"

	graphite_default_delimiter = "."
	graphite_default_prefix    = "io.collectbeat.graphite/"

	GraphiteBuilder = "graphite_annotations"
)

var baseConfig = map[string]interface{}{
	"module":     "graphite",
	"metricsets": []string{"server"},
	"enabled":    true,
}

var debug = logp.MakeDebug(GraphiteBuilder)

func init() {
	registry.BuilderRegistry.AddBuilder(GraphiteBuilder, NewGraphiteAnnotationBuilder)
}

type podMap struct {
	pods      map[string][]string
	templates map[string]server.TemplateConfig
	sync.RWMutex
}

// GraphiteAnnotationBuilder implements graphite server metricset based on pod annotations
type GraphiteAnnotationBuilder struct {
	Prefix        string
	Config        *common.Config
	BaseTemplates map[string]server.TemplateConfig
	podMap        *podMap
}

func newPodMap() podMap {
	return podMap{
		pods:      make(map[string][]string),
		templates: make(map[string]server.TemplateConfig),
	}
}

func NewGraphiteAnnotationBuilder(cfg *common.Config, _ builder.ClientInfo, _ metagen.MetaGen) (builder.Builder, error) {
	config := struct {
		Prefix string         `config:"prefix"`
		Config *common.Config `config:"config"`
	}{
		Prefix: graphite_default_prefix,
	}
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("Failed to unpack the `graphite_annotations` builder configuration: %s", err)
	}

	//Add / to the end of the annotation namespace
	if config.Prefix[len(config.Prefix)-1] != '/' {
		config.Prefix = config.Prefix + "/"
	}

	templateMap := make(map[string]server.TemplateConfig)

	gConf := server.DefaultGraphiteCollectorConfig()
	err = config.Config.Unpack(&gConf)

	if err != nil {
		return nil, fmt.Errorf("Failed to unpack the `graphite server` builder configuration: %s", err)
	}
	for _, templ := range gConf.Templates {
		templateMap[templ.Filter] = templ
	}

	err = config.Config.Merge(gConf)
	if err != nil {
		return nil, fmt.Errorf("Failed to merge graphite server config with base graphite config: %v", err)
	}

	err = config.Config.Merge(baseConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to merge graphite server config with base config: %v", err)
	}

	pMap := newPodMap()
	return &GraphiteAnnotationBuilder{
		Prefix:        config.Prefix,
		Config:        config.Config,
		BaseTemplates: templateMap,
		podMap:        &pMap,
	}, nil

}

func (p *GraphiteAnnotationBuilder) Name() string {
	return "Graphite Annotation Builder"
}

func (g *GraphiteAnnotationBuilder) AddModuleConfig(obj interface{}) *dcommon.ConfigHolder {
	holder := g.ModuleConfig()
	pod, ok := obj.(*kubernetes.Pod)
	if !ok {
		logp.Err("Unable to cast %v to type *v1.Pod", obj)
		return holder
	}

	templ := g.getTemplateFromPod(pod)

	if templ == nil {
		return holder
	}

	config := server.GraphiteServerConfig{}
	holder.GetConfigFromHolder().Unpack(&config)

	if _, ok := g.BaseTemplates[templ.Filter]; ok {
		logp.Err("Can not register filter that is present in base config %s", templ.Filter)
		return holder
	}

	podMetaNS := fmt.Sprintf("%s/%s", pod.Metadata.Namespace, pod.Metadata.Name)
	if pods, ok := g.podMap.pods[templ.Filter]; ok {
		if tempConf, ok := g.podMap.templates[templ.Filter]; ok {
			if tempConf.Template != templ.Template {
				logp.Err("Can not register new template for already existing filter %s", tempConf.Template)
				return holder
			}

			if tempConf.Namespace != templ.Namespace {
				logp.Err("Can not register new namespace for already existing filter %s", tempConf.Namespace)
				return holder
			}

			if tempConf.Delimiter != templ.Delimiter {
				logp.Err("Can not register new delimiter for already existing filter %s", tempConf.Delimiter)
				return holder
			}
			pods = append(pods, podMetaNS)
			g.podMap.pods[templ.Filter] = pods
		}
	} else {
		g.podMap.templates[templ.Filter] = *templ
		g.podMap.pods[templ.Filter] = []string{podMetaNS}
	}

	newHolder := g.ModuleConfig()
	err := newHolder.GetConfigFromHolder().Merge(baseConfig)
	if err != nil {
		logp.Err("Error merging base config: %v", err)
		return nil
	}

	return newHolder
}

func (g *GraphiteAnnotationBuilder) RemoveModuleConfig(obj interface{}) *dcommon.ConfigHolder {

	holder := g.ModuleConfig()

	pod, ok := obj.(*kubernetes.Pod)
	if !ok {
		logp.Err("Unable to cast %v to type *v1.Pod", obj)
		return holder
	}

	templ := g.getTemplateFromPod(pod)
	if templ == nil {
		return holder
	}

	config := server.GraphiteServerConfig{}
	holder.GetConfigFromHolder().Unpack(&config)

	podMetaNS := fmt.Sprintf("%s/%s", pod.Metadata.Namespace, pod.Metadata.Name)
	if pods, ok := g.podMap.pods[templ.Filter]; ok {
		if tempConf, ok := g.podMap.templates[templ.Filter]; ok {
			if tempConf.Template != templ.Template {
				logp.Err("Can not unregister template as template is different for given filter %s", tempConf.Template)
				return holder
			}

			if tempConf.Namespace != templ.Namespace {
				logp.Err("Can not unregister namespace as namespace is different for given filter %s", tempConf.Namespace)
				return holder
			}

			if tempConf.Delimiter != templ.Delimiter {
				logp.Err("Can not unregister delimiter as delimiter is different for given filter %s", tempConf.Delimiter)
				return holder
			}

			// Remove pod reference from filter map
			for i := 0; i < len(pods); i++ {
				if pods[i] == podMetaNS {
					pods = append(pods[0:i], pods[i+1:]...)
					break
				}
			}

			// If no more pods are using the template then remove it
			if len(pods) == 0 {
				delete(g.podMap.pods, templ.Filter)
				delete(g.podMap.templates, templ.Filter)
			} else {
				g.podMap.pods[templ.Filter] = pods
			}
		}
	}

	newHolder := g.ModuleConfig()
	cfg := newHolder.GetConfigFromHolder()
	err := cfg.Merge(baseConfig)
	if err != nil {
		logp.Err("Error merging base config: %v", err)
		return nil
	}

	newHolder.Config = dcommon.GetMapFromConfig(cfg)
	return newHolder
}

func (g *GraphiteAnnotationBuilder) ModuleConfig() *dcommon.ConfigHolder {
	gConf := server.GraphiteServerConfig{}
	err := g.Config.Unpack(&gConf)

	if err != nil {
		logp.Err("Error unpacking configuration %v", err)
		return nil
	}

	for _, template := range g.podMap.templates {
		gConf.Templates = append(gConf.Templates, template)
	}

	cfg, err := common.NewConfigFrom(gConf)
	if err != nil {
		logp.Err("Error packing configuration %v", err)
		return nil
	}

	err = cfg.Merge(g.Config)
	if err != nil {
		logp.Err("Error merging base config: %v", err)
		return nil
	}

	return &dcommon.ConfigHolder{
		Config: dcommon.GetMapFromConfig(cfg),
	}
}

func (g *GraphiteAnnotationBuilder) getTemplateFromPod(pod *kubernetes.Pod) *server.TemplateConfig {
	debug("Entering pod %s for graphite builder", pod.Metadata.Name)

	// Process only if a pod got an IP
	ip := kubecommon.GetPodIp(pod)
	if ip == "" {
		return nil
	}

	// Check if filter is present in annotations
	filter := g.getFilter(pod)
	if filter == "" {
		return nil
	}

	// Check if template is present in annotations
	template := g.getTemplate(pod)
	if template == "" {
		return nil
	}

	// Check if template is present in annotations
	namespace := g.getNamespace(pod)
	if namespace == "" {
		return nil
	}

	del := g.getDelimiter(pod)
	tags := g.getTags(pod)

	return &server.TemplateConfig{
		Namespace: namespace,
		Template:  template,
		Filter:    filter,
		Delimiter: del,
		Tags:      tags,
	}

}

func (p *GraphiteAnnotationBuilder) getNamespace(pod *kubernetes.Pod) string {
	return kubecommon.GetAnnotationWithPrefix(namespace, p.Prefix, pod)
}

func (p *GraphiteAnnotationBuilder) getFilter(pod *kubernetes.Pod) string {
	return kubecommon.GetAnnotationWithPrefix(filter, p.Prefix, pod)
}

func (p *GraphiteAnnotationBuilder) getTemplate(pod *kubernetes.Pod) string {
	return kubecommon.GetAnnotationWithPrefix(template, p.Prefix, pod)
}

func (p *GraphiteAnnotationBuilder) getDelimiter(pod *kubernetes.Pod) string {
	d := kubecommon.GetAnnotationWithPrefix(delimiter, p.Prefix, pod)
	if d != "" {
		return d
	} else {
		return graphite_default_delimiter
	}
}

func (p *GraphiteAnnotationBuilder) getTags(pod *kubernetes.Pod) map[string]string {
	tagsMap := make(map[string]string)
	tagStr := kubecommon.GetAnnotationWithPrefix(tags, p.Prefix, pod)
	if tagStr != "" {
		values := strings.Split(tagStr, ",")
		if len(values) != 0 {
			for _, value := range values {
				keyvalue := strings.Split(value, "=")
				if len(keyvalue) == 2 {
					tagsMap[keyvalue[0]] = keyvalue[1]
				}
			}
		}
	}

	return tagsMap
}
