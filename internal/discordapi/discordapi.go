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
	userElo map[string]string

	users struct {
		Users []user `json:"users"`
	}

	user struct {
		DiscordUserID string `json:"discord_user_id"`
		SteamUsername string `json:"steam_username"`
		oldElo        userElo
		newElo        userElo
	}
)

var cmdMutex sync.Mutex

const configPath = "config/config.json"

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
	for _, u := range us.Users {
		wg.Add(1)
		go func(u user) {
			memberElo, err := aoeapi.QueryAll(u.SteamUsername)
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

func (u user) getMemberElo(st *discordgo.State, guildId string) (userElo, error) {
	member, err := st.Member(guildId, u.DiscordUserID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving member: %w", err)
	}

	memberElo := make(map[string]string)
	for _, roleID := range member.Roles {
		role, err := st.Role(guildId, roleID)
		if err != nil {
			fmt.Printf("error retrieving role %s for member %s: %v\n ", roleID, member.User.Username, err)
			continue
		}

		roleName := role.Name
		if err != nil {
			return nil, fmt.Errorf("error getting role info: %w", err)
		}

		for _, eloT := range aoeapi.GetEloTypes() {
			if strings.Contains(roleName, eloT+" Elo:") {
				memberElo[eloT] = strings.Split(roleName, " ")[2]
			}
		}
	}

	return memberElo, nil
}

func (us users) updateAllEloRoles(s *discordgo.Session, guildId string) error {
	for _, u := range us.Users {
		member, err := s.State.Member(guildId, u.DiscordUserID)
		if err != nil {
			return fmt.Errorf("error retrieving guild member %s: %w", u.DiscordUserID, err)
		}
		roleSet := make(map[string]*discordgo.Role)
		for _, roleId := range member.Roles {
			role, err := s.State.Role(guildId, roleId)
			if err != nil {
				return fmt.Errorf("error retrieving guild role %s: %w", roleId, err)
			}
			if strings.Contains(role.Name, "Elo:") {
				roleSet[strings.Split(role.Name, " ")[0]] = role
			}
		}

		for _, eloType := range aoeapi.GetEloTypes() {
			if elo, ok := u.newElo[eloType]; ok && elo != u.oldElo[eloType] {
				roleName := fmt.Sprintf("%s Elo: %s", eloType, elo)
				if role, ok := roleSet[eloType]; ok {
					_, err = s.GuildRoleEdit(guildId, role.ID, roleName, 1, false, 0, false)
					if err != nil {
						return fmt.Errorf("error editing guild role: %w", err)
					}
				} else {
					role, err := s.GuildRoleCreate(guildId)
					if err != nil {
						return fmt.Errorf("error creating guild role: %w", err)
					}
					role, err = s.GuildRoleEdit(guildId, role.ID, roleName, 1, false, 0, false)
					if err != nil {
						return fmt.Errorf("error editing guild role: %w", err)
					}
					if err := s.GuildMemberRoleAdd(guildId, u.DiscordUserID, role.ID); err != nil {
						return fmt.Errorf("error adding guild role: %w", err)
					}
				}
			}
		}
	}
	return nil
}

// func removeAllEloRoles(s *discordgo.Session, guildId string) error {
// 	guild, err := s.State.Guild(guildId)
// 	if err != nil {
// 		return fmt.Errorf("error getting guild from state: %w", err)
// 	}

// 	s.State.RLock()
// 	defer s.State.RUnlock()
// 	for _, role := range guild.Roles {
// 		if strings.Contains(role.Name, "Elo:") {
// 			if err := s.GuildRoleDelete(guildId, role.ID); err != nil {
// 				fmt.Printf("error removing role %s: %v\n", role.ID, err)
// 			}
// 		}
// 	}

// 	return nil
// }
