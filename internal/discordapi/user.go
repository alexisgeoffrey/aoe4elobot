package discordapi

import (
	"fmt"
	"log"
	"sync"

	"github.com/alexisgeoffrey/aoe4api"
	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
	"github.com/alexisgeoffrey/aoe4elobot/internal/db"
	"github.com/bwmarrin/discordgo"
)

type user db.User

// UpdateGuildElo retrieves and updates all Elo roles on the server specified by the guildId parameter.
func UpdateGuildElo(s *discordgo.Session, guildId string) error {
	log.Println("Updating Elo...")

	users, err := db.GetUsers(guildId)
	if err != nil {
		return fmt.Errorf("error getting users: %v", err)
	}

	var wg sync.WaitGroup
	for _, u := range users {
		wg.Add(1)
		go func(u *db.User) {
			defer wg.Done()
			(*user)(u).updateMemberElo(s, guildId)
		}(u)
	}
	wg.Wait()

	if err := updateGuildEloRoles(users, s, guildId); err != nil {
		return fmt.Errorf("error updating elo roles: %w", err)
	}

	return nil
}

func (u *user) updateMemberElo(s *discordgo.Session, guildId string) error {
	eloAndTs := []struct {
		elo      *int32
		teamSize aoe4api.TeamSize
	}{
		{&u.NewElo.OneVOne, aoe4api.OneVOne},
		{&u.NewElo.TwoVTwo, aoe4api.TwoVTwo},
		{&u.NewElo.ThreeVThree, aoe4api.ThreeVThree},
		{&u.NewElo.FourVFour, aoe4api.FourVFour},
		{&u.NewElo.Custom, 5},
	}

	builder := aoe4api.NewRequestBuilder().
		SetUserAgent(config.UserAgent).
		SetSearchPlayer(u.Aoe4Username)

	for i, t := range config.GetEloTypes() {
		if t.Enabled {
			var req aoe4api.Request
			var err error
			if eloAndTs[i].teamSize == 5 {
				req, err = builder.SetMatchType(aoe4api.Custom).
					Request()
			} else {
				req, err = builder.SetTeamSize(eloAndTs[i].teamSize).
					Request()
			}
			if err != nil {
				return fmt.Errorf("error building request: %v", err)
			}

			memberElo, err := req.QueryElo(u.Aoe4Id)
			if err != nil {
				// fmt.Printf("error querying member Elo: %v\n", err)
				continue
			}

			*eloAndTs[i].elo = int32(memberElo)
		}
	}

	if err := db.UpdateUser(u.DiscordUserID, guildId, u.NewElo); err != nil {
		return fmt.Errorf("error updating user in db: %v", err)
	}

	return nil
}

func updateGuildEloRoles(us []*db.User, s *discordgo.Session, guildId string) error {
	for _, u := range us {
		if err := (*user)(u).changeMemberEloRoles(s, guildId); err != nil {
			return fmt.Errorf("error getting member elo: %v", err)
		}
	}

	return nil
}

func (u *user) changeMemberEloRoles(s *discordgo.Session, guildId string) error {
	eloTypes := config.GetEloTypes()
	oldNewElo := []struct {
		oldElo int32
		newElo int32
	}{
		{u.CurrentElo.OneVOne, u.NewElo.OneVOne},
		{u.CurrentElo.TwoVTwo, u.NewElo.TwoVTwo},
		{u.CurrentElo.ThreeVThree, u.NewElo.ThreeVThree},
		{u.CurrentElo.FourVFour, u.NewElo.FourVFour},
		{u.CurrentElo.Custom, u.NewElo.Custom},
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
				member, err := s.State.Member(guildId, u.DiscordUserID)
				if err != nil {
					return fmt.Errorf("error getting member from state: %v", err)
				}

				var currentRoleId string
				var currentRolePriority int32 = 9999
				for _, currentRole := range member.Roles {
					if rolePriority, ok := eloType.RoleMap[currentRole]; ok {
						currentRoleId = currentRole
						currentRolePriority = rolePriority
						break
					}
				}
				if currentRoleId != role.RoleId {
					changeMemberEloRole(s, member, currentRoleId, role.RoleId)
					roleObj, err := s.State.Role(guildId, role.RoleId)
					if err != nil {
						return fmt.Errorf("error getting role from state: %v", err)
					}

					if currentRolePriority > role.RolePriority {
						s.ChannelMessageSend(
							config.Cfg.BotChannelId,
							fmt.Sprintf("Congrats %s, you are now in %s!",
								member.Mention(),
								roleObj.Name),
						)
					}
				}
				break eloTypeLoop
			}
		}
	}

	return nil
}

func changeMemberEloRole(s *discordgo.Session, m *discordgo.Member, currentRoleId string, newRoleId string) error {
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
