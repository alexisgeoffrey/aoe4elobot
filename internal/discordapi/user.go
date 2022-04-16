package discordapi

import (
	"fmt"
	"log"
	"sync"

	"github.com/alexisgeoffrey/aoe4api"
	"github.com/alexisgeoffrey/aoe4elobot/v2/internal/config"
	"github.com/alexisgeoffrey/aoe4elobot/v2/internal/db"
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
	for i := range users {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			user := (*user)(&users[i])
			user.updateMemberElo(s, guildId)
		}(i)
	}
	wg.Wait()

	if err := updateGuildEloRoles(users, s, guildId); err != nil {
		return fmt.Errorf("error updating elo roles: %w", err)
	}

	return nil
}

func (u *user) updateMemberElo(s *discordgo.Session, guildId string) error {
	eloAndTs := []struct {
		currentElo *int32
		newElo     *int32
		teamSize   aoe4api.TeamSize
	}{
		{&u.CurrentElo.OneVOne, &u.NewElo.OneVOne, aoe4api.OneVOne},
		{&u.CurrentElo.TwoVTwo, &u.NewElo.TwoVTwo, aoe4api.TwoVTwo},
		{&u.CurrentElo.ThreeVThree, &u.NewElo.ThreeVThree, aoe4api.ThreeVThree},
		{&u.CurrentElo.FourVFour, &u.NewElo.FourVFour, aoe4api.FourVFour},
		{&u.CurrentElo.Custom, &u.NewElo.Custom, 5},
	}

	builder := aoe4api.NewRequestBuilder().
		SetUserAgent(config.UserAgent).
		SetSearchPlayer(u.Aoe4Username)

	var wg sync.WaitGroup
	for i, t := range config.Cfg.EloTypes {
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

			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				memberElo, err := req.QueryElo(u.Aoe4Id)
				if err != nil {
					*eloAndTs[i].newElo = *eloAndTs[i].currentElo
					// log.Printf("no response from api for %s for elo %s", u.Aoe4Username, [...]string{"1v1", "2v2", "3v3", "4v4", "Custom"}[i])
					return
				}

				*eloAndTs[i].newElo = int32(memberElo)
			}(i)
		}
	}
	wg.Wait()

	// user := (*user)(u)
	// if err := user.updateMemberEloRoles(s, guildId); err != nil {
	// 	return fmt.Errorf("error getting member elo: %v", err)
	// }

	if err := db.UpdateUserElo(u.DiscordUserID, guildId, u.NewElo); err != nil {
		return fmt.Errorf("error updating user in db: %v", err)
	}

	return nil
}

func updateGuildEloRoles(us []db.User, s *discordgo.Session, guildId string) error {
	for _, u := range us {
		user := user(u)
		if err := user.updateMemberEloRoles(s, guildId); err != nil {
			return fmt.Errorf("error getting member elo: %v", err)
		}
	}

	return nil
}

func (u *user) updateMemberEloRoles(s *discordgo.Session, guildId string) error {
	eloTypes := config.Cfg.EloTypes
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

	member, err := s.State.Member(guildId, u.DiscordUserID)
	if err != nil {
		return fmt.Errorf("error getting member from state: %v", err)
	}
eloTypeLoop:
	for _, eloType := range eloTypes {
		var currentRoleId string
		var currentRolePriority int32 = 9999
		for _, currentRole := range member.Roles {
			if rolePriority, ok := eloType.RoleMap[currentRole]; ok {
				currentRoleId = currentRole
				currentRolePriority = rolePriority
				break
			}
		}

		for _, role := range eloType.Roles {
			if highestElo >= role.StartingElo && highestElo <= role.EndingElo {
				if currentRoleId == role.RoleId {
					break eloTypeLoop
				}
				if err := changeMemberEloRole(s, member, currentRoleId, role.RoleId); err != nil {
					return fmt.Errorf("error changing member elo role from %s to %s: %v", currentRoleId, role.RoleId, err)
				}
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
				break eloTypeLoop
			}
		}

		if err := changeMemberEloRole(s, member, currentRoleId, ""); err != nil {
			return fmt.Errorf("error removing member elo role from %s: %v", currentRoleId, err)
		}
		break // TODO
	}

	return nil
}

func changeMemberEloRole(s *discordgo.Session, m *discordgo.Member, currentRoleId string, newRoleId string) error {
	if currentRoleId != "" {
		if err := s.GuildMemberRoleRemove(m.GuildID, m.User.ID, currentRoleId); err != nil {
			return fmt.Errorf("error removing role: %v", err)
		}
		log.Printf("role %s removed from user %s", currentRoleId, m.Mention())
	}

	if newRoleId != "" {
		if err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, newRoleId); err != nil {
			return fmt.Errorf("error adding role: %v", err)
		}
		log.Printf("role %s added to user %s", newRoleId, m.Mention())
	}

	return nil
}
