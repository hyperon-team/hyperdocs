package discordgoutil

import "github.com/bwmarrin/discordgo"

// OptionsToMap converts application command options to a map.
func OptionsToMap(options []*discordgo.ApplicationCommandInteractionDataOption) (res map[string]*discordgo.ApplicationCommandInteractionDataOption) {
	res = make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))

	for _, opt := range options {
		res[opt.Name] = opt
	}
	return
}
