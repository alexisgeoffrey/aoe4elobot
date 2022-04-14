package discordapi

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
	"github.com/alexisgeoffrey/aoe4elobot/internal/db"
	"github.com/bwmarrin/discordgo"
)

const (
	usageString = "Usage:\n```\n!setEloInfo SteamUsername/XboxLiveUsername, STEAMID64/XboxLiveID\nAliases: !set, !link\n\n!updateElo\nAliases: !update, !u\n\n!eloInfo [@User]\nAliases: !info, !stats, !i, !s\n```\nFind STEAMID64 @ https://steamid.io/lookup"
)

var cmdMutex sync.Mutex

// MessageCreate is the handler for Discordgo MessageCreate events.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	dedupedMessage := strings.Join(strings.Fields(m.Content), " ")
	switch lowerTrimmedMessage := strings.TrimSpace(strings.ToLower(dedupedMessage)); {
	case // !setEloInfo
		strings.HasPrefix(lowerTrimmedMessage, "!seteloinfo "),
		strings.HasPrefix(lowerTrimmedMessage, "!set "),
		strings.HasPrefix(lowerTrimmedMessage, "!link "):

		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		setEloInfoError := func() {
			s.ChannelMessageSendReply(
				m.ChannelID,
				fmt.Sprint("Your AOE4 info failed to update.\n", usageString),
				m.Reference())
			log.Printf("error updating info: %v\n", fmt.Errorf("invalid input for info: %s", m.Content))
		}

		input := strings.SplitN(dedupedMessage, " ", 2)
		if len(input) <= 1 {
			setEloInfoError()
			return
		}

		var mention string
		var infoInput []string
		if strings.HasPrefix(input[1], "<@") {
			targetMember, err := s.State.Member(m.GuildID, m.Author.ID)
			if err != nil {
				s.ChannelMessageSendReply(
					m.ChannelID,
					fmt.Sprint("Unable to retrieve Elo info.\n", usageString),
					m.Reference())
				log.Printf("error getting member from state: %v", err)
				return
			}

			var isAdmin bool
			for _, roleId := range targetMember.Roles {
				if config.Cfg.AdminRolesMap[roleId] {
					isAdmin = true
					break
				}
			}

			if !isAdmin {
				s.ChannelMessageSendReply(
					m.ChannelID,
					fmt.Sprint("Insufficient privileges to set Elo info for another user.\n", usageString),
					m.Reference())
				return
			}
			fullInfoInput := strings.SplitN(input[1], " ", 2)
			if len(fullInfoInput) <= 1 {
				setEloInfoError()
				return
			}
			mention = fullInfoInput[0]
			infoInput = strings.Split(fullInfoInput[1], ",")
		} else {
			infoInput = strings.Split(input[1], ",")
		}
		if len(infoInput) <= 1 {
			setEloInfoError()
			return
		}

		aoe4Username, aoe4Id := strings.TrimSpace(infoInput[0]), strings.TrimSpace(infoInput[1])

		sendUpdateMessage := func(mention string) {
			s.ChannelMessageSendReply(
				m.ChannelID,
				fmt.Sprintf("%s's AOE4 username has been updated to %s and ID has been updated to %s.",
					mention,
					aoe4Username,
					aoe4Id),
				m.Reference())
		}

		if mention == "" {
			if err := db.RegisterUser(aoe4Username, aoe4Id, m.Author.ID, m.GuildID); err != nil {
				setEloInfoError()
				return
			}

			sendUpdateMessage(m.Author.Mention())
		} else {
			if err := db.RegisterUser(aoe4Username, aoe4Id, strings.Trim(mention, "<@>"), m.GuildID); err != nil {
				setEloInfoError()
				return
			}

			sendUpdateMessage(mention)
		}

	case // !updateElo
		lowerTrimmedMessage == "!updateelo",
		lowerTrimmedMessage == "!update",
		lowerTrimmedMessage == "!u":

		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		s.ChannelMessageSend(m.ChannelID, "Updating elo...")
		if err := UpdateGuildElo(s, m.GuildID); err != nil {
			s.ChannelMessageSend(m.ChannelID, "Elo failed to update.")
			log.Printf("error updating elo: %v\n", err)
			return
		}

		s.ChannelMessageSend(m.ChannelID, "Elo updated!")

	case // !eloInfo
		strings.HasPrefix(lowerTrimmedMessage, "!eloinfo"),
		strings.HasPrefix(lowerTrimmedMessage, "!info"),
		strings.HasPrefix(lowerTrimmedMessage, "!i"),
		strings.HasPrefix(lowerTrimmedMessage, "!stats"),
		strings.HasPrefix(lowerTrimmedMessage, "!s"):

		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		eloInfoError := func() {
			s.ChannelMessageSendReply(
				m.ChannelID,
				fmt.Sprint("Unable to retrieve Elo info.\n", usageString),
				m.Reference())
		}

		input := strings.SplitN(dedupedMessage, " ", 2)
		if len(input) == 1 {
			u, err := db.GetUser(m.Author.ID, m.GuildID)
			if err != nil {
				s.ChannelMessageSendReply(
					m.ChannelID,
					fmt.Sprint("You are not registered.\n", usageString),
					m.Reference())
				log.Printf("error getting info: %v\n", err)
				return
			}

			if err := (*user)(u).updateMemberElo(s, m.GuildID); err != nil {
				eloInfoError()
				log.Printf("error updating member elo: %v\n", err)
				return
			}

			s.ChannelMessageSendReply(
				m.ChannelID,
				u.NewElo.GenerateEloString(m.Author.Username),
				m.Reference())
		} else if len(input) == 2 {
			if !strings.HasPrefix(input[1], "<@") {
				eloInfoError()
				log.Printf("error updating member elo: %v\n", fmt.Errorf("invalid input"))
				return
			}
			u, err := db.GetUser(strings.Trim(input[1], "<@>"), m.GuildID)
			if err != nil {
				s.ChannelMessageSendReply(
					m.ChannelID,
					fmt.Sprint("User is not registered.\n", usageString),
					m.Reference())
				log.Printf("error getting info: %v\n", err)
				return
			}

			if err := (*user)(u).updateMemberElo(s, m.GuildID); err != nil {
				eloInfoError()
				log.Printf("error updating member elo: %v\n", err)
				return
			}

			targetMember, err := s.State.Member(m.GuildID, u.DiscordUserID)
			if err != nil {
				eloInfoError()
				log.Printf("error getting member from state: %v", err)
			}

			s.ChannelMessageSendReply(
				m.ChannelID,
				u.NewElo.GenerateEloString(targetMember.User.Username),
				m.Reference())
		} else {
			s.ChannelMessageSendReply(
				m.ChannelID,
				fmt.Sprint("Unable to retrieve Elo info.\n", usageString),
				m.Reference())
			log.Printf("error getting info: %v\n", fmt.Errorf("invalid input for info: %s", m.Content))
			return
		}

	case // !help
		lowerTrimmedMessage == "!help",
		lowerTrimmedMessage == "!h":

		s.ChannelMessageSend(m.ChannelID, usageString)
	}
}
