package cfgfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/factory"

	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/hashstructure"
)

var (
	debug = logp.MakeDebug("cfgfile_factory")
)

func init() {
	factory.RegisterFactoryPlugin("cfgfile", newCfgfileFactory)
}

type cfgfileFactory struct {
	cfgfiles cfgfileCache
	path     string
	prefix   string
}

type cfgfileCache struct {
	sync.Mutex
	cfgfiles map[uint64]*common.Config
}

func NewCfgfileCache() cfgfileCache {
	return cfgfileCache{
		cfgfiles: make(map[uint64]*common.Config),
	}
}

func newCfgfileFactory(cfg *common.Config, _ factory.Meta) (factory.Factory, error) {
	config := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("Unable to unpack config with error: %v", err)
	}

	if config.ReloaderConfig.Enabled() == false {
		return nil, fmt.Errorf("config.* needs to be enabled to use cfgfile factory")
	}
	reloadConfig := cfgfile.DefaultDynamicConfig
	err = config.ReloaderConfig.Unpack(&reloadConfig)

	if err != nil {
		return nil, fmt.Errorf("Unable to unpack reloader config with error: %v", err)
	}

	dir := filepath.Dir(reloadConfig.Path)

	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("Unable to create config dir directory")
		}
	}
	paths, err := filepath.Glob(dir)

	if err != nil {
		return nil, fmt.Errorf("Unable to determine reloader path due to error: %v", err)
	}

	if len(paths) > 1 {
		return nil, fmt.Errorf("%d paths determined from path glob. only one path is allowed", len(paths))
	}

	cfgFactory := &cfgfileFactory{
		cfgfiles: NewCfgfileCache(),
		path:     dir,
		prefix:   config.Prefix,
	}

	files, _ := filepath.Glob(fmt.Sprintf("%s/*", dir))
	for _, file := range files {
		cfgFactory.deleteFile(file)
	}

	return cfgFactory, nil
}

func (r *cfgfileFactory) Start(configHolder *dcommon.ConfigHolder) error {
	config := configHolder.Config
	if config == nil {
		return nil
	}
	r.cfgfiles.Lock()
	defer r.cfgfiles.Unlock()
	rawCfg := []map[string]interface{}{}
	err := config.Unpack(&rawCfg)
	if err != nil {
		return fmt.Errorf("Unable to unpack config due to error: %v", err)
	}

	debug("Current raw config coming in for creation: %v", rawCfg)

	if len(rawCfg) == 0 {
		return nil
	}

	hash, err := hashstructure.Hash(rawCfg, nil)
	if err != nil {
		return err
	}

	debug("Current hash coming in for creation: %s", hash)

	if _, ok := r.cfgfiles.cfgfiles[hash]; ok {
		return nil
	}
	bytes, err := yaml.Marshal(rawCfg)
	if err != nil {
		return err
	}
	if err != nil {
		return fmt.Errorf("Unable to pack config due to error: %v", err)
	}
	if len(bytes) == 0 {
		return nil
	}

	file := fmt.Sprintf("%s/%s%d.yml", r.path, r.prefix, hash)
	debug("Creating file %s with contents: %v", file, string(bytes))
	err = ioutil.WriteFile(file, bytes, 0644)
	if err != nil {
		return fmt.Errorf("Unable to write cfgfile due to error: %v", err)
	}

	r.cfgfiles.cfgfiles[hash] = config
	logp.Info("Deployed config file %d", hash)

	return nil
}

func (r *cfgfileFactory) Stop(configHolder *dcommon.ConfigHolder) error {
	r.cfgfiles.Lock()
	defer r.cfgfiles.Unlock()
	config := configHolder.Config
	if config == nil {
		return nil
	}

	rawCfg := []map[string]interface{}{}
	err := config.Unpack(&rawCfg)
	if err != nil {
		return fmt.Errorf("Unable to unpack config due to error: %v", err)
	}

	debug("Current raw config coming in for deletion: %v", rawCfg)

	if len(rawCfg) == 0 {
		return nil
	}

	hash, err := hashstructure.Hash(rawCfg, nil)
	if err != nil {
		return err
	}

	debug("Current hash coming in for deletion: %s", hash)

	if _, ok := r.cfgfiles.cfgfiles[hash]; !ok {
		debug("hash %s for deletion not found", hash)
		return nil
	}

	file := fmt.Sprintf("%s/%s%d.yml", r.path, r.prefix, hash)
	debug("File being deleted: %s", file)
	err = r.deleteFile(file)

	if err != nil {
		return err
	}

	delete(r.cfgfiles.cfgfiles, hash)
	logp.Info("Removed config file %d", hash)
	return nil
}

func (r *cfgfileFactory) Restart(old, new *dcommon.ConfigHolder) error {
	err := r.Stop(old)
	if err != nil {
		return err
	}

	err = r.Start(new)
	return err
}

func (r *cfgfileFactory) deleteFile(file string) error {
	f := path.Base(file)
	if strings.HasPrefix(f, r.prefix) == false || strings.HasSuffix(file, ".yml") == false {
		return nil
	}
	err := os.Remove(file)
	if err != nil {
		return fmt.Errorf("Unable to delete file %s due to error: %v", file, err)
	}

	return nil
}
