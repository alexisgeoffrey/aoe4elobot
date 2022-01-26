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

	var us users
	if err := json.Unmarshal(configBytes, &us); err != nil {
		return "", fmt.Errorf("error unmarshaling config bytes: %w", err)
	}

	for _, u := range us.Users {
		memberElo, err := getMemberElo(s.State, u, guildId)
		if err != nil {
			return "", fmt.Errorf("error retrieving existing member roles: %w", err)
		}
		u.oldElo = memberElo
	}

	// if err := removeAllEloRoles(s, guildId); err != nil {
	// 	return "", fmt.Errorf("error removing existing roles: %w", err)
	// }

	var wg sync.WaitGroup
	var mu sync.Mutex
	updatedElo := make([]aoeapi.UserElo, 0, len(us.Users))
	for _, u := range us.Users {
		wg.Add(1)
		go func(steamU string) {
			memberElo, err := aoeapi.QueryAll(steamU)
			if err != nil {
				fmt.Printf("error updating member Elo: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			updatedElo = append(updatedElo, memberElo)

			wg.Done()
		}(u.SteamUsername)
	}
	wg.Wait()

	for i, elo := range updatedElo {
		us.Users[i].newElo = elo
	}

	if err := updateAllEloRoles(s, guildId, us); err != nil {
		return "", fmt.Errorf("error formatting update message: %w", err)
	}

	updateMessage, err := formatUpdateMessage(s.State, us.Users, guildId)
	if err != nil {
		return "", fmt.Errorf("error formatting update message: %w", err)
	}

	fmt.Println(updateMessage)

	return updateMessage, nil
}

func getMemberElo(st *discordgo.State, u user, guildId string) (aoeapi.UserElo, error) {
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

		for _, eloT := range aoeapi.GetEloTypes() {
			if strings.Contains(roleName, eloT+" Elo:") {
				memberElo.Elo[eloT] = strings.Split(roleName, " ")[2]
			}
		}
	}

	return memberElo, nil
}

func updateAllEloRoles(s *discordgo.Session, guildId string, us users) error {
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

		ue := u.newElo
		for _, eloType := range aoeapi.GetEloTypes() {
			if elo, ok := ue.Elo[eloType]; ok {
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
