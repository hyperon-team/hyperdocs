package sources

import (
	"context"
	"hyperdocs/config"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
)

// Source is the representation of a single documentation source.
type Source interface {
	// Name of the resource. It is being used as the source command name
	Name() string
	// Source description. It is set as the source command description
	Description() string
	// Source command options
	Options() []*discordgo.ApplicationCommandOption

	// Process is a hook to process and prepare data for the Search function
	Process(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
	// Search processes the input and returns the symbol by specified parameters.
	Search(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (Symbol, error)
}

// Sources is a list of constructors for of all supported documentation sources
var SourcesRaw = []func(config.Config, *redis.Client) Source{
	NewDiscord,
}

// Sources returns a map of all supported documentation sources
func Sources(cfg config.Config, redisClient *redis.Client) (r []Source) {
	for _, s := range SourcesRaw {
		r = append(r, s(cfg, redisClient))
	}
	return
}
