package log_annotations

import (
	"fmt"
	"strconv"
	"strings"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/builder"
	"github.com/ebay/collectbeat/discoverer/common/metagen"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	kubecommon "github.com/ebay/collectbeat/discoverer/kubernetes/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
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
	baseConfig          common.MapStr
	metadata            metagen.MetaGen
}

func NewPodLogAnnotationBuilder(cfg *common.Config, _ builder.ClientInfo, meta metagen.MetaGen) (builder.Builder, error) {
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
		metadata:         meta,
	}, nil
}

func (l *PodLogAnnotationBuilder) Name() string {
	return "Log Annotation Builder"
}

func (l *PodLogAnnotationBuilder) BuildModuleConfigs(obj interface{}) []*dcommon.ConfigHolder {
	holders := []*dcommon.ConfigHolder{}
	pod, ok := obj.(*kubernetes.Pod)
	if !ok {
		logp.Err("Unable to cast %v to type *v1.Pod", obj)
		return holders
	}

	debug("Entering pod %s for logs annotations builder", pod.Metadata.Name)

	// Don't spin up a prospector unless pod goes into running state
	if kubecommon.GetPodIp(pod) == "" && kubecommon.GetPodPhase(pod) != "Running" {
		return holders
	}

	ns := l.getNamespace(pod)
	for _, container := range pod.Status.ContainerStatuses {
		name := container.Name
		meta := dcommon.Meta{}
		containerConfig := l.baseConfig.Clone()

		cid := container.ContainerID
		var cmeta common.MapStr
		var path string
		if cid != "" {
			parts := strings.Split(cid, "//")
			if len(parts) == 2 {
				cid = parts[1]
				path = fmt.Sprintf("%s%s/*.log", l.logsPath, cid)

			}
			if l.metadata != nil {
				cmeta = l.metadata.GetMetaData(cid)
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
		}

		if len(paths) == 0 {
			// Set json only when listening to stdout
			setJsonLog(containerConfig)
			containerConfig["paths"] = []string{path}
		} else if len(paths) != 0 {
			containerConfig["paths"] = paths
			meta[cid] = paths
		}
		setNamespace(ns, containerConfig)
		if cmeta != nil {
			kubecommon.SetKubeMetadata(cmeta, containerConfig)
		}

		holder := &dcommon.ConfigHolder{
			Config: containerConfig,
			Meta:   meta,
		}
		holders = append(holders, holder)
		debug("Config for pod %s, container %s is %v", pod.Metadata.Name, name, containerConfig)
	}

	return holders
}

func (l *PodLogAnnotationBuilder) getNamespace(pod *kubernetes.Pod) string {
	ns := kubecommon.GetAnnotationWithPrefix(namespace, l.prefix, pod)
	if ns == "" {
		return l.defaultNamespace
	}

	return ns
}

func (l *PodLogAnnotationBuilder) getPattern(pod *kubernetes.Pod, container string) string {
	return l.getAnnotationWithPrefixForContainer(pattern, container, pod)
}

func (l *PodLogAnnotationBuilder) getNegate(pod *kubernetes.Pod, container string) bool {
	negateStr := l.getAnnotationWithPrefixForContainer(negate, container, pod)
	negateBool, _ := strconv.ParseBool(negateStr)

	return negateBool
}

func (l *PodLogAnnotationBuilder) getMatch(pod *kubernetes.Pod, container string) string {
	matchStr := l.getAnnotationWithPrefixForContainer(match, container, pod)
	if matchStr == "" {
		return "after"
	}

	return matchStr
}

func (l *PodLogAnnotationBuilder) getPaths(pod *kubernetes.Pod, container string) []string {
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

func (l *PodLogAnnotationBuilder) getAnnotationWithPrefixForContainer(key, container string, pod *kubernetes.Pod) string {
	if container == "" {
		return kubecommon.GetAnnotationWithPrefix(key, l.prefix+"/", pod)
	}

	return kubecommon.GetAnnotationWithPrefix(key, l.prefix+"."+container+"/", pod)
}

func defaultBaseProspectorConfig() common.MapStr {
	base := common.MapStr{
		"type":    "log",
		"enabled": true,
	}

	return base
}

func setNamespace(ns string, config common.MapStr) {
	if ns != "" {
		if _, ok := config["fields"]; !ok {
			config["fields"] = common.MapStr{
				"namespace": ns,
			}
		} else {
			config["fields"].(common.MapStr)["namespace"] = ns
		}
		config["fields_under_root"] = true
	}
}

func setMultilineConfig(config common.MapStr, pattern string, negate bool, match string) {
	config["multiline"] = common.MapStr{
		"pattern": pattern,
		"negate":  negate,
		"match":   match,
	}
}
func setJsonLog(containerConfig common.MapStr) {
	containerConfig["json"] = common.MapStr{
		"message_key":     "log",
		"keys_under_root": true,
	}
}
