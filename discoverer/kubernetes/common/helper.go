package common

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

func GetAnnotation(key string, pod *kubernetes.Pod) string {
	annotations := pod.Metadata.Annotations

	if annotations == nil {
		return ""
	}

	value, ok := annotations[key]
	if ok {
		return value
	}

	return ""
}

func GetAnnotationWithPrefix(key, prefix string, pod *kubernetes.Pod) string {
	return GetAnnotation(fmt.Sprintf("%s%s", prefix, key), pod)
}

func GetPodIp(pod *kubernetes.Pod) string {
	ip := pod.Status.PodIP
	return ip
}

func GetPodPhase(pod *kubernetes.Pod) string {
	phase := pod.Status.Phase
	return phase
}

func SetKubeMetadata(kubemeta, config common.MapStr) {
	if kubemeta != nil {
		if _, ok := config["fields"]; !ok {
			config["fields"] = common.MapStr{
				"kubernetes": kubemeta,
			}
		} else {
			config["fields"].(common.MapStr)["kubernetes"] = kubemeta
		}
		config["fields_under_root"] = true
	}
}
