package discoverer

import (
	"errors"
	"fmt"

	p "github.com/elastic/beats/libbeat/plugin"
)

var pluginKey = "collectbeat.discoverer"

type discovererPlugin struct {
	name   string
	bulker Constructor
}

func Plugin(name string, b Constructor) map[string][]interface{} {
	return p.MakePlugin(pluginKey, discovererPlugin{name, b})
}

func init() {
	p.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		b, ok := ifc.(discovererPlugin)
		if !ok {
			return errors.New("plugin does not match bulker plugin type")
		}

		name := b.name
		if discovererPlugins[name] != nil {
			return fmt.Errorf("bulker type %v already registered", name)
		}

		RegisterDiscovererPlugin(name, b.bulker)
		return nil
	})
}
