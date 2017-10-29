package runner

import (
	"fmt"
	"sync"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/factory"
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	debug = logp.MakeDebug("runner_factory")
)

func init() {
	factory.RegisterFactoryPlugin("runner", newRunnerFactory)
}

type runnerFactory struct {
	factory cfgfile.RunnerFactory
	runners runnerCache
}

type runnerCache struct {
	sync.Mutex
	runners map[uint64]cfgfile.Runner
}

func NewRunnerCache() runnerCache {
	return runnerCache{
		runners: make(map[uint64]cfgfile.Runner),
	}
}

func newRunnerFactory(_ *common.Config, meta factory.Meta) (factory.Factory, error) {
	if meta == nil {
		return nil, fmt.Errorf("Unable to get config file runner factory")
	}

	if factory, ok := meta.(cfgfile.RunnerFactory); ok {
		return &runnerFactory{factory: factory, runners: NewRunnerCache()}, nil
	} else {
		return nil, fmt.Errorf("Unable to cast object to cfgfile.RunnerFactory")
	}
}

func (r *runnerFactory) Start(holders []*dcommon.ConfigHolder) error {
	for _, holder := range holders {
		config := holder.Config
		id := configHash(config)
		runner, err := r.buildModuleRunner(config)

		if err != nil {
			return err
		}
		r.runners.Lock()
		runner.Start()
		logp.Info("Starting runner %d", id)
		r.runners.runners[id] = runner
		r.runners.Unlock()
	}

	return nil
}

func (r *runnerFactory) Stop(holders []*dcommon.ConfigHolder) error {
	for _, holder := range holders {
		config := holder.Config
		id := configHash(config)

		r.runners.Lock()
		if run, ok := r.runners.runners[id]; ok {
			run.Stop()
			logp.Info("Stopping runner %d", id)
			delete(r.runners.runners, id)
		}
	}

	return nil
}

func (r *runnerFactory) Restart(oldHolder, newHolder *dcommon.ConfigHolder) error {
	old := oldHolder.Config
	oldID := configHash(old)
	new := newHolder.Config
	newID := configHash(new)

	// Do not restart the module if there is no change
	r.runners.Lock()
	_, ok := r.runners.runners[oldID]
	r.runners.Unlock()
	if oldID == newID && ok {
		debug("Not restarting as configs remain the same")
		return nil
	}

	r.runners.Lock()
	if run, ok := r.runners.runners[oldID]; ok {
		run.Stop()
		logp.Info("Stopping old runner %d", oldID)
		delete(r.runners.runners, oldID)
	}
	r.runners.Unlock()

	runner, err := r.buildModuleRunner(new)
	if err != nil {
		return err
	}

	if runner != nil {
		r.runners.Lock()
		logp.Info("Starting new runner %d", newID)
		runner.Start()
		r.runners.runners[newID] = runner
		r.runners.Unlock()
	}

	return nil
}

func (r *runnerFactory) buildModuleRunner(config common.MapStr) (cfgfile.Runner, error) {
	cfg := factory.GetConfigFromMapStr(config)
	if cfg == nil {
		return nil, fmt.Errorf("Unable to create module runner")
	}
	// Create a runner object
	runner, err := r.factory.Create(cfg)
	if err != nil {
		return nil, fmt.Errorf("Unable to create module runner due to error %v", err)
	}
	return runner, nil
}

func configHash(config common.MapStr) uint64 {
	hash, err := hashstructure.Hash(config, nil)
	if err != nil {
		debug("Error generating hash due to error: %v", err)
		return 0
	}

	return hash
}
