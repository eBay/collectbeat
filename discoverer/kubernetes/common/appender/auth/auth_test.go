package auth

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"io/ioutil"
	"os"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/stretchr/testify/assert"
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

	c, err := common.NewConfigFrom(map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"test"},
		"namespace":  "bar",
	})
	if err != nil {
		t.Fatal(err)
	}

	h := &dcommon.ConfigHolder{Config: c}

	sec.Append(h)
	ok := h.Config.HasField("headers")
	assert.Equal(t, ok, false)

	writeFile("test", "foo bar")
	sec.Append(h)
	ok = h.Config.HasField("headers")
	assert.Equal(t, ok, true)

	new, err := h.Config.Child("headers", -1)
	assert.Nil(t, err)

	obj := map[string]interface{}{}
	new.Unpack(obj)

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
