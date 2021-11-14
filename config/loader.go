package config

import "github.com/kelseyhightower/envconfig"

func Load() (cfg Config, err error) {
	err = envconfig.Process("hyperdocs", &cfg)
	return
}
