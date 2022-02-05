package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexisgeoffrey/aoe4elobot/internal/discordapi"
	"github.com/bwmarrin/discordgo"
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
		log.Printf("Error creating Discord session: %v\n", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(discordapi.MessageCreate)

	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildPresences |
		discordgo.IntentsGuildMessages

	dg.LogLevel = 2

	dg.UserAgent = discordapi.UserAgent

	// c := cron.New()
	// _, err = c.AddFunc("@midnight", func() {
	// 	log.Println("Running scheduled Elo update.")

	// 	for _, g := range dg.State.Guilds {
	// 		if updateMessage, err := discordapi.UpdateAllElo(dg, g.ID); err != nil {
	// 			log.Printf("error updating elo on server %s: %v\n", g.ID, err)
	// 		}
	// 		dg.ChannelMessageSend(g.Channels[0].ID)
	// 	}

	// 	log.Println("Scheduled Elo update complete.")
	// })
	// if err != nil {
	// 	log.Printf("error adding cron job: %v\n", err)
	// 	return
	// }

	// Open a websocket connection to Discord and begin listening.
	if err := dg.Open(); err != nil {
		log.Printf("error opening connection to Discord: %v\n", err)
		return
	}

	// c.Start()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("AOE4 Elo Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Cron job and Discord session.
	fmt.Println("Shutting down...")
	// c.Stop()
	dg.Close()
}
