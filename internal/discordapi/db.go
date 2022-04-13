package discordapi

import (
	"context"
	"fmt"

	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
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

func updateUser(discordId string, guildId string, elo userElo) error {
	updateUser, err := Db.Exec(context.Background(),
		`update users set elo_1v1 = $1, elo_2v2 = $2, elo_3v3 = $3, elo_4v4 = $4, elo_custom = $5
		where discord_id = $6 and guild_id = $7`,
		elo.oneVOne, elo.twoVTwo, elo.threeVThree, elo.fourVFour, elo.custom, discordId, guildId)
	if err != nil {
		return fmt.Errorf("error updating user in db: %v", err)
	}
	if updateUser.RowsAffected() != 1 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func getUser(discordId string, guildId string) (user, error) {
	row := Db.QueryRow(context.Background(), "select * from users where discord_id = $1 and guild_id = $2", discordId, guildId)

	var oneVOne, twoVTwo, threeVThree, fourVFour, custom pgtype.Int4
	u := user{}
	if err := row.Scan(
		&u.discordUserID,
		&u.aoe4Username,
		nil,
		&u.aoe4Id,
		&oneVOne,
		&twoVTwo,
		&threeVThree,
		&fourVFour,
		&custom); err != nil {
		return user{}, err
	}

	if config.Config.OneVOne.Enabled && oneVOne.Status == pgtype.Present {
		u.currentElo.oneVOne = oneVOne.Int
	}
	if config.Config.TwoVTwo.Enabled && twoVTwo.Status == pgtype.Present {
		u.currentElo.twoVTwo = twoVTwo.Int
	}
	if config.Config.ThreeVThree.Enabled && threeVThree.Status == pgtype.Present {
		u.currentElo.threeVThree = threeVThree.Int
	}
	if config.Config.FourVFour.Enabled && fourVFour.Status == pgtype.Present {
		u.currentElo.fourVFour = fourVFour.Int
	}
	if config.Config.Custom.Enabled && custom.Status == pgtype.Present {
		u.currentElo.custom = custom.Int
	}

	return u, nil
}

func getUsers(guildId string) ([]*user, error) {
	rows, err := Db.Query(context.Background(), "select * from users where guild_id = $1", guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*user

	for rows.Next() {
		var oneVOne, twoVTwo, threeVThree, fourVFour, custom pgtype.Int4
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

		if config.Config.OneVOne.Enabled && oneVOne.Status == pgtype.Present {
			u.currentElo.oneVOne = oneVOne.Int
		}
		if config.Config.TwoVTwo.Enabled && twoVTwo.Status == pgtype.Present {
			u.currentElo.twoVTwo = twoVTwo.Int
		}
		if config.Config.ThreeVThree.Enabled && threeVThree.Status == pgtype.Present {
			u.currentElo.threeVThree = threeVThree.Int
		}
		if config.Config.FourVFour.Enabled && fourVFour.Status == pgtype.Present {
			u.currentElo.fourVFour = fourVFour.Int
		}
		if config.Config.Custom.Enabled && custom.Status == pgtype.Present {
			u.currentElo.custom = custom.Int
		}

		users = append(users, &u)
	}

	return users, nil
}
