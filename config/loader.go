package config

import (
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

func Load() (cfg Config, err error) {
	var prefix string = "hyperdocs"
	switch strings.ToLower(os.Getenv("HYPERDOCS_ENV")) {
	case "production":
		prefix = ""
	case "testing":
		prefix += "_test"
	}
	err = envconfig.Process(prefix, &cfg)
	return
}
