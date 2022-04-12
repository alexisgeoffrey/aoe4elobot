package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexisgeoffrey/aoe4elobot/internal/discordapi"
	"github.com/bwmarrin/discordgo"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/robfig/cron/v3"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yml"
	}

	err := cleanenv.ReadConfig(configPath, &discordapi.Config)
	if errors.Is(err, os.ErrNotExist) {
		genConfig(configPath)
		return
	} else if err != nil {
		log.Fatalf("error reading config file: %v\n", err)
	}

	// Open connection to user database
	discordapi.Db, err = pgxpool.Connect(context.Background(), discordapi.Config.DbUrl)
	if err != nil {
		log.Fatalf("error connecting to database: %v\n", err)
	}

	if err := setupDb(discordapi.Db); err != nil {
		log.Fatalf("error setting up database: %v\n", err)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + discordapi.Config.BotToken)
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
	// if _, err = c.AddFunc("0 */2 * * *", func() { eloUpdateCron(dg) }); err != nil {
	// 	log.Fatalf("error adding cron job: %v\n", err)
	// }
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
	discordapi.Db.Close()
}
