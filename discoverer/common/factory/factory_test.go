package factory

import (
	"testing"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestFactory(t *testing.T) {
	RegisterFactoryPlugin("foo", NewFakeFactory)

	_, ok := factoryPlugins["foo"]
	assert.True(t, ok)

	cfg, err := common.NewConfigFrom(map[string]interface{}{
		"name": "foo",
	})

	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	f, err := InitFactory(cfg, nil)
	assert.Nil(t, err)

	assert.NotNil(t, f)
	assert.Equal(t, "foo", f.Name)

	cfg.SetString("name", -1, "bar")
	f, err = InitFactory(cfg, nil)
	assert.NotNil(t, err)
	assert.Nil(t, f)
}

type fakeFactory struct{}

func (f *fakeFactory) Start(config []*dcommon.ConfigHolder) error {
	return nil
}

func (f *fakeFactory) Stop(config []*dcommon.ConfigHolder) error {
	return nil
}

func (f *fakeFactory) Restart(old, new *dcommon.ConfigHolder) error {
	return nil
}

func NewFakeFactory(_ *common.Config, _ Meta) (Factory, error) {
	return &fakeFactory{}, nil
}
