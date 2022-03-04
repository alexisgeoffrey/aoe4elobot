package discordapi

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type users struct {
	Users []user `json:"users"`
}

// func (us users) updateAllEloRoles(s *discordgo.Session, guildId string) error {
// 	reorder := false
// 	for _, u := range us.Users {
// 		member, err := s.State.Member(guildId, u.DiscordUserID)
// 		if err != nil {
// 			return fmt.Errorf("error retrieving guild member %s: %w", u.DiscordUserID, err)
// 		}
// 		roleSet := map[string]*discordgo.Role{}
// 		for _, roleId := range member.Roles {
// 			role, err := s.State.Role(guildId, roleId)
// 			if err != nil {
// 				return fmt.Errorf("error retrieving guild role %s: %w", roleId, err)
// 			}
// 			if strings.Contains(role.Name, "Elo:") {
// 				roleSet[strings.ToLower(strings.Split(role.Name, " ")[0])] = role
// 			}
// 		}

// 		for _, eloType := range getEloTypes() {
// 			if elo, ok := u.NewElo[eloType]; ok && elo != u.OldElo[eloType] {
// 				roleName := fmt.Sprintf("%s Elo: %s", strings.Title(eloType), elo)
// 				if role, ok := roleSet[eloType]; ok {
// 					_, err = s.GuildRoleEdit(guildId, role.ID, roleName, 1, false, 0, false)
// 					if err != nil {
// 						return fmt.Errorf("error editing guild role: %w", err)
// 					}
// 				} else {
// 					role, err := s.GuildRoleCreate(guildId)
// 					if err != nil {
// 						return fmt.Errorf("error creating guild role: %w", err)
// 					}
// 					role, err = s.GuildRoleEdit(guildId, role.ID, roleName, 1, false, 0, false)
// 					if err != nil {
// 						return fmt.Errorf("error editing guild role: %w", err)
// 					}
// 					if err := s.GuildMemberRoleAdd(guildId, u.DiscordUserID, role.ID); err != nil {
// 						return fmt.Errorf("error adding guild role: %w", err)
// 					}
// 					reorder = true
// 				}
// 			}
// 		}
// 	}

// 	if reorder {
// 		g, err := s.State.Guild(guildId)
// 		if err != nil {
// 			return fmt.Errorf("error retrieving guild %s: %w", guildId, err)
// 		}
// 		s.State.Lock()
// 		defer s.State.Unlock()
// 		rs := make([]*discordgo.Role, 0, len(g.Roles))

// 		for _, role := range g.Roles {
// 			if strings.Contains(role.Name, "Elo:") {
// 				rs = append(rs, role)
// 			}
// 		}

// 		sort.SliceStable(rs, func(i, j int) bool {
// 			matchSize := func(prefix string) aoe4api.TeamSize {
// 				switch prefix {
// 				case "1v1":
// 					return aoe4api.OneVOne
// 				case "2v2":
// 					return aoe4api.TwoVTwo
// 				case "3v3":
// 					return aoe4api.ThreeVThree
// 				case "4v4":
// 					return aoe4api.FourVFour
// 				case "Custom":
// 					return 5
// 				}
// 				return 0
// 			}

// 			iPrefix, jPrefix := strings.Split(rs[i].Name, " ")[0], strings.Split(rs[j].Name, " ")[0]
// 			return matchSize(iPrefix) > matchSize(jPrefix)
// 		})

// 		for i := range rs {
// 			rs[i].Position = i
// 		}

// 		if _, err := s.GuildRoleReorder(guildId, rs); err != nil {
// 			return fmt.Errorf("error reordering guild roles: %w", err)
// 		}
// 	}

// 	return nil
// }

func (us users) generateUpdateMessage(st *discordgo.State, guildId string) (string, error) {
	var updateMessage strings.Builder
	updateMessage.WriteString("Elo updated!\n\n")

	for _, u := range us.Users {
		if u.NewElo == nil || fmt.Sprint(u.NewElo) == fmt.Sprint(u.OldElo) {
			continue
		}

		member, err := st.Member(guildId, u.DiscordUserID)
		if err != nil {
			return "", fmt.Errorf("error retrieving member %s name: %w", u.DiscordUserID, err)
		}

		updateMessage.WriteString(fmt.Sprint(member.User.Username, ":\n"))

		for _, eloT := range getEloTypes() {
			if oldElo, newElo := u.OldElo[eloT], u.NewElo[eloT]; oldElo != newElo {
				updateMessage.WriteString(fmt.Sprintln(strings.Title(eloT), "Elo:", oldElo, "->", newElo))
			}
		}

		updateMessage.WriteByte('\n')
	}

	return strings.TrimSpace(updateMessage.String()), nil
}
