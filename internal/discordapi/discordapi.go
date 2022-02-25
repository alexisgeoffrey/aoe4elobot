package discordapi

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/alexisgeoffrey/aoe4api"
	"github.com/bwmarrin/discordgo"
)

type userElo map[string]string

const UserAgent = "AOE 4 Elo Bot/0.0.0 (github.com/alexisgeoffrey/aoe4elobot; alexisgeoffrey1@gmail.com)"

const usageString = "Usage: `!setEloInfo aoe4_username, aoe4_id`\nFind STEAMID64 @ https://steamid.io/lookup"

var cmdMutex sync.Mutex

// MessageCreate is the handler for Discordgo MessageCreate events.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(strings.ToLower(m.Content), "!seteloinfo") {
		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		name, id, err := parseRegistration(m.Content, m.Author.ID)
		if err != nil {
			s.ChannelMessageSendReply(
				m.ChannelID,
				fmt.Sprint("Your AOE4 info failed to update.\n",
					usageString),
				m.Reference())
			log.Printf("error updating info: %v\n", err)
			return
		}
		// Send response as a reply to message
		s.ChannelMessageSendReply(
			m.ChannelID,
			fmt.Sprintf("%s's AOE4 username has been updated to %s and ID has been updated to %s.",
				m.Author.Mention(),
				name,
				id),
			m.Reference())
	} else if strings.HasPrefix(strings.ToLower(m.Content), "!updateelo") {
		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		s.ChannelMessageSend(m.ChannelID, "Updating elo...")
		updateMessage, err := UpdateAllElo(s, m.GuildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Elo failed to update.")
			log.Printf("error updating elo: %v\n", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, updateMessage)
	} else if strings.HasPrefix(strings.ToLower(m.Content), "!elousage") {
		s.ChannelMessageSend(m.ChannelID, usageString)
	}
}

// UpdateAllElo retrieves and updates all Elo roles on the server specified by the guildId
// parameter. It returns an update message containing all changed Elo values for each server member.
func UpdateAllElo(s *discordgo.Session, guildId string) (string, error) {
	log.Println("Updating Elo...")

	configBytes, err := configFileToBytes()
	if err != nil {
		return "", fmt.Errorf("error converting config file to bytes: %w", err)
	}

	var us users
	if err := json.Unmarshal(configBytes, &us); err != nil {
		return "", fmt.Errorf("error unmarshaling config bytes: %w", err)
	}

	// for i, u := range us.Users {
	// 	memberElo, err := u.getMemberElo(s.State, guildId)
	// 	if err != nil {
	// 		return "", fmt.Errorf("error retrieving existing member roles: %w", err)
	// 	}
	// 	us.Users[i].OldElo = memberElo
	// }

	for i, u := range us.Users {
		us.Users[i].OldElo = u.NewElo
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	updatedElo := make(map[string]userElo, len(us.Users))
	builder := aoe4api.NewRequestBuilder().
		SetUserAgent(UserAgent)
	for _, u := range us.Users {
		wg.Add(1)
		req, err := builder.
			SetSearchPlayer(u.Aoe4Username).
			Request()
		go func(u user) {
			if err != nil {
				log.Printf("error building api request: %v", err)
			}

			memberElo, err := req.QueryAllElo(u.Aoe4Id)
			if err != nil {
				log.Printf("error querying member Elo: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			updatedElo[u.DiscordUserID] = memberElo

			wg.Done()
		}(u)
	}
	wg.Wait()

	for i, u := range us.Users {
		us.Users[i].NewElo = updatedElo[u.DiscordUserID]
		saveToConfig(u)
	}

	// if err := us.updateAllEloRoles(s, guildId); err != nil {
	// 	return "", fmt.Errorf("error updating elo roles: %w", err)
	// }

	updateMessage, err := us.generateUpdateMessage(s.State, guildId)
	if err != nil {
		return "", fmt.Errorf("error formatting update message: %w", err)
	}

	log.Println(updateMessage)

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
