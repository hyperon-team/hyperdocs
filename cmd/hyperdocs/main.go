package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	discordgoutil "github.com/hyperon-team/hyperdocs/pkg/discordgo"
)

var (
	banner = `
 ______  __                                ________                       
 ___  / / /_____  __________ _____ ___________  __ \______ _______________
 __  /_/ / __  / / /___  __ \_  _ \__  ___/__  / / /_  __ \_  ___/__  ___/
 _  __  /  _  /_/ / __  /_/ //  __/_  /    _  /_/ / / /_/ // /__  _(__  ) 
 /_/ /_/   _\__, /  _  .___/ \___/ /_/     /_____/  \____/ \___/  /____/  
           /____/   /_/                                                   
                                                                          
	`
)

var discordDocTmpl = `https://discord.dev/{{ .topic.Value }}/{{ .page.Value }}{{ with $x := (index . "paragraph-id").Value }}#{{ $x }}{{ end }}`

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "docs",
			Description: "Search through documentation",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "discord",
					Description: "Discord API documentation",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "topic",
							Description: "Topic name",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "page",
							Description: "Page name",
							Type:        discordgo.ApplicationCommandOptionString,
							Required:    true,
						},
						{
							Name:        "paragraph-id",
							Description: "Id of the paragraph to retrieve",
							Type:        discordgo.ApplicationCommandOptionString,
						},
					},
				},
			},
		},
	}
	commandHandlers = discordgoutil.InteractionHandlerMap{
		"docs discord": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			optionsMap := map[string]*discordgo.ApplicationCommandInteractionDataOption{}
			for _, opt := range i.ApplicationCommandData().Options[0].Options {
				if opt.Type == discordgo.ApplicationCommandOptionString {
					nesting := strings.Split(opt.StringValue(), ":")
					for i, element := range nesting {
						nesting[i] = strings.Join(strings.Split(strings.TrimSpace(element), " "), "-")
					}

					opt.Value = strings.ToLower(strings.Join(nesting, "-"))
				}
				optionsMap[opt.Name] = opt

			}

			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: executeTemplate(discordDocTmpl, optionsMap),
				},
			})

			if err != nil {
				panic(err)
			}
		},
	}
)

func executeTemplate(tmpl string, data interface{}) string {

	buf := new(bytes.Buffer)
	err := template.Must(template.New("doc-url").Parse(tmpl)).Execute(buf, data)
	if err != nil {
		panic(fmt.Errorf("template: %w", err))
	}

	return buf.String()
}

func registerCommands(session *discordgo.Session) {
	guild := os.Getenv("DISCORD_GUILD")
	app := os.Getenv("DISCORD_ID")
	for _, cmd := range commands {
		_, err := session.ApplicationCommandCreate(app, guild, cmd)
		if err != nil {
			log.Fatal(fmt.Errorf("cannot register %q command: %w", cmd.Name, err))
		}
	}
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(fmt.Errorf("cannot load env file: %w", err))
	}
}

func awaitForInterrupt() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sigchan
}

func main() {
	fmt.Println(banner)
	session, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		panic(fmt.Errorf("cannot construct session: %w", err))
	}

	registerCommands(session)

	session.AddHandler(discordgoutil.NewCommandHandler(commandHandlers))
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Println("Bot is up!")
	})

	err = session.Open()
	if err != nil {
		panic(fmt.Errorf("gateway returned with a error: %w", err))
	}
	defer session.Close()

	awaitForInterrupt()
}
