package common

type Config struct {
	TLS     *TLSConfig `config:"ssl"`
	Host    string     `config:"host"`
	RootDir string     `config:"root_dir"`
}

type TLSConfig struct {
	Enabled     *bool  `config:"enabled"`
	CA          string `config:"certificate_authority"`
	Certificate string `config:"certificate"`
	Key         string `config:"key"`
}

func (c *TLSConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

func DefaultDockerConfig() Config {
	return Config{
		Host:    "unix:///var/run/docker.sock",
		RootDir: "/var/lib/docker",
	}
}
