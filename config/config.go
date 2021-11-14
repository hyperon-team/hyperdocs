package config

import (
	"time"
)

type CacheConfig struct {
	Addr     string
	Password string
	DB       int
}

type SourcesConfig struct {
	Discord struct {
		// TTL of discord docs sources (in seconds)
		SourceCacheTTL time.Duration `envconfig:"ttl" default:"1m"`
	}
}

type Config struct {
	Token        string `required:"true"`
	DiscordID    string `envconfig:"discord_id" required:"true"`
	TestingGuild string
	Cache        CacheConfig // CONFIG_CACHE_ADDR=redis CACHE_PASSWORD=youshallnotpass CACHE_USERNAME=root
	Sources      SourcesConfig
}
