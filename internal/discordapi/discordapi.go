package discordapi

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const (
	UserAgent   = "AOE 4 Elo Bot/2.0.0 (github.com/alexisgeoffrey/aoe4elobot; alexisgeoffrey1@gmail.com)"
	usageString = "Usage: `!setEloInfo SteamUsername/XboxLiveUsername, STEAMID64/XboxLiveID`\nFind STEAMID64 @ https://steamid.io/lookup"
)

var cmdMutex sync.Mutex

// MessageCreate is the handler for Discordgo MessageCreate events.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch lowerMessage := strings.ToLower(m.Content); {
	case // !setEloInfo
		strings.HasPrefix(lowerMessage, "!seteloinfo "),
		strings.HasPrefix(lowerMessage, "!set "),
		strings.HasPrefix(lowerMessage, "!link "):

		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		setEloInfoError := func(err error) {
			s.ChannelMessageSendReply(
				m.ChannelID,
				fmt.Sprint("Your AOE4 info failed to update.\n", usageString),
				m.Reference())
			log.Printf("error updating info: %v\n", err)
		}

		input := strings.SplitN(m.Content, " ", 2)
		if len(input) <= 1 {
			setEloInfoError(fmt.Errorf("invalid input for info: %s", m.Content))
			return
		}
		infoInput := strings.Split(input[1], ",")
		if len(infoInput) <= 1 {
			setEloInfoError(fmt.Errorf("invalid input for info: %s", m.Content))
			return
		}
		aoe4Username, aoe4Id := strings.TrimSpace(infoInput[0]), strings.TrimSpace(infoInput[1])

		err := registerUser(aoe4Username, aoe4Id, m.Author.ID, m.GuildID)
		if err != nil {
			setEloInfoError(err)
			return
		}
		// Send response as a reply to message
		s.ChannelMessageSendReply(
			m.ChannelID,
			fmt.Sprintf("%s's AOE4 username has been updated to %s and ID has been updated to %s.",
				m.Author.Mention(),
				aoe4Username,
				aoe4Id),
			m.Reference())

	// case // !updateElo
	// 	lowerMessage == "!updateelo",
	// 	lowerMessage == "!update",
	// 	lowerMessage == "!u":
	// 	cmdMutex.Lock()
	// 	defer cmdMutex.Unlock()

	// 	s.ChannelMessageSend(m.ChannelID, "Updating elo...")
	// 	if err := UpdateAllElo(s, m.GuildID); err != nil {
	// 		s.ChannelMessageSend(m.ChannelID, "Elo failed to update.")
	// 		log.Printf("error updating elo: %v\n", err)
	// 		return
	// 	}

	case // !eloInfo
		strings.HasPrefix(lowerMessage, "!eloinfo"),
		strings.HasPrefix(lowerMessage, "!info"):

	case // !help
		lowerMessage == "!help",
		lowerMessage == "!h":

		s.ChannelMessageSend(m.ChannelID, usageString)
	}
}

// UpdateAllElo retrieves and updates all Elo roles on the server specified by the guildId
// parameter. It returns an update message containing all changed Elo values for each server member.
// func UpdateAllElo(s *discordgo.Session, guildId string) error {
// 	log.Println("Updating Elo...")

// 	users, err := getUsers(guildId)
// 	if err != nil {
// 		return fmt.Errorf("error getting users: %v", err)
// 	}

// 	var wg sync.WaitGroup
// 	var mu sync.Mutex

// 	builder := aoe4api.NewRequestBuilder().
// 		SetUserAgent(UserAgent)

// 	getElo := func(u user, eloField *int32, teamSize aoe4api.TeamSize) error {
// 		req, err := builder.SetSearchPlayer(u.aoe4Username).
// 			SetTeamSize(teamSize).
// 			Request()
// 		if err != nil {
// 			return fmt.Errorf("error building request: %v", err)
// 		}
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()

// 			memberElo, err := req.QueryElo(u.aoe4Id)
// 			if err != nil {
// 				fmt.Printf("error querying member Elo: %v\n", err)
// 			}

// 			mu.Lock()
// 			defer mu.Unlock()
// 			*eloField = memberElo
// 		}()

// 		return nil
// 	}

// 	for i, u := range users {
// 		if Config.OneVOne.Enabled {
// 			getElo(u, &users[i].newElo.oneVOne, aoe4api.OneVOne)
// 		}
// 		if Config.TwoVTwo.Enabled {
// 			getElo(u, &users[i].newElo.twoVTwo, aoe4api.TwoVTwo)
// 		}
// 		if Config.ThreeVThree.Enabled {
// 			getElo(u, &users[i].newElo.threeVThree, aoe4api.ThreeVThree)
// 		}
// 		if Config.FourVFour.Enabled {
// 			getElo(u, &users[i].newElo.fourVFour, aoe4api.FourVFour)
// 		}
// 		if Config.Custom.Enabled {
// 			getElo(u, &users[i].newElo.custom, 5)
// 		}
// 	}
// 	wg.Wait()

// 	if err := updateAllEloRoles(users, s, guildId); err != nil {
// 		return fmt.Errorf("error updating elo roles: %w", err)
// 	}

// 	return nil
// }
