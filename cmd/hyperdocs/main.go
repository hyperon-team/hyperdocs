package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"

	"hyperdocs/config"
	"hyperdocs/internal/sources"
	discordgoutil "hyperdocs/pkg/discordgo"
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

func registerCommands(cfg config.Config, session *discordgo.Session, commands []*discordgo.ApplicationCommand) {
	for _, cmd := range commands {
		_, err := session.ApplicationCommandCreate(cfg.AppID, cfg.TestingGuild, cmd)
		if err != nil {
			log.Fatal(fmt.Errorf("cannot register %q command: %w", cmd.Name, err))
		}
	}
}

func init() {
	err := godotenv.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(fmt.Errorf("cannot load configuration: %w", err))
	}
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.Fatal(fmt.Errorf("cannot construct session: %w", err))
	}
	redisOpts, err := redis.ParseURL(cfg.Redis)
	if err != nil {
		log.Fatal(fmt.Errorf("cannot parse redis url: %w", err))
	}

	sourcesList := sources.Sources(cfg, redis.NewClient(redisOpts))

	registerCommands(cfg, session, []*discordgo.ApplicationCommand{
		{
			Name:        "docs",
			Description: "Open sesame the documentation vault.",
			Options:     optionsFromSourceList(sourcesList),
		},
		{
			Name:        "invite",
			Description: "Invite the bot",
		},
	})

	session.AddHandler(discordgoutil.NewCommandHandler(makeHandlersMap(sourcesList)))
	session.AddHandler(discordgoutil.NewCommandHandler(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"invite": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   1 << 6,
					Content: fmt.Sprintf(`To invite me - click [here](https://discord.com/api/oauth2/authorize?client_id=%s&permissions=378944&scope=bot+applications.commands)`, s.State.User.ID),
				},
			})
		},
	}))
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
