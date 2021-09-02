package discordgoutil

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// InteractionHandlerMap is an alias to a map of discordgo.InteractionCreate event handlers.
// Subcommands are space separated. For example: "tic tac toe" - that would be command "tic", with subcommand group "tac" and subcommand "toe".
type InteractionHandlerMap = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)

func callCommand(handlers InteractionHandlerMap, s *discordgo.Session, i *discordgo.InteractionCreate, name string) bool {
	h, ok := handlers[name]
	if ok {
		h(s, i)
	}
	return ok
}

// NewCommandHandler constructs interaction event handler for specified command handlers.
func NewCommandHandler(commandHandlers InteractionHandlerMap) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()

		if len(data.Options) > 0 {
			switch data.Options[0].Type {
			case discordgo.ApplicationCommandOptionSubCommand:
				callCommand(commandHandlers, s, i, data.Name+" "+data.Options[0].Name)
			case discordgo.ApplicationCommandOptionSubCommandGroup:
				callCommand(commandHandlers, s, i, strings.Join([]string{data.Name, data.Options[0].Name, data.Options[0].Options[0].Name}, " "))
			}
		}

		if callCommand(commandHandlers, s, i, data.Name) {
			return
		}
	}
}
