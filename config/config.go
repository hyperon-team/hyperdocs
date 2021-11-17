package config

import (
	"time"
)

type SourcesConfig struct {
	Discord struct {
		// TTL of discord docs sources (in seconds)
		RedisTTL time.Duration `envconfig:"ttl" default:"1m"`
	}
}

type Config struct {
	Token        string `required:"true"`
	AppID        string `envconfig:"app_id" required:"true"`
	TestingGuild string
	Redis        string `envconfig:"redis_url" required:"true"`
	Sources      SourcesConfig
}
