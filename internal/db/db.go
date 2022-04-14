package db

import (
	"context"
	"fmt"
	"log"

	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
)

type User struct {
	DiscordUserID string
	Aoe4Username  string
	Aoe4Id        string
	CurrentElo    UserElo
	NewElo        UserElo
}

type UserElo struct {
	OneVOne     int32
	TwoVTwo     int32
	ThreeVThree int32
	FourVFour   int32
	Custom      int32
}

var Db *pgxpool.Pool

func init() {
	// Open connection to user database
	var err error
	if Db, err = pgxpool.Connect(context.Background(), config.Cfg.DbUrl); err != nil {
		log.Fatalf("error connecting to database: %v\n", err)
	}

	if _, err := Db.Exec(context.Background(),
		`create table if not exists users(
		discord_id	varchar(20),
		username	text not null,
		guild_id	varchar(20),
		aoe_id		varchar(40) not null,
		elo_1v1		int,
		elo_2v2		int,
		elo_3v3		int,
		elo_4v4		int,
		elo_custom	int,
		primary key(discord_id, guild_id)
		)`); err != nil {
		log.Fatalf("error setting up database: %v\n", err)
	}
}

func RegisterUser(username string, aoeId string, discordId string, guildId string) error {
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

func UpdateUser(discordId string, guildId string, elo UserElo) error {
	updateUser, err := Db.Exec(context.Background(),
		`update users set elo_1v1 = $1, elo_2v2 = $2, elo_3v3 = $3, elo_4v4 = $4, elo_custom = $5
		where discord_id = $6 and guild_id = $7`,
		elo.OneVOne, elo.TwoVTwo, elo.ThreeVThree, elo.FourVFour, elo.Custom, discordId, guildId)
	if err != nil {
		return fmt.Errorf("error updating user in db: %v", err)
	}
	if updateUser.RowsAffected() != 1 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func GetUser(discordId string, guildId string) (*User, error) {
	row := Db.QueryRow(context.Background(), "select * from users where discord_id = $1 and guild_id = $2", discordId, guildId)

	var oneVOne, twoVTwo, threeVThree, fourVFour, custom pgtype.Int4
	u := &User{}
	if err := row.Scan(
		&u.DiscordUserID,
		&u.Aoe4Username,
		nil,
		&u.Aoe4Id,
		&oneVOne,
		&twoVTwo,
		&threeVThree,
		&fourVFour,
		&custom); err != nil {
		return &User{}, err
	}

	u.pgToInt(oneVOne, twoVTwo, threeVThree, fourVFour, custom)

	return u, nil
}

func GetUsers(guildId string) ([]User, error) {
	rows, err := Db.Query(context.Background(), "select * from users where guild_id = $1", guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var oneVOne, twoVTwo, threeVThree, fourVFour, custom pgtype.Int4
		u := User{}
		if err := rows.Scan(
			&u.DiscordUserID,
			&u.Aoe4Username,
			nil,
			&u.Aoe4Id,
			&oneVOne,
			&twoVTwo,
			&threeVThree,
			&fourVFour,
			&custom); err != nil {
			return nil, err
		}

		u.pgToInt(oneVOne, twoVTwo, threeVThree, fourVFour, custom)

		users = append(users, u)
	}

	return users, nil
}

func (u *User) pgToInt(
	oneVOne pgtype.Int4,
	twoVTwo pgtype.Int4,
	threeVThree pgtype.Int4,
	fourVFour pgtype.Int4,
	custom pgtype.Int4,
) {
	if config.Cfg.OneVOne.Enabled && oneVOne.Status == pgtype.Present {
		u.CurrentElo.OneVOne = oneVOne.Int
	}
	if config.Cfg.TwoVTwo.Enabled && twoVTwo.Status == pgtype.Present {
		u.CurrentElo.TwoVTwo = twoVTwo.Int
	}
	if config.Cfg.ThreeVThree.Enabled && threeVThree.Status == pgtype.Present {
		u.CurrentElo.ThreeVThree = threeVThree.Int
	}
	if config.Cfg.FourVFour.Enabled && fourVFour.Status == pgtype.Present {
		u.CurrentElo.FourVFour = fourVFour.Int
	}
	if config.Cfg.Custom.Enabled && custom.Status == pgtype.Present {
		u.CurrentElo.Custom = custom.Int
	}
}
