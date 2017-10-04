package log_path

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/appender"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	dc "github.com/ebay/collectbeat/discoverer/docker/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/fsouza/go-dockerclient"
)

const (
	LogPath = "log_path"

	Aufs         string = "aufs"
	Overlay      string = "overlay"
	Overlay2     string = "overlay2"
	DeviceMapper string = "devicemapper"
)

func init() {
	registry.BuilderRegistry.AddAppender(LogPath, NewLogPathAppender)
}

type LogPathAppender struct {
	dockerClient *docker.Client
	rootDir      string
}

func NewLogPathAppender(cfg *common.Config) (appender.Appender, error) {
	config := dc.DefaultDockerConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	client, err := dc.NewDockerClient(config.Host, config)
	if err != nil {
		return nil, err
	}

	_, err = client.Info()
	if err != nil {
		return nil, err
	}

	return &LogPathAppender{
		dockerClient: client,
	}, nil
}

func (l *LogPathAppender) Append(configHolder *dcommon.ConfigHolder) {
	// There are no custom log paths
	if len(configHolder.Meta) == 0 {
		return
	}

	rawConfigs := []map[string]interface{}{}
	config := configHolder.Config
	if config == nil {
		return
	}

	err := config.Unpack(&rawConfigs)
	if err != nil {
		return
	}

	for key, value := range configHolder.Meta {
		container, err := l.dockerClient.InspectContainer(key)
		if err != nil {
			logp.Err("Unable to get container info for container %s due to error: %v", key, err)
			continue
		}

		paths, _ := value.([]string)
		if len(paths) == 0 {
			continue
		}
		driver := container.Driver
		switch driver {
		case Overlay, Overlay2:
			if container.GraphDriver.Name == driver {
				mergedDir, ok := container.GraphDriver.Data["MergedDir"]
				if ok {
					appendDockerStoragePath(rawConfigs, paths, mergedDir)
				}
			}
		case Aufs:
			if container.GraphDriver.Name == driver {
				imagePath := fmt.Sprintf("%s/image/aufs/layerdb/mounts/%s/mount-id", l.rootDir, key)
				if _, err = os.Stat(imagePath); os.IsExist(err) {
					bytes, err := ioutil.ReadFile(imagePath)
					if err != nil {
						logp.Err("Unable to read file %s due to error %v", imagePath, err)
						continue
					}

					if len(bytes) == 0 {
						logp.Err("Unable to find filesystem id for container %s", key)
						continue
					}
					fsId := string(bytes)
					rootFs := fmt.Sprintf("%s/aufs/mnt/%s", l.rootDir, fsId)
					appendDockerStoragePath(rawConfigs, paths, rootFs)
				}
			}
		case DeviceMapper:
			if container.GraphDriver.Name == driver {
				deviceName, ok := container.GraphDriver.Data["DeviceName"]
				if ok {
					deviceNameParts := strings.Split(deviceName, "-")
					if len(deviceNameParts) > 0 {
						fsId := deviceNameParts[len(deviceNameParts)-1]
						rootFs := fmt.Sprintf("%s/devicemapper/mnt/%s/rootfs", l.rootDir, fsId)
						appendDockerStoragePath(rawConfigs, paths, rootFs)
					}
				}
			}
		default:
			logp.Err("Unsupported driver %s", driver)
		}
	}

	newCfg, err := common.NewConfigFrom(rawConfigs)
	if err != nil {
		return
	}

	configHolder.Config = newCfg
}
func appendDockerStoragePath(rawConfigs []map[string]interface{}, paths []string, mergedDir string) {
	for i := 0; i < len(rawConfigs); i++ {
		rawPaths, _ := rawConfigs[i]["paths"].([]interface{})
		pathConf := []string{}

		for _, rawPath := range rawPaths {
			pathConf = append(pathConf, fmt.Sprint(rawPath))
		}
		if reflect.DeepEqual(pathConf, paths) == true {
			for j := 0; j < len(pathConf); j++ {
				pathConf[j] = fmt.Sprintf("%s%s", mergedDir, pathConf[j])
			}
			rawConfigs[i]["paths"] = pathConf
		}
	}
}
