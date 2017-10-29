package runner

import (
	"testing"
	"time"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	pubtest "github.com/elastic/beats/libbeat/publisher/testing"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/module"
)

func TestRunnerStartStop(t *testing.T) {
	pubClient, f := newPubClientFactory()
	pipeline := pubtest.PublisherWithClient(f())

	config := common.MapStr{
		"module":     moduleName,
		"metricsets": []string{eventFetcherName},
	}
	holder := []*dcommon.ConfigHolder{
		{
			Config: config,
		},
	}

	fac := module.NewFactory(time.Second*1, pipeline)
	runner, err := newRunnerFactory(nil, fac)
	assert.Nil(t, err)

	err = runner.Start(holder)
	assert.NotNil(t, <-pubClient.Channel)
	assert.Nil(t, err)

	err = runner.Stop(holder)
	assert.Nil(t, err)

	_, err = newRunnerFactory(nil, nil)
	assert.NotNil(t, err)

	_, err = newRunnerFactory(nil, struct{}{})
	assert.NotNil(t, err)
}

func TestRunnerRestart(t *testing.T) {
	pubClient, f := newPubClientFactory()
	pipeline := pubtest.PublisherWithClient(f())

	config := common.MapStr{
		"module":     moduleName,
		"metricsets": []string{eventFetcherName},
	}
	holder := &dcommon.ConfigHolder{
		Config: config,
	}

	fac := module.NewFactory(time.Second*1, pipeline)
	runner, err := newRunnerFactory(nil, fac)
	assert.Nil(t, err)

	err = runner.Restart(holder, holder)
	assert.Nil(t, err)
	assert.NotNil(t, <-pubClient.Channel)

	err = runner.Restart(holder, holder)
	assert.Nil(t, err)

	err = runner.Stop([]*dcommon.ConfigHolder{holder})
	assert.Nil(t, err)
}

const (
	moduleName       = "fake"
	eventFetcherName = "EventFetcher"
)

func init() {
	if err := mb.Registry.AddMetricSet(moduleName, eventFetcherName, newFakeEventFetcher); err != nil {
		panic(err)
	}
}

type fakeEventFetcher struct {
	mb.BaseMetricSet
}

func (ms *fakeEventFetcher) Fetch() (common.MapStr, error) {
	t, _ := time.Parse(time.RFC3339, "2016-05-10T23:27:58.485Z")
	return common.MapStr{"@timestamp": common.Time(t), "metric": 1}, nil
}

func (ms *fakeEventFetcher) Close() error {
	return nil
}

func newFakeEventFetcher(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &fakeEventFetcher{BaseMetricSet: base}, nil
}

func newPubClientFactory() (*pubtest.ChanClient, func() beat.Client) {
	client := pubtest.NewChanClient(10)
	return client, func() beat.Client { return client }
}
