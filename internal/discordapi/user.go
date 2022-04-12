package discordapi

import (
	"fmt"
	"log"
	"sync"

	"github.com/alexisgeoffrey/aoe4api"
	"github.com/bwmarrin/discordgo"
)

type user struct {
	discordUserID string
	aoe4Username  string
	aoe4Id        string
	oldElo        userElo
	newElo        userElo
}

func updateAllEloRoles(us []user, s *discordgo.Session, guildId string) error {
	for _, u := range us {
		if err := u.updateMemberEloRoles(s, guildId); err != nil {
			return fmt.Errorf("error getting member elo: %v", err)
		}
	}

	return nil
}

func (u user) updateMemberEloRoles(s *discordgo.Session, guildId string) error {
	oldNewElo := []struct {
		oldElo int32
		newElo int32
	}{
		{u.oldElo.oneVOne, u.newElo.oneVOne},
		{u.oldElo.twoVTwo, u.newElo.twoVTwo},
		{u.oldElo.threeVThree, u.newElo.threeVThree},
		{u.oldElo.fourVFour, u.newElo.fourVFour},
		{u.oldElo.custom, u.newElo.custom},
	}
	for i, elo := range oldNewElo {
		if EloTypes[i].Enabled && elo.oldElo != elo.newElo {
			for _, role := range EloTypes[i].Roles {
				if elo.newElo >= role.StartingElo && elo.newElo <= role.EndingElo {
					member, err := s.State.Member(guildId, u.discordUserID)
					if err != nil {
						return fmt.Errorf("error getting member from state: %v", err)
					}

					var currentRoleId string
					for _, currentRole := range member.Roles {
						if EloTypes[i].RoleMap[currentRole] {
							currentRoleId = currentRole
							break
						}
					}
					if currentRoleId != role.RoleId {
						updateEloRole(s, member, currentRoleId, role.RoleId)
						roleObj, err := s.State.Role(guildId, role.RoleId)
						if err != nil {
							return fmt.Errorf("error getting role from state: %v", err)
						}

						s.ChannelMessageSend(
							Config.BotChannelId,
							fmt.Sprintf("Congrats %s, you are now in %s!",
								member.Mention(),
								roleObj.Name),
						)
					}
					break
				}
			}
			log.Println("no valid role for elo ", elo.newElo)
		}

		// if Config.OneVOne.Enabled && u.oldElo.oneVOne != u.newElo.oneVOne {
		// 	for _, role := range Config.OneVOne.Roles {
		// 		if u.newElo.oneVOne >= role.StartingElo && u.newElo.oneVOne <= role.EndingElo {
		// 			member, err := s.State.Member(guildId, u.discordUserID)
		// 			if err != nil {
		// 				return fmt.Errorf("error getting member from state: %v", err)
		// 			}

		// 			var currentRoleId string
		// 			for _, currentRole := range member.Roles {
		// 				if Config.OneVOne.RoleMap[currentRole] {
		// 					currentRoleId = currentRole
		// 					break
		// 				}
		// 			}
		// 			if currentRoleId != role.RoleId {
		// 				updateEloRole(s, member, currentRoleId, role.RoleId)
		// 				roleObj, err := s.State.Role(guildId, role.RoleId)
		// 				if err != nil {
		// 					return fmt.Errorf("error getting role from state: %v", err)
		// 				}

		// 				s.ChannelMessageSend(
		// 					Config.BotChannelId,
		// 					fmt.Sprintf("Congrats %s, you are now in %s!",
		// 						member.Mention(),
		// 						roleObj.Name),
		// 				)
		// 			}
		// 			break
		// 		}
		// 	}
		// 	log.Println("no valid role for elo ", u.newElo.oneVOne)
		// }
	}

	return nil
}

func updateEloRole(s *discordgo.Session, m *discordgo.Member, currentRoleId string, newRoleId string) error {
	if currentRoleId != "" {
		err := s.GuildMemberRoleRemove(m.GuildID, m.User.ID, currentRoleId)
		if err != nil {
			return fmt.Errorf("error removing role: %v", err)
		}
	}

	if err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, newRoleId); err != nil {
		return fmt.Errorf("error adding role: %v", err)
	}

	return nil
}

func (u *user) updateMemberElo(s *discordgo.Session, guildId string) error {
	eloAndSize := []struct {
		elo      *int32
		teamSize aoe4api.TeamSize
	}{
		{&u.newElo.oneVOne, aoe4api.OneVOne},
		{&u.newElo.twoVTwo, aoe4api.TwoVTwo},
		{&u.newElo.threeVThree, aoe4api.ThreeVThree},
		{&u.newElo.fourVFour, aoe4api.FourVFour},
		{&u.newElo.custom, 5},
	}

	builder := aoe4api.NewRequestBuilder().
		SetUserAgent(UserAgent).
		SetSearchPlayer(u.aoe4Username)

	for i, t := range EloTypes {
		if t.Enabled {
			var req aoe4api.Request
			var err error
			if eloAndSize[i].teamSize == 5 {
				req, err = builder.SetMatchType(aoe4api.Custom).
					Request()
			} else {
				req, err = builder.SetTeamSize(eloAndSize[i].teamSize).
					Request()
			}
			if err != nil {
				return fmt.Errorf("error building request: %v", err)
			}

			memberElo, err := req.QueryElo(u.aoe4Id)
			if err != nil {
				// fmt.Printf("error querying member Elo: %v\n", err)
				continue
			}

			*eloAndSize[i].elo = int32(memberElo)
		}
	}

	if err := updateUser(u.discordUserID, guildId, u.newElo); err != nil {
		return fmt.Errorf("error updating user in db: %v", err)
	}

	return nil
}

// UpdateAllElo retrieves and updates all Elo roles on the server specified by the guildId parameter.
func UpdateAllElo(s *discordgo.Session, guildId string) error {
	log.Println("Updating Elo...")

	users, err := getUsers(guildId)
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}

	var wg sync.WaitGroup
	// var mu sync.Mutex

	for _, u := range users {
		wg.Add(1)
		go func(u user) {
			defer wg.Done()
			u.updateMemberElo(s, guildId)
		}(u)
	}
	wg.Wait()

	if err := updateAllEloRoles(users, s, guildId); err != nil {
		return fmt.Errorf("error updating elo roles: %w", err)
	}

	return nil
}
