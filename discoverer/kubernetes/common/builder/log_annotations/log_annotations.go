package log_annotations

import (
	"fmt"
	"strconv"
	"strings"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/builder"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	kubecommon "github.com/ebay/collectbeat/discoverer/kubernetes/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	corev1 "github.com/ericchiang/k8s/api/v1"
)

const (
	namespace = "namespace"
	pattern   = "pattern"
	negate    = "negate"
	match     = "after"
	paths     = "paths"

	default_prefix = "io.collectbeat.logs"

	LogAnnotationsBuilder = "log_annotations"
)

var (
	debug = logp.MakeDebug(LogAnnotationsBuilder)
)

func init() {
	registry.BuilderRegistry.AddBuilder(LogAnnotationsBuilder, NewPodLogAnnotationBuilder)
}

// PodLogAnnotationBuilder implements default modules based on pod annotations
type PodLogAnnotationBuilder struct {
	prefix              string
	logsPath            string
	defaultNamespace    string
	enableCustomLogPath bool
	baseConfig          *common.Config
}

func NewPodLogAnnotationBuilder(cfg *common.Config, _ builder.ClientInfo) (builder.Builder, error) {
	config := DefaultLogPathConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the `logs_annotations` builder configuration: %s", err)
	}

	return &PodLogAnnotationBuilder{
		prefix:           config.Prefix,
		baseConfig:       config.BaseProspectorConfig,
		logsPath:         config.LogsPath,
		defaultNamespace: config.DefaultNamespace,
	}, nil
}

func (l *PodLogAnnotationBuilder) Name() string {
	return "Log Annotation Builder"
}

func (l *PodLogAnnotationBuilder) BuildModuleConfigs(obj interface{}) []*dcommon.ConfigHolder {
	holders := []*dcommon.ConfigHolder{}
	meta := dcommon.Meta{}
	rawConfigs := []map[string]interface{}{}
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		logp.Err("Unable to cast %v to type *v1.Pod", obj)
		return holders
	}

	debug("Entering pod %s for logs annotations builder", pod.Metadata.GetName())

	// Don't spin up a prospector unless pod goes into running state
	if kubecommon.GetPodIp(pod) == "" && kubecommon.GetPodPhase(pod) != "Running" {
		return holders
	}

	ns := l.getNamespace(pod)
	globalPaths := make([]string, 0)

	for _, container := range pod.GetStatus().GetContainerStatuses() {
		name := container.GetName()

		containerConfig := map[string]interface{}{}
		err := l.baseConfig.Unpack(containerConfig)
		if err != nil {
			logp.Err("Unable to unpack config for pod/container %s/%s due to error: %v", pod.GetMetadata().GetName(),
				name, err)
			return holders
		}

		cid := container.GetContainerID()
		var path string
		if cid != "" {
			parts := strings.Split(cid, "//")
			if len(parts) == 2 {
				cid = parts[1]
				path = fmt.Sprintf("%s%s/*.log", l.logsPath, cid)

			}
		} else {
			continue
		}

		var paths []string
		if l.enableCustomLogPath {
			paths = l.getPaths(pod, name)
		}

		containerPattern := l.getPattern(pod, name)
		if containerPattern != "" {
			containerMatch := l.getMatch(pod, name)
			containerNegate := l.getNegate(pod, name)

			setMultilineConfig(containerConfig, containerPattern, containerNegate, containerMatch)
			setNamespace(ns, containerConfig)
		}

		if len(paths) == 0 && containerPattern == "" {
			globalPaths = append(globalPaths, path)
			continue
		}

		if len(paths) == 0 {
			// Set json only when listening to stdout
			setJsonLog(containerConfig)
			containerConfig["paths"] = []string{path}
		} else if len(paths) != 0 {
			containerConfig["paths"] = paths
			meta[cid] = paths
		}

		rawConfigs = append(rawConfigs, containerConfig)
		debug("Config for pod %s, container %s is %v", pod.Metadata.GetName(), name, containerConfig)
	}

	if len(globalPaths) != 0 {
		globalConfig := map[string]interface{}{}
		err := l.baseConfig.Unpack(globalConfig)
		if err != nil {
			logp.Err("Unable to unpack config for pod %s due to error: %v", pod.GetMetadata().GetName(), err)
			return holders
		}

		// Add conditional when custom file path is implemented
		setJsonLog(globalConfig)

		globalConfig["paths"] = globalPaths
		globalPattern := l.getPattern(pod, "")

		if globalPattern != "" {
			globalNegate := l.getNegate(pod, "")
			globalMatch := l.getMatch(pod, "")

			setMultilineConfig(globalConfig, globalPattern, globalNegate, globalMatch)
		}

		setNamespace(ns, globalConfig)
		rawConfigs = append(rawConfigs, globalConfig)
		debug("Config for pod %s is %v", pod.Metadata.GetName(), globalConfig)
	}

	config, err := common.NewConfigFrom(rawConfigs)
	if err != nil {
		logp.Err("Unable to pack config due to error: %v", err)
		return holders
	}
	holder := &dcommon.ConfigHolder{
		Config: config,
		Meta:   meta,
	}
	holders = append(holders, holder)
	return holders
}

func (l *PodLogAnnotationBuilder) getNamespace(pod *corev1.Pod) string {
	ns := kubecommon.GetAnnotationWithPrefix(namespace, l.prefix, pod)
	if ns == "" {
		return l.defaultNamespace
	}

	return ns
}

func (l *PodLogAnnotationBuilder) getPattern(pod *corev1.Pod, container string) string {
	return l.getAnnotationWithPrefixForContainer(pattern, container, pod)
}

func (l *PodLogAnnotationBuilder) getNegate(pod *corev1.Pod, container string) bool {
	negateStr := l.getAnnotationWithPrefixForContainer(negate, container, pod)
	negateBool, _ := strconv.ParseBool(negateStr)

	return negateBool
}

func (l *PodLogAnnotationBuilder) getMatch(pod *corev1.Pod, container string) string {
	matchStr := l.getAnnotationWithPrefixForContainer(match, container, pod)
	if matchStr == "" {
		return "after"
	}

	return matchStr
}

func (l *PodLogAnnotationBuilder) getPaths(pod *corev1.Pod, container string) []string {
	if container == "" {
		return []string{}
	}

	pathStr := l.getAnnotationWithPrefixForContainer(paths, container, pod)
	paths := strings.Split(pathStr, ",")

	output := []string{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if len(path) != 0 {
			output = append(output, path)
		}
	}
	return output
}

func (l *PodLogAnnotationBuilder) getAnnotationWithPrefixForContainer(key, container string, pod *corev1.Pod) string {
	if container == "" {
		return kubecommon.GetAnnotationWithPrefix(key, l.prefix+"/", pod)
	}

	return kubecommon.GetAnnotationWithPrefix(key, l.prefix+"."+container+"/", pod)
}

func defaultBaseProspectorConfig() *common.Config {
	base := map[string]interface{}{
		"type":    "log",
		"enabled": true,
	}
	config, err := common.NewConfigFrom(base)
	if err != nil {
		return common.NewConfig()
	}

	return config
}

func setNamespace(ns string, config map[string]interface{}) {
	if ns != "" {
		if _, ok := config["fields"]; !ok {
			config["fields"] = map[string]interface{}{
				"namespace": ns,
			}
		} else {
			config["fields"].(map[string]interface{})["namespace"] = ns
		}
		config["fields_under_root"] = true
	}
}

func setMultilineConfig(config map[string]interface{}, pattern string, negate bool, match string) {
	config["multiline"] = map[string]interface{}{
		"pattern": pattern,
		"negate":  negate,
		"match":   match,
	}
}
func setJsonLog(containerConfig map[string]interface{}) {
	containerConfig["json"] = map[string]interface{}{
		"message_key":     "log",
		"keys_under_root": true,
	}
}
