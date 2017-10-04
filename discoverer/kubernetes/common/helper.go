package common

import (
	"fmt"

	corev1 "github.com/ericchiang/k8s/api/v1"
)

func GetAnnotation(key string, pod *corev1.Pod) string {
	annotations := pod.GetMetadata().GetAnnotations()

	if annotations == nil {
		return ""
	}

	value, ok := annotations[key]
	if ok {
		return value
	}

	return ""
}

func GetAnnotationWithPrefix(key, prefix string, pod *corev1.Pod) string {
	return GetAnnotation(fmt.Sprintf("%s%s", prefix, key), pod)
}

func GetPodIp(pod *corev1.Pod) string {
	ip := pod.Status.GetPodIP()
	return ip
}

func GetPodPhase(pod *corev1.Pod) string {
	phase := pod.Status.GetPhase()
	return phase
}
