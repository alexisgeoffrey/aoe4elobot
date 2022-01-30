package discordapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/alexisgeoffrey/aoe4api"
	"github.com/bwmarrin/discordgo"
)

type userElo map[string]string

var cmdMutex sync.Mutex

// MessageCreate is the handler for Discordgo MessageCreate events.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!setEloName") {
		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		name, err := saveToConfig(m.Content, m.Author.ID)
		if err != nil {
			s.ChannelMessageSendReply(
				m.ChannelID,
				"Your Steam username failed to update.\nUsage: `!setEloName Steam_username`",
				m.Reference())
			fmt.Printf("error updating username: %v\n", err)
			return
		}
		// Send response as a reply to message
		s.ChannelMessageSendReply(
			m.ChannelID,
			fmt.Sprintf("Steam username for %s has been updated to %s.", m.Author.Mention(), name),
			m.Reference())
	} else if strings.HasPrefix(m.Content, "!updateElo") {
		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		s.ChannelMessageSend(m.ChannelID, "Updating elo...")
		updateMessage, err := UpdateAllElo(s, m.GuildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Elo failed to update.")
			fmt.Printf("error updating elo: %v\n", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, updateMessage)
	}
}

// UpdateAllElo retrieves and updates all Elo roles on the server specified by the guildId
// parameter. It returns an update message containing all changed Elo values for each server member.
func UpdateAllElo(s *discordgo.Session, guildId string) (string, error) {
	fmt.Println("Updating Elo...")

	configBytes, err := configFileToBytes()
	if err != nil {
		return "", fmt.Errorf("error converting config file to bytes: %w", err)
	}

	var us users
	if err := json.Unmarshal(configBytes, &us); err != nil {
		return "", fmt.Errorf("error unmarshaling config bytes: %w", err)
	}

	for i, u := range us.Users {
		memberElo, err := u.getMemberElo(s.State, guildId)
		if err != nil {
			return "", fmt.Errorf("error retrieving existing member roles: %w", err)
		}
		us.Users[i].oldElo = memberElo
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	updatedElo := make(map[string]userElo)
	builder := aoe4api.NewRequestBuilder()
	for _, u := range us.Users {
		wg.Add(1)
		go func(u user) {
			req, err := builder.
				SetSearchPlayer(u.SteamUsername).
				Request()
			if err != nil {
				fmt.Printf("error building api request: %v", err)
			}

			memberElo, err := req.QueryAllElo()
			if err != nil {
				fmt.Printf("error querying member Elo: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			updatedElo[u.DiscordUserID] = memberElo

			wg.Done()
		}(u)
	}
	wg.Wait()

	for i, u := range us.Users {
		us.Users[i].newElo = updatedElo[u.DiscordUserID]
	}

	if err := us.updateAllEloRoles(s, guildId); err != nil {
		return "", fmt.Errorf("error updating elo roles: %w", err)
	}

	updateMessage, err := us.generateUpdateMessage(s.State, guildId)
	if err != nil {
		return "", fmt.Errorf("error formatting update message: %w", err)
	}

	fmt.Println(updateMessage)

	return updateMessage, nil
}

func getEloTypes() [5]string {
	return [...]string{
		"1v1",
		"2v2",
		"3v3",
		"4v4",
		"custom",
	}
}
