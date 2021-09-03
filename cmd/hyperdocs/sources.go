package main

import (
	"context"
	"errors"
	"fmt"
	"hyperdocs/internal/sources"
	discordgoutil "hyperdocs/pkg/discordgo"

	"github.com/bwmarrin/discordgo"
)

func newCommandFromSource(src sources.Source) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        src.Name(),
		Description: src.Description(),
		Options:     src.Options(),
	}
}

func optionsFromSourceList(list []sources.Source) (res []*discordgo.ApplicationCommandOption) {
	for _, v := range list {
		res = append(res, newCommandFromSource(v))
	}
	return
}

func makeInteractionHandler(src sources.Source) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		var symbol sources.Symbol
		var err error
		src.Process(context.TODO(), s, i)
		symbol, err = src.Search(context.TODO(), s, i)

		if errors.Is(err, sources.ErrSymbolNotFound) {
			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "**Error: nothing had been found by this query**",
					Flags:   1 << 6,
				},
			})
			if err != nil {
				fmt.Println(err)
			}
			return
		}
		desc, fields := symbol.Render()
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       symbol.GetName(),
						URL:         symbol.GetLink(),
						Description: desc,
						Fields:      fields,
						Color:       0x0F0D0D,
					},
				},
			},
		})

		if err != nil {
			fmt.Println(err)
		}

		// TODO: context

		// ctx, cancel := context.WithTimeout(context.Background(), discordgo.InteractionDeadline-time.Second)

		// var symbol sources.Symbol, err error

		// go func() {
		// 	src.Process(ctx, s, i)
		// 	symbol, err = src.Search(ctx, s, i)
		// 	cancel()
		// }()

		// <-ctx.Done()
		// if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		// 	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// 		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		// 	})

		// 	if err != nil {
		// 		fmt.Println(err) // TODO: logrus
		// 	}
		// } else {
		// 	s.InteractionRespond()
		// }
		// case <-end:
		// 	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		// 			} else {

		// 	}
		// }
	}
}

func makeHandlersMap(sourcesList []sources.Source) (handlerMap discordgoutil.InteractionHandlerMap) {
	handlerMap = make(discordgoutil.InteractionHandlerMap, len(sourcesList))
	for _, src := range sourcesList {
		handlerMap["docs"+" "+src.Name()] = makeInteractionHandler(src)
	}
	return
}
