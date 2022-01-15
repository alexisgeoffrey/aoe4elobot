package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type Usernames struct {
	Usernames []Username `json:"usernames"`
}

type Username struct {
	DiscordUsername string `json:"discord_username"`
	SteamUsername   string `json:"steam_username"`
}

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.StringVar(&guildID, "g", "", "Guild ID")
	flag.Parse()
}

var (
	token   string
	guildID string
)

func main() {
	if token == "" {
		fmt.Println("No token provided.")
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	c := cron.New()
	c.AddFunc("@midnight", func() { updateELO(dg) })

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	c.Start()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("AOE4 ELO Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Cron job and Discord session.
	c.Stop()
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
}

func updateELO(s *discordgo.Session) {
	members, err := s.GuildMembers(guildID, "", 20)
	if err != nil {
		fmt.Println("error getting members: ", err)
		return
	}
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		fmt.Println("error getting roles: ", err)
		return
	}
	filteredRoles := make(map[string]bool, len(roles))
	for _, role := range roles {
		if strings.Contains(role.Name, "ELO:") {
			filteredRoles[role.ID] = true
		}
	}

	for _, member := range members {
		for _, roleID := range member.Roles { // remove existing roles
			if filteredRoles[roleID] {
				s.GuildMemberRoleRemove(guildID, member.User.ID, roleID)
			}
		}
	}
}
