package discordapi

import (
	"fmt"
	"log"
	"sync"

	"github.com/alexisgeoffrey/aoe4api"
	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
	"github.com/bwmarrin/discordgo"
)

type user struct {
	discordUserID string
	aoe4Username  string
	aoe4Id        string
	currentElo    userElo
	newElo        userElo
}

func updateAllEloRoles(us []*user, s *discordgo.Session, guildId string) error {
	for _, u := range us {
		if err := u.updateMemberEloRoles(s, guildId); err != nil {
			return fmt.Errorf("error getting member elo: %v", err)
		}
	}

	return nil
}

func (u user) updateMemberEloRoles(s *discordgo.Session, guildId string) error {
	eloTypes := config.GetEloTypes()
	oldNewElo := []struct {
		oldElo int32
		newElo int32
	}{
		{u.currentElo.oneVOne, u.newElo.oneVOne},
		{u.currentElo.twoVTwo, u.newElo.twoVTwo},
		{u.currentElo.threeVThree, u.newElo.threeVThree},
		{u.currentElo.fourVFour, u.newElo.fourVFour},
		{u.currentElo.custom, u.newElo.custom},
	}
	var highestElo int32
	for i, elo := range oldNewElo {
		if eloTypes[i].Enabled && elo.newElo > highestElo {
			highestElo = elo.newElo
		}
	}

eloTypeLoop:
	for _, eloType := range eloTypes {
		for _, role := range eloType.Roles {
			if highestElo >= role.StartingElo && highestElo <= role.EndingElo {
				member, err := s.State.Member(guildId, u.discordUserID)
				if err != nil {
					return fmt.Errorf("error getting member from state: %v", err)
				}

				var currentRoleId string
				currentRolePriority := int32(9999)
				for _, currentRole := range member.Roles {
					if rolePriority, ok := eloType.RoleMap[currentRole]; ok {
						currentRoleId = currentRole
						currentRolePriority = rolePriority
						break
					}
				}
				if currentRoleId != role.RoleId {
					updateEloRole(s, member, currentRoleId, role.RoleId)
					roleObj, err := s.State.Role(guildId, role.RoleId)
					if err != nil {
						return fmt.Errorf("error getting role from state: %v", err)
					}

					if currentRolePriority > role.RolePriority {
						s.ChannelMessageSend(
							config.Config.BotChannelId,
							fmt.Sprintf("Congrats %s, you are now in %s!",
								member.Mention(),
								roleObj.Name),
						)
					}
				}
				break eloTypeLoop
			}
		}
		log.Println("no valid role for elo", highestElo)
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

	for i, t := range config.GetEloTypes() {
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
	for _, u := range users {
		wg.Add(1)
		go func(u *user) {
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
