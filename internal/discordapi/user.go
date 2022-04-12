package discordapi

type (
	user struct {
		discordUserID string
		aoe4Username  string
		aoe4Id        string
		oldElo        userElo
		newElo        userElo
	}

	userElo struct {
		oneVOne     int32
		twoVTwo     int32
		threeVThree int32
		fourVFour   int32
		custom      int32
	}
)

// func (u user) getMemberElo(st *discordgo.State, guildId string) (userElo, error) {
// 	member, err := st.Member(guildId, u.discordUserID)
// 	if err != nil {
// 		return nil, fmt.Errorf("error retrieving member: %w", err)
// 	}

// 	memberElo := map[string]string{}
// 	for _, roleID := range member.Roles {
// 		role, err := st.Role(guildId, roleID)
// 		if err != nil {
// 			log.Printf("error retrieving role %s for member %s: %v\n ", roleID, member.User.Username, err)
// 			continue
// 		}

// 		roleName := role.Name
// 		if err != nil {
// 			return nil, fmt.Errorf("error getting role info: %w", err)
// 		}

// 		for _, eloT := range getEloTypes() {
// 			if strings.Contains(roleName, cases.Title(language.English).String(eloT)+" Elo:") {
// 				memberElo[eloT] = strings.Split(roleName, " ")[2]
// 			}
// 		}
// 	}

// 	return memberElo, nil
// }
