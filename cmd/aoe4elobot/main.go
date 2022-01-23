package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"

	"github.com/alexisgeoffrey/aoe4elobot/internal/discordapi"
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.Parse()
}

var token string

func main() {
	if token == "" {
		fmt.Println("No token provided.")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Printf("Error creating Discord session: %v\n", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(discordapi.MessageCreate)

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMessages

	dg.LogLevel = 2

	c := cron.New()
	c.AddFunc("@midnight", func() {
		fmt.Println("Running scheduled Elo update.")

		dg.State.RLock()
		defer dg.State.RUnlock()

		guilds := dg.State.Guilds
		for _, g := range guilds {
			discordapi.UpdateAllElo(dg, g.ID)
		}
	})

	// Open a websocket connection to Discord and begin listening.
	if err := dg.Open(); err != nil {
		fmt.Printf("error opening connection to Discord: %v\n", err)
		return
	}

	c.Start()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("AOE4 Elo Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Cron job and Discord session.
	fmt.Println("Shutting down...")
	c.Stop()
	dg.Close()
}