package sources

import (
	"context"

	"github.com/bwmarrin/discordgo"
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

// SourcesList is a map of supported all documentation sources
var Sources = []Source{
	Discord{},
}
