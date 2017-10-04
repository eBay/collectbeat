package graphite_annotations

import (
	"encoding/json"
	"testing"

	"github.com/ebay/collectbeat/discoverer/common/builder"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/graphite/server"

	"fmt"

	corev1 "github.com/ericchiang/k8s/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestGraphiteAnnotations(t *testing.T) {
	config, err := getBaseConfig(t)

	bRaw, err := NewGraphiteAnnotationBuilder(config, nil)
	assert.NotNil(t, bRaw)
	assert.Nil(t, err)

	b, ok := bRaw.(builder.PushBuilder)
	ok = assert.Equal(t, ok, true)
	if !ok {
		t.FailNow()
	}

	confs := b.ModuleConfig()
	assert.NotNil(t, confs)

	gConf := server.GraphiteServerConfig{}
	err = confs.Config.Unpack(&gConf)

	assert.Nil(t, err)
	assert.Equal(t, gConf.Protocol, "tcp")
	assert.Equal(t, gConf.DefaultTemplate.Namespace, "foo")
}

func TestAddModule(t *testing.T) {
	config, _ := getBaseConfig(t)
	bRaw, err := NewGraphiteAnnotationBuilder(config, nil)
	assert.NotNil(t, bRaw)
	assert.Nil(t, err)

	fmt.Println(bRaw)
	b, ok := bRaw.(builder.PushBuilder)
	assert.Equal(t, ok, true)

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
				"foo/template": "foo.metric*",
			},
			length: 0,
		},
		{
			annotations: map[string]interface{}{
				"foo/template":  "foo.metric*",
				"foo/namespace": "abc",
			},
			length: 0,
		},
		{
			annotations: map[string]interface{}{
				"foo/template":  "foo.metric*",
				"foo/namespace": "abc",
				"foo/filter":    ":foo*",
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

		data, _ := json.Marshal(iface)
		json.Unmarshal(data, pod)

		confs := b.AddModuleConfig(pod)

		gConf := server.GraphiteServerConfig{}
		err := confs.Config.Unpack(&gConf)
		assert.Nil(t, err)
		assert.Equal(t, len(gConf.Templates), test.length)
	}
}

func TestRemoveModule(t *testing.T) {
	config, _ := getBaseConfig(t)
	bRaw, err := NewGraphiteAnnotationBuilder(config, nil)
	assert.NotNil(t, bRaw)
	assert.Nil(t, err)

	b, ok := bRaw.(builder.PushBuilder)
	assert.Equal(t, ok, true)

	iface := map[string]interface{}{
		"metadata": map[string]interface{}{
			"namespace": "foo",
			"name":      "bar",
			"annotations": map[string]interface{}{
				"foo/template":  "foo.metric*",
				"foo/namespace": "abc",
				"foo/filter":    ":foo*",
			},
		},
		"status": map[string]interface{}{
			"podIP": "4.5.6.7",
		},
	}
	pod := &corev1.Pod{}
	data, _ := json.Marshal(iface)
	json.Unmarshal(data, pod)

	confs := b.AddModuleConfig(pod)

	gConf := server.GraphiteServerConfig{}
	err = confs.Config.Unpack(&gConf)
	assert.Nil(t, err)
	assert.Equal(t, len(gConf.Templates), 1)

	tests := []struct {
		annotations map[string]interface{}
		length      int
	}{
		{
			annotations: map[string]interface{}{},
			length:      1,
		},
		{
			annotations: map[string]interface{}{
				"foo/template": "foo.metric*",
			},
			length: 1,
		},
		{
			annotations: map[string]interface{}{
				"foo/template":  "foo.metric*",
				"foo/namespace": "abc",
			},
			length: 1,
		},
		{
			annotations: map[string]interface{}{
				"foo/template":  "foo.metric*",
				"foo/namespace": "abc",
				"foo/filter":    ":foo*",
			},
			length: 0,
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

		data, _ := json.Marshal(iface)
		json.Unmarshal(data, pod)

		confs := b.RemoveModuleConfig(pod)

		gConf := server.GraphiteServerConfig{}
		err := confs.Config.Unpack(&gConf)
		assert.Nil(t, err)
		assert.Equal(t, len(gConf.Templates), test.length)
	}
}

func getBaseConfig(t *testing.T) (*common.Config, error) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"prefix": "foo",
		"config": map[string]interface{}{
			"host":     "localhost",
			"port":     "2000",
			"protocol": "tcp",
			"default_template": map[string]interface{}{
				"filter":    "*",
				"namespace": "foo",
				"template":  "metric*",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return config, err
}
