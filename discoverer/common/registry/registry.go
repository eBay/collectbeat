package registry

import (
	"errors"
	"sync"

	"fmt"

	"github.com/ebay/collectbeat/discoverer/common/appender"
	"github.com/ebay/collectbeat/discoverer/common/builder"

	"github.com/elastic/beats/libbeat/common"
	p "github.com/elastic/beats/libbeat/plugin"
)

// BuilderRegistry is the singleton Register instance where all Builders and Matchers
// are stored
var BuilderRegistry = NewRegister()

// Register contains Builder to use on pod indexing and event matching
type Register struct {
	sync.RWMutex
	builders  map[string]builder.BuilderConstructor
	appenders map[string]appender.AppenderConstructor

	defaultBuilderConfigs  map[string]common.Config
	defaultAppenderConfigs map[string]common.Config
}

// NewRegister creates and returns a new Register.
func NewRegister() *Register {
	return &Register{
		builders:               make(map[string]builder.BuilderConstructor, 0),
		appenders:              make(map[string]appender.AppenderConstructor, 0),
		defaultBuilderConfigs:  make(map[string]common.Config, 0),
		defaultAppenderConfigs: make(map[string]common.Config, 0),
	}
}

// Add Builder to the register
func (r *Register) AddBuilder(name string, indexer builder.BuilderConstructor) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()
	r.builders[name] = indexer
}

// Get Builder from the register
func (r *Register) GetBuilder(name string) builder.BuilderConstructor {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()

	builder, _ := r.builders[name]
	return builder
}

// Add Appender to the register
func (r *Register) AddDefaultAppenderConfig(name string, config common.Config) {
	r.defaultAppenderConfigs[name] = config
}

// Add Appender to the register
func (r *Register) AddAppender(name string, appender appender.AppenderConstructor) {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()
	r.appenders[name] = appender
}

// Get Appender from the register
func (r *Register) GetAppender(name string) appender.AppenderConstructor {
	r.RWMutex.Lock()
	defer r.RWMutex.Unlock()

	appender, _ := r.appenders[name]
	return appender
}

// Add Builder to the register
func (r *Register) AddDefaultBuilderConfig(name string, config common.Config) {
	r.defaultBuilderConfigs[name] = config
}

func (r *Register) GetDefaultBuilderConfigs() map[string]common.Config {
	return r.defaultBuilderConfigs
}

func (r *Register) GetDefaultAppenderConfigs() map[string]common.Config {
	return r.defaultAppenderConfigs
}

var (
	builderKey  = "collectbeat.discovery.builder"
	appenderKey = "collectbeat.discovery.appender"
)

type builderPlugin struct {
	name        string
	constructor builder.BuilderConstructor
}

type appenderPlugin struct {
	name        string
	constructor appender.AppenderConstructor
}

func BuilderPlugin(name string, b builder.BuilderConstructor) map[string][]interface{} {
	return p.MakePlugin(builderKey, builderPlugin{name, b})
}

func AppenderPlugin(name string, a appender.AppenderConstructor) map[string][]interface{} {
	return p.MakePlugin(appenderKey, appenderPlugin{name, a})
}

func init() {
	p.MustRegisterLoader(builderKey, func(ifc interface{}) error {
		m, ok := ifc.(builderPlugin)
		if !ok {
			return errors.New("plugin does not match builder plugin type")
		}

		name := m.name
		fmt.Println(name)
		if BuilderRegistry.GetBuilder(name) != nil {
			return fmt.Errorf("builder type %v already registered", name)
		}

		BuilderRegistry.AddBuilder(name, m.constructor)
		return nil
	})

	p.MustRegisterLoader(appenderKey, func(ifc interface{}) error {
		m, ok := ifc.(appenderPlugin)
		if !ok {
			return errors.New("plugin does not match appender plugin type")
		}

		name := m.name
		if BuilderRegistry.GetAppender(name) != nil {
			return fmt.Errorf("appender type %v already registered", name)
		}

		BuilderRegistry.AddAppender(name, m.constructor)
		return nil
	})
}
