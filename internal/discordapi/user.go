package discordapi

import (
	"errors"
	"fmt"
	"log"
	"strings"
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
		return fmt.Errorf("error getting users: %w", err)
	}

	var wg sync.WaitGroup
	for i := range users {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			user := (*user)(&users[i])
			if err := user.updateMemberElo(s, guildId); err != nil {
				log.Println(err)
			}
		}(i)
	}
	wg.Wait()

	if err := updateGuildEloRoles(users, s, guildId); err != nil {
		return fmt.Errorf("error updating elo roles: %w", err)
	}

	return nil
}

func (u *user) updateMemberElo(s *discordgo.Session, guildId string) (err error) {
	eloAndTs := []struct {
		newElo     *int16
		currentElo int16
		teamSize   aoe4api.TeamSize
	}{
		{&u.NewElo.OneVOne, u.CurrentElo.OneVOne, aoe4api.OneVOne},
		{&u.NewElo.TwoVTwo, u.CurrentElo.TwoVTwo, aoe4api.TwoVTwo},
		{&u.NewElo.ThreeVThree, u.CurrentElo.ThreeVThree, aoe4api.ThreeVThree},
		{&u.NewElo.FourVFour, u.CurrentElo.FourVFour, aoe4api.FourVFour},
		{&u.NewElo.Custom, u.CurrentElo.Custom, 5},
	}

	builder := aoe4api.NewRequestBuilder().
		SetUserAgent(config.UserAgent).
		SetSearchPlayer(u.Aoe4Username)

	var wg sync.WaitGroup
	for i, t := range config.Cfg.EloTypes {
		if !t.Enabled {
			continue
		}

		var req aoe4api.Request
		if eloAndTs[i].teamSize == 5 {
			req, err = builder.
				SetMatchType(aoe4api.Custom).
				Request()
		} else {
			req, err = builder.
				SetTeamSize(eloAndTs[i].teamSize).
				Request()
		}
		if err != nil {
			return fmt.Errorf("error building request: %w", err)
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			memberElo, err := req.QueryElo(u.Aoe4Id)
			if err != nil {
				*eloAndTs[i].newElo = eloAndTs[i].currentElo
				// log.Printf("no response from api for %s for elo %s", u.Aoe4Username, [...]string{"1v1", "2v2", "3v3", "4v4", "Custom"}[i])
				return
			}

			*eloAndTs[i].newElo = int16(memberElo)
		}(i)
	}
	wg.Wait()

	if err := db.UpdateUserElo(u.DiscordUserID, guildId, u.NewElo); err != nil {
		return fmt.Errorf("error updating user in db: %w", err)
	}

	return
}

func updateGuildEloRoles(us []db.User, s *discordgo.Session, guildId string) error {
	for _, u := range us {
		user := user(u)
		if err := user.updateMemberEloRoles(s, guildId); errors.Is(err, discordgo.ErrStateNotFound) {
			log.Println(err)
			continue
		} else if err != nil {
			return fmt.Errorf("error getting member elo: %w", err)
		}
	}

	return nil
}

func (u *user) updateMemberEloRoles(s *discordgo.Session, guildId string) error {
	member, err := s.State.Member(guildId, u.DiscordUserID)
	if err != nil {
		return fmt.Errorf("error getting member %s from state: %w", u.DiscordUserID, err)
	}

	highestElo := u.getHighestElo()

eloTypeLoop:
	for _, eloType := range config.Cfg.EloTypes {
		var currentRoleId string
		var currentRolePriority int16 = 9999
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
					return fmt.Errorf("error changing member elo role from %s to %s for member %s: %w", currentRoleId, role.RoleId, u.DiscordUserID, err)
				}
				roleObj, err := s.State.Role(guildId, role.RoleId)
				if err != nil {
					return fmt.Errorf("error getting role %s from state: %w", role.RoleId, err)
				}

				if currentRolePriority > role.RolePriority {
					s.ChannelMessageSend( //nolint:errcheck
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
			return fmt.Errorf("error removing member elo role from %s: %w", currentRoleId, err)
		}
		// TODO
		break //nolint
	}

	return nil
}

func (u *user) getHighestElo() (highestElo int16) {
	eloVals := []int16{
		u.NewElo.OneVOne,
		u.NewElo.TwoVTwo,
		u.NewElo.ThreeVThree,
		u.NewElo.FourVFour,
		u.NewElo.Custom,
	}

	for _, elo := range eloVals {
		if elo > highestElo {
			highestElo = elo
		}
	}

	return
}

func changeMemberEloRole(s *discordgo.Session, m *discordgo.Member, currentRoleId string, newRoleId string) error {
	if currentRoleId != "" {
		if err := s.GuildMemberRoleRemove(m.GuildID, m.User.ID, currentRoleId); err != nil {
			return fmt.Errorf("error removing role: %w", err)
		}
		log.Printf("role %s removed from user %s", currentRoleId, m.Mention())
	}

	if newRoleId != "" {
		if err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, newRoleId); err != nil {
			return fmt.Errorf("error adding role: %w", err)
		}
		log.Printf("role %s added to user %s", newRoleId, m.Mention())
	}

	return nil
}

func (u *user) EloString(name string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s:\n", name))

	eloVals := []int16{
		u.NewElo.OneVOne,
		u.NewElo.TwoVTwo,
		u.NewElo.ThreeVThree,
		u.NewElo.FourVFour,
		u.NewElo.Custom,
	}

	for i, label := range [...]string{"1v1", "2v2", "3v3", "4v4", "Custom"} {
		if config.Cfg.EloTypes[i].Enabled {
			if eloVals[i] == 0 {
				builder.WriteString(fmt.Sprintf("%s: None\n", label))
			} else {
				builder.WriteString(fmt.Sprintf("%s: %d\n", label, eloVals[i]))
			}
		}
	}

	return builder.String()
}
