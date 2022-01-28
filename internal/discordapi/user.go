package discordapi

import (
	"fmt"
	"strings"

	"github.com/alexisgeoffrey/aoe4elobot/pkg/aoeapi"
	"github.com/bwmarrin/discordgo"
)

type user struct {
	DiscordUserID string `json:"discord_user_id"`
	SteamUsername string `json:"steam_username"`
	oldElo        userElo
	newElo        userElo
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
