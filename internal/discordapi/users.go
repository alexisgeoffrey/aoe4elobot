package discordapi

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/alexisgeoffrey/aoe4elobot/pkg/aoeapi"
)

type users struct {
	Users []user `json:"users"`
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

func (us users) generateUpdateMessage(st *discordgo.State, guildId string) (string, error) {
	var updateMessage strings.Builder
	updateMessage.WriteString("Elo updated!\n\n")

	for _, u := range us.Users {
		if u.newElo == nil || fmt.Sprint(u.newElo) == fmt.Sprint(u.oldElo) {
			continue
		}

		member, err := st.Member(guildId, u.DiscordUserID)
		if err != nil {
			return "", fmt.Errorf("error retrieving member %s name: %w", u.DiscordUserID, err)
		}

		updateMessage.WriteString(fmt.Sprint(member.Mention(), ":\n"))

		for _, eloT := range aoeapi.GetEloTypes() {
			if oldElo, newElo := u.oldElo[eloT], u.newElo[eloT]; oldElo != newElo {
				updateMessage.WriteString(fmt.Sprintln(eloT, "Elo:", oldElo, "->", newElo))
			}
		}

		updateMessage.WriteByte('\n')
	}

	return strings.TrimSpace(updateMessage.String()), nil
}
