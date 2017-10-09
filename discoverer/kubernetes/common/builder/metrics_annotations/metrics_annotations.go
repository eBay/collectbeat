package metrics_annotations

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/builder"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	kubecommon "github.com/ebay/collectbeat/discoverer/kubernetes/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	corev1 "github.com/ericchiang/k8s/api/v1"
)

const (
	metrictype           = "type"
	namespace            = "namespace"
	endpoints            = "endpoints"
	metricsets           = "metricsets"
	interval             = "interval"
	timeout              = "timeout"
	scheme               = "scheme"
	insecure_skip_verify = "insecure_skip_verify"

	default_prefix   = "io.collectbeat.metrics/"
	default_timeout  = time.Second * 3
	default_interval = time.Minute

	AnnotationsBuilder = "metrics_annotations"
)

var (
	debug = logp.MakeDebug(AnnotationsBuilder)
)

func init() {
	registry.BuilderRegistry.AddBuilder(AnnotationsBuilder, NewPodAnnotationBuilder)
}

// PodAnnotationBuilder implements default modules based on pod annotations
type PodAnnotationBuilder struct {
	Prefix string
}

func NewPodAnnotationBuilder(cfg *common.Config, _ builder.ClientInfo) (builder.Builder, error) {
	config := struct {
		Prefix string `config:"prefix"`
	}{
		Prefix: default_prefix,
	}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the `annotations` builder configuration: %s", err)
	}

	//Add . to the end of the annotation namespace
	if config.Prefix[len(config.Prefix)-1] != '/' {
		config.Prefix = config.Prefix + "/"
	}

	return &PodAnnotationBuilder{Prefix: config.Prefix}, nil
}

func (p *PodAnnotationBuilder) Name() string {
	return "Annotation Builder"
}

func (p *PodAnnotationBuilder) BuildModuleConfigs(obj interface{}) []*dcommon.ConfigHolder {
	holders := []*dcommon.ConfigHolder{}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		logp.Err("Unable to cast %v to type *v1.Pod", obj)
		return holders
	}

	debug("Entering pod %s for annotations builder", pod.Metadata.GetName())
	ip := kubecommon.GetPodIp(pod)
	if ip == "" {
		return holders
	}

	mendpoints := p.getEndpoints(ip, pod)
	if len(mendpoints) == 0 {
		return holders
	}

	mtype := p.getMetricType(pod)
	if mtype == "" {
		return holders
	}

	msets := p.getMetricSets(mtype, pod)
	if len(msets) == 0 {
		return holders
	}

	minterval := p.getInterval(pod)
	mtimeout := p.getTimeout(pod)
	mverify := p.getInsecureSkipVerify(pod)

	moduleConfig := map[string]interface{}{
		"module":     mtype,
		"metricsets": msets,
		"hosts":      mendpoints,
		"timeout":    mtimeout,
		"period":     minterval,
		"enabled":    true,
	}

	ns := p.getNamespace(pod)
	if p.isNamespaceRequired(mtype) == true && ns == "" {
		return holders
	} else {
		moduleConfig["namespace"] = ns
	}

	if mverify == true {
		ssl := map[string]interface{}{
			"verification_mode": "none",
		}
		moduleConfig["ssl"] = ssl
	}

	config, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		return holders
	}

	debug("Config for pod %s is %v", pod.Metadata.GetName(), *config)

	holder := &dcommon.ConfigHolder{
		Config: config,
	}
	holders = append(holders, holder)
	return holders
}

func (p *PodAnnotationBuilder) getMetricType(pod *corev1.Pod) string {
	return kubecommon.GetAnnotationWithPrefix(metrictype, p.Prefix, pod)
}

func (p *PodAnnotationBuilder) getNamespace(pod *corev1.Pod) string {
	return kubecommon.GetAnnotationWithPrefix(namespace, p.Prefix, pod)
}

func (p *PodAnnotationBuilder) isNamespaceRequired(module string) bool {
	if module == "prometheus" || module == "jolokia" || module == "dropwizard" || module == "http" {
		return true
	}
	return false
}

func (p *PodAnnotationBuilder) getEndpoints(ip string, pod *corev1.Pod) []string {
	endpointStr := kubecommon.GetAnnotationWithPrefix(endpoints, p.Prefix, pod)
	eps := strings.Split(endpointStr, ",")

	scheme := p.getScheme(pod)
	if scheme != "" {
		ip = scheme + "://" + ip
	}
	output := []string{}

	for _, ep := range eps {
		ep = strings.TrimSpace(ep)
		if ep != "" {
			output = append(output, fmt.Sprintf("%s%s", ip, ep))
		}
	}

	return output
}

func (p *PodAnnotationBuilder) getMetricSets(key string, pod *corev1.Pod) []string {
	msetStr := kubecommon.GetAnnotationWithPrefix(metricsets, p.Prefix, pod)
	msets := strings.Split(msetStr, ",")

	registeredSets := mb.Registry.MetricSets(key)
	if len(msetStr) == 0 {
		if key == "prometheus" {
			return []string{"collector"}
		}
		return registeredSets
	} else {
		output := []string{}
		for _, mset := range msets {
			output = append(output, strings.TrimSpace(mset))
		}

		return output
	}
}

func (p *PodAnnotationBuilder) getInterval(pod *corev1.Pod) time.Duration {
	i := kubecommon.GetAnnotationWithPrefix(interval, p.Prefix, pod)

	var dur time.Duration
	if i != "" {
		dur, err := time.ParseDuration(i)
		if err == nil {
			return dur
		}
	}

	dur = default_interval
	return dur
}

func (p *PodAnnotationBuilder) getTimeout(pod *corev1.Pod) time.Duration {
	t := kubecommon.GetAnnotationWithPrefix(timeout, p.Prefix, pod)
	var dur time.Duration
	if t != "" {
		dur, err := time.ParseDuration(t)
		if err == nil {
			return dur
		}
	}

	dur = default_timeout
	return dur
}

func (p *PodAnnotationBuilder) getScheme(pod *corev1.Pod) string {
	return kubecommon.GetAnnotationWithPrefix(scheme, p.Prefix, pod)
}

func (p *PodAnnotationBuilder) getInsecureSkipVerify(pod *corev1.Pod) bool {
	verifyStr := kubecommon.GetAnnotationWithPrefix(insecure_skip_verify, p.Prefix, pod)

	verify, _ := strconv.ParseBool(verifyStr)
	return verify
}
