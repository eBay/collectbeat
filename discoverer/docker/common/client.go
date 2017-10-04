package common

import (
	"github.com/fsouza/go-dockerclient"
)

func NewDockerClient(endpoint string, config Config) (*docker.Client, error) {
	var err error
	var client *docker.Client

	if !config.TLS.IsEnabled() {
		client, err = docker.NewClient(endpoint)
	} else {
		client, err = docker.NewTLSClient(
			endpoint,
			config.TLS.Certificate,
			config.TLS.Key,
			config.TLS.CA,
		)
	}
	if err != nil {
		return nil, err
	}

	return client, nil
}
