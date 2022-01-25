package discordapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/alexisgeoffrey/aoe4elobot/pkg/aoeapi"
)

type (
	users struct {
		Users []user `json:"users"`
	}

	user struct {
		DiscordUserID string `json:"discord_user_id"`
		SteamUsername string `json:"steam_username"`
		oldElo        aoeapi.UserElo
		newElo        aoeapi.UserElo
	}
)

var cmdMutex sync.Mutex

const configPath string = "config/config.json"

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!setEloName") {
		cmdMutex.Lock()
		defer cmdMutex.Unlock()

		name, err := saveToConfig(m)
		if err != nil {
			s.ChannelMessageSendReply(m.ChannelID,
				"Your Steam username failed to update.\nUsage: `!setEloName Steam_username`", m.Reference())
			fmt.Printf("error updating username: %v\n", err)
			return
		}
		// Send response as a reply to message
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Steam username for %s has been updated to %s.", m.Author.Mention(), name), m.Reference())
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

func UpdateAllElo(s *discordgo.Session, guildId string) (string, error) {
	fmt.Println("Updating Elo...")

	configBytes, err := configFileToBytes()
	if err != nil {
		return "", fmt.Errorf("error converting config file to bytes: %w", err)
	}

	var u users
	if err := json.Unmarshal(configBytes, &u); err != nil {
		return "", fmt.Errorf("error unmarshaling config bytes: %w", err)
	}

	for _, u := range u.Users {
		memberElo, err := getMemberElo(s.State, u, guildId)
		if err != nil {
			return "", fmt.Errorf("error retrieving existing member roles: %w", err)
		}
		u.oldElo = memberElo
	}

	if err := removeAllEloRoles(s, guildId); err != nil {
		return "", fmt.Errorf("error removing existing roles: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	updatedElo := make([]map[string]string, 0, len(u.Users))
	for _, u := range u.Users {
		wg.Add(1)
		go func(u user) {
			memberElo, err := aoeapi.QueryAll(u.SteamUsername)
			if err != nil {
				fmt.Printf("error updating member Elo: %v", err)
			}
			mu.Lock()
			updatedElo = append(updatedElo, memberElo)
			mu.Unlock()

			wg.Done()
		}(u)
	}
	wg.Done()

	if err := addAllEloRoles(s, guildId, u, updatedElo); err != nil {
		return "", fmt.Errorf("error formatting update message: %w", err)
	}

	for i, eloMap := range updatedElo {
		// convert elo map to userElo struct
		userEloJson, err := json.Marshal(eloMap)
		if err != nil {
			return "", fmt.Errorf("error marshaling json userElo: %w", err)
		}
		var ue aoeapi.UserElo
		if err := json.Unmarshal(userEloJson, &ue); err != nil {
			return "", fmt.Errorf("error unmarshaling json userElo: %w", err)
		}
		u.Users[i].newElo = ue
	}

	updateMessage, err := formatUpdateMessage(s.State, u.Users, guildId)
	if err != nil {
		return "", fmt.Errorf("error formatting update message: %w", err)
	}

	fmt.Println(updateMessage)

	return updateMessage, nil
}

func getMemberElo(st *discordgo.State, u user, guildId string) (aoeapi.UserElo, error) {
	st.RLock()
	defer st.RUnlock()

	member, err := st.Member(guildId, u.DiscordUserID)
	if err != nil {
		return aoeapi.UserElo{}, fmt.Errorf("error retrieving member: %w", err)
	}

	var memberElo aoeapi.UserElo
	for _, roleID := range member.Roles {
		role, err := st.Role(guildId, roleID)
		if err != nil {
			fmt.Printf("error retrieving role %s for member %s: %v\n ", roleID, member.User.Username, err)
			continue
		}

		roleName := role.Name
		if err != nil {
			return aoeapi.UserElo{}, fmt.Errorf("error getting role info: %w", err)
		}

		switch {
		case strings.Contains(roleName, "1v1 Elo:"):
			memberElo.Elo1v1 = strings.Split(roleName, " ")[2]
		case strings.Contains(roleName, "2v2 Elo:"):
			memberElo.Elo2v2 = strings.Split(roleName, " ")[2]
		case strings.Contains(roleName, "3v3 Elo:"):
			memberElo.Elo3v3 = strings.Split(roleName, " ")[2]
		case strings.Contains(roleName, "4v4 Elo:"):
			memberElo.Elo4v4 = strings.Split(roleName, " ")[2]
		case strings.Contains(roleName, "Custom Elo:"):
			memberElo.EloCustom = strings.Split(roleName, " ")[2]
		}
	}

	return memberElo, nil
}

func addAllEloRoles(s *discordgo.Session, guildId string, us users, ue []map[string]string) error {
	for i, u := range us.Users {
		for _, eloType := range aoeapi.GetEloTypes() {
			if elo, ok := ue[i][eloType]; ok {
				role, err := s.GuildRoleCreate(guildId)
				if err != nil {
					return fmt.Errorf("error creating guild role: %w", err)
				}
				role, err = s.GuildRoleEdit(guildId, role.ID, fmt.Sprintf("%s Elo: %s", eloType, elo), 1, false, 0, false)
				if err != nil {
					return fmt.Errorf("error editing guild role: %w", err)
				}
				if err := s.GuildMemberRoleAdd(guildId, u.DiscordUserID, role.ID); err != nil {
					return fmt.Errorf("error adding guild role: %w", err)
				}
			}
		}
	}
	return nil
}

func removeAllEloRoles(s *discordgo.Session, guildId string) error {
	s.State.RLock()
	defer s.State.RUnlock()

	guild, err := s.State.Guild(guildId)
	if err != nil {
		return fmt.Errorf("error getting guild from state: %w", err)
	}

	for _, role := range guild.Roles {
		if strings.Contains(role.Name, "Elo:") {
			if err := s.GuildRoleDelete(guildId, role.ID); err != nil {
				fmt.Printf("error removing role %s: %v\n", role.ID, err)
			}
		}
	}

	return nil
}
