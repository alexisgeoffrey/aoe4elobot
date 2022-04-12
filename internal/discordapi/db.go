package discordapi

import (
	"context"
	"fmt"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
)

var Db *pgxpool.Pool

func registerUser(username string, aoeId string, discordId string, guildId string) error {
	updateUser, err := Db.Exec(context.Background(),
		"update users set username = $1, aoe_id = $2 where discord_id = $3 and guild_id = $4",
		username, aoeId, discordId, guildId)
	if err != nil {
		return fmt.Errorf("error updating user in db: %v", err)
	}
	if updateUser.RowsAffected() == 0 {
		_, err := Db.Exec(context.Background(),
			"insert into users(username, aoe_id, discord_id, guild_id) values($1, $2, $3, $4)",
			username, aoeId, discordId, guildId)
		if err != nil {
			return fmt.Errorf("error inserting user in db: %v", err)
		}
	}
	return nil
}

func getUsers(guildId string) ([]user, error) {
	rows, err := Db.Query(context.Background(), "select * from users where guild_id = $1", guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []user
	var oneVOne, twoVTwo, threeVThree, fourVFour, custom pgtype.Int4

	for rows.Next() {
		u := user{}
		if err := rows.Scan(
			&u.discordUserID,
			&u.aoe4Username,
			nil,
			&u.aoe4Id,
			&oneVOne,
			&twoVTwo,
			&threeVThree,
			&fourVFour,
			&custom); err != nil {
			return nil, err
		}

		if Config.OneVOne.Enabled && oneVOne.Status == pgtype.Present {
			u.oldElo.oneVOne = oneVOne.Int
		}
		if Config.TwoVTwo.Enabled && twoVTwo.Status == pgtype.Present {
			u.oldElo.twoVTwo = twoVTwo.Int
		}
		if Config.ThreeVThree.Enabled && threeVThree.Status == pgtype.Present {
			u.oldElo.threeVThree = threeVThree.Int
		}
		if Config.FourVFour.Enabled && fourVFour.Status == pgtype.Present {
			u.oldElo.fourVFour = fourVFour.Int
		}
		if Config.Custom.Enabled && custom.Status == pgtype.Present {
			u.oldElo.custom = custom.Int
		}

		users = append(users, u)
	}

	return users, nil
}
