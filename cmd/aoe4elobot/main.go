package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
	"github.com/alexisgeoffrey/aoe4elobot/internal/db"
	"github.com/alexisgeoffrey/aoe4elobot/internal/discordapi"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.Cfg.BotToken)
	if err != nil {
		log.Fatalf("error creating Discord session: %v\n", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(discordapi.MessageCreate)

	dg.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentGuildMembers |
		discordgo.IntentGuildPresences |
		discordgo.IntentGuildMessages

	dg.LogLevel = 2

	// Open a websocket connection to Discord and begin listening.
	if err := dg.Open(); err != nil {
		log.Fatalf("error opening connection to Discord: %v\n", err)
	}

	c := cron.New()
	if _, err = c.AddFunc("0 */2 * * *", func() { eloUpdateCron(dg) }); err != nil {
		log.Fatalf("error adding cron job: %v\n", err)
	}
	c.Start()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("AOE4 Elo Bot is now running. Press Ctrl-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Cron job and Discord session.
	fmt.Println("Shutting down...")
	c.Stop()
	dg.Close()
	db.Db.Close()
}
