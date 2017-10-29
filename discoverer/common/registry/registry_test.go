package registry

import (
	"testing"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/appender"
	"github.com/ebay/collectbeat/discoverer/common/builder"

	"github.com/stretchr/testify/assert"

	"github.com/ebay/collectbeat/discoverer/common/metagen"

	"github.com/elastic/beats/libbeat/common"
)

func TestRegistry(t *testing.T) {
	register := NewRegister()

	assert.Empty(t, register.builders)
	assert.Empty(t, register.appenders)

	register.AddBuilder("foo", newFakeBuilder)
	register.AddAppender("bar", newFakeAppender)

	assert.Len(t, register.builders, 1)
	assert.Len(t, register.appenders, 1)

	conf := common.NewConfig()
	register.AddDefaultBuilderConfig("foo", *conf)
	register.AddDefaultAppenderConfig("bar", *conf)

	assert.Len(t, register.defaultBuilderConfigs, 1)
	assert.Len(t, register.defaultAppenderConfigs, 1)

	foo := register.GetBuilder("foo")
	assert.NotNil(t, foo)

	fConf, ok := register.GetDefaultBuilderConfigs()["foo"]
	assert.True(t, ok)
	assert.NotNil(t, fConf)

	f, err := foo(&fConf, nil, nil)
	assert.NotNil(t, f)
	assert.Nil(t, err)

	bar := register.GetAppender("bar")
	assert.NotNil(t, bar)

	bConf, ok := register.GetDefaultAppenderConfigs()["bar"]
	assert.True(t, ok)
	assert.NotNil(t, bConf)

	b, err := foo(&bConf, nil, nil)
	assert.NotNil(t, b)
	assert.Nil(t, err)

	noBar := register.GetBuilder("bar")
	assert.Nil(t, noBar)

	noFoo := register.GetAppender("foo")
	assert.Nil(t, noFoo)
}

// Define a fake builder
type fakeBuilder struct{}

func (f *fakeBuilder) Name() string {
	return "fake_builder"
}

func newFakeBuilder(_ *common.Config, _ builder.ClientInfo, _ metagen.MetaGen) (builder.Builder, error) {
	return &fakeBuilder{}, nil
}

func (f *fakeBuilder) BuildModuleConfigs(obj interface{}) []*dcommon.ConfigHolder {
	return []*dcommon.ConfigHolder{}
}

// Define a fake appender
type fakeAppender struct{}

func (f *fakeAppender) Append(config *dcommon.ConfigHolder) {}

func newFakeAppender(_ *common.Config) (appender.Appender, error) {
	return &fakeAppender{}, nil
}
