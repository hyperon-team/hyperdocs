package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

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

func registerCommands(session *discordgo.Session, commands []*discordgo.ApplicationCommand) {
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

	registerCommands(session, []*discordgo.ApplicationCommand{
		{
			Name:        "docs",
			Description: "Open sesame the documentation vault.",
			Options:     optionsFromSourceList(sources.Sources),
		},
	})

	session.AddHandler(discordgoutil.NewCommandHandler(makeHandlersMap(sources.Sources)))
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
