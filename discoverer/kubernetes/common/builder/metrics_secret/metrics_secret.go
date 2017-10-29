package metrics_secret

import (
	"context"
	"fmt"
	"strings"
	"time"

	dcommon "github.com/ebay/collectbeat/discoverer/common"
	"github.com/ebay/collectbeat/discoverer/common/builder"
	"github.com/ebay/collectbeat/discoverer/common/registry"
	kubecommon "github.com/ebay/collectbeat/discoverer/kubernetes/common"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/ebay/collectbeat/discoverer/common/metagen"
	"github.com/ericchiang/k8s"

	kubernetes "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

const (
	secret_name = "config"
	host        = "$HOST"

	default_prefix   = "io.collectbeat.metrics/"
	default_timeout  = time.Second * 3
	default_interval = time.Minute

	SecretsBuilder = "metrics_secret"
)

func init() {
	registry.BuilderRegistry.AddBuilder(SecretsBuilder, NewSecretBuilder)
}

// PodAnnotationBuilder implements default modules based on pod annotations
type SecretBuilder struct {
	Prefix string
	client *k8s.Client
	ctx    context.Context
	meta   metagen.MetaGen
}

func NewSecretBuilder(cfg *common.Config, clientInfo builder.ClientInfo, meta metagen.MetaGen) (builder.Builder, error) {
	config := struct {
		Prefix string `config:"prefix"`
	}{
		Prefix: default_prefix,
	}

	err := cfg.Unpack(&config)

	//Add . to the end of the annotation namespace
	if config.Prefix[len(config.Prefix)-1] != '/' {
		config.Prefix = config.Prefix + "/"
	}

	if err != nil {
		return nil, fmt.Errorf("fail to unpack the `secrets` builder configuration: %s", err)
	}

	ctx := context.Background()

	var client *k8s.Client
	if clientRaw, ok := clientInfo[kubecommon.ClientKey]; ok {
		if client, ok = clientRaw.(*k8s.Client); !ok {
			return nil, fmt.Errorf("client is not of type *k8s.Client")
		}
	} else {
		return nil, fmt.Errorf("unable to get kube-client from ClientInfo")
	}

	return &SecretBuilder{Prefix: config.Prefix, client: client, ctx: ctx, meta: meta}, nil
}

func (s *SecretBuilder) Name() string {
	return "Secret Builder"
}

func (s *SecretBuilder) BuildModuleConfigs(obj interface{}) []*dcommon.ConfigHolder {
	holders := []*dcommon.ConfigHolder{}

	pod, ok := obj.(*kubernetes.Pod)
	if !ok {
		logp.Err("Unable to cast %v to type *v1.Pod", obj)
		return holders
	}

	ip := kubecommon.GetPodIp(pod)
	if ip == "" {
		return holders
	}

	secretName := kubecommon.GetAnnotation(fmt.Sprintf("%s%s", s.Prefix, secret_name), pod)
	if secretName == "" {
		return holders
	}

	secret, err := s.client.CoreV1().GetSecret(s.ctx, secretName, pod.Metadata.Namespace)
	if err != nil {
		logp.Err("Unable to get secret %s from namespace %s due to error %v", secretName,
			pod.Metadata.Namespace, err)
		return holders
	}

	data := secret.GetData()
	modulesYaml, ok := data["modules"]
	if !ok {
		return holders
	}

	modulesCfg, err := common.NewConfigWithYAML(modulesYaml, "")
	if err != nil {
		return holders
	}

	modules := []*common.Config{}
	modulesCfg.Unpack(&modules)

	for _, module := range modules {
		mCfg := &mb.ModuleConfig{}
		module.Unpack(mCfg)

		s.applyHostIps(ip, mCfg)
		s.applyDuration(mCfg)
		s.applyTimeout(mCfg)

		module.Merge(*mCfg)

		rawModule := map[string]interface{}{}
		err := module.Unpack(rawModule)
		if err != nil {
			logp.Err("Unable to parse config object due to error: ", err)
			continue
		}

		if s.meta != nil && len(mCfg.Hosts) != 0 {
			meta := s.meta.GetMetaData(mCfg.Hosts[0])
			kubecommon.SetKubeMetadata(meta, rawModule)
		}

		module := common.MapStr(rawModule)
		holder := &dcommon.ConfigHolder{
			Config: module,
		}
		holders = append(holders, holder)
	}
	return holders
}

func (s *SecretBuilder) applyHostIps(ip string, module *mb.ModuleConfig) {
	for i := 0; i < len(module.Hosts); i++ {
		module.Hosts[i] = strings.Replace(module.Hosts[i], host, ip, 1)
	}
}

func (s *SecretBuilder) applyTimeout(module *mb.ModuleConfig) {
	if module.Timeout.Seconds() == 0 {
		module.Timeout = default_timeout
	}
}

func (s *SecretBuilder) applyDuration(module *mb.ModuleConfig) {
	if module.Period.Seconds() == 0 {
		module.Period = default_interval
	}
}
