package auth

import (
	"io/ioutil"
	"os"
	"testing"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestAuth(t *testing.T) {
	config, err := common.NewConfigFrom(map[string]interface{}{
		"namespaces": []string{"foo", "bar"},
		"token_path": "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	sec, err := NewSecurityAppender(config)
	assert.NotNil(t, sec)
	assert.Nil(t, err)

	c := common.MapStr{
		"module":     "prometheus",
		"metricsets": []string{"test"},
		"namespace":  "bar",
	}
	if err != nil {
		t.Fatal(err)
	}

	h := &dcommon.ConfigHolder{Config: c}

	sec.Append(h)
	_, ok := h.Config["headers"]
	assert.Equal(t, ok, false)

	writeFile("test", "foo bar")
	sec.Append(h)
	_, ok = h.Config["headers"]
	assert.Equal(t, ok, true)

	new, ok := h.Config["headers"]
	assert.Equal(t, ok, true)

	obj := new.(common.MapStr)

	header, ok := obj["Authorization"]
	assert.Equal(t, ok, true)
	assert.Equal(t, header, "Bearer foo bar")

	deleteFile("test")
}

func writeFile(name, message string) {
	ioutil.WriteFile(name, []byte(message), os.ModePerm)
}

func deleteFile(name string) {
	os.Remove(name)
}
