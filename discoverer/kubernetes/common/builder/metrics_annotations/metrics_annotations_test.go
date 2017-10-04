package metrics_annotations

import (
	"encoding/json"
	"testing"

	"github.com/ebay/collectbeat/discoverer/common/builder"

	"github.com/elastic/beats/libbeat/common"

	corev1 "github.com/ericchiang/k8s/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestMetricsAnnotations(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"prefix": "foo",
	})
	if err != nil {
		t.Fatal(err)
	}

	bRaw, err := NewPodAnnotationBuilder(config, nil)
	assert.NotNil(t, bRaw)
	assert.Nil(t, err)

	pod := &corev1.Pod{}

	annotations := map[string]interface{}{}
	iface := map[string]interface{}{
		"metadata": map[string]interface{}{
			"namespace":   "foo",
			"name":        "bar",
			"annotations": annotations,
		},
	}

	data, _ := json.Marshal(iface)
	json.Unmarshal(data, pod)

	b, ok := bRaw.(builder.PollerBuilder)
	assert.Equal(t, ok, true)

	confs := b.BuildModuleConfigs(pod)
	assert.Equal(t, len(confs), 0)

	iface["status"] = map[string]interface{}{
		"podIP": "1.2.3.4",
	}

	data, _ = json.Marshal(iface)
	json.Unmarshal(data, pod)

	confs = b.BuildModuleConfigs(pod)
	assert.Equal(t, len(confs), 0)

	tests := []struct {
		annotations map[string]interface{}
		length      int
	}{
		{
			annotations: map[string]interface{}{},
			length:      0,
		},
		{
			annotations: map[string]interface{}{
				"foo/type": "prometheus",
			},
			length: 0,
		},
		{
			annotations: map[string]interface{}{
				"foo/type":      "prometheus",
				"foo/namespace": "abc",
			},
			length: 0,
		},
		{
			annotations: map[string]interface{}{
				"foo/type":      "prometheus",
				"foo/namespace": "abc",
				"foo/endpoints": ":8080",
			},
			length: 1,
		},
	}

	for _, test := range tests {
		iface := map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace":   "foo",
				"name":        "bar",
				"annotations": test.annotations,
			},
			"status": map[string]interface{}{
				"podIP": "4.5.6.7",
			},
		}
		pod := &corev1.Pod{}

		data, _ = json.Marshal(iface)
		json.Unmarshal(data, pod)

		confs = b.BuildModuleConfigs(pod)
		assert.Equal(t, len(confs), test.length)
	}
}
