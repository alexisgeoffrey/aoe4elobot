package db

import (
	"context"
	"fmt"
	"log"

	"github.com/alexisgeoffrey/aoe4elobot/v2/internal/config"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4/pgxpool"
)

type User struct {
	DiscordUserID string
	Aoe4Username  string
	Aoe4Id        string
	CurrentElo    userElo
	NewElo        userElo
}

type userElo struct {
	OneVOne     int16
	TwoVTwo     int16
	ThreeVThree int16
	FourVFour   int16
	Custom      int16
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
		 elo_1v1	smallint,
		 elo_2v2	smallint,
		 elo_3v3	smallint,
		 elo_4v4	smallint,
		 elo_custom	smallint,
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
		return fmt.Errorf("error updating user in db: %w", err)
	}
	if updateUser.RowsAffected() == 0 {
		if _, err := Db.Exec(context.Background(),
			"insert into users(username, aoe_id, discord_id, guild_id) values($1, $2, $3, $4)",
			username, aoeId, discordId, guildId); err != nil {
			return fmt.Errorf("error inserting user in db: %w", err)
		}
	}

	return nil
}

func UpdateUserElo(discordId string, guildId string, elo userElo) error {
	updateUser, err := Db.Exec(context.Background(),
		`update users set elo_1v1 = $1, elo_2v2 = $2, elo_3v3 = $3, elo_4v4 = $4, elo_custom = $5
		 where discord_id = $6 and guild_id = $7`,
		elo.OneVOne, elo.TwoVTwo, elo.ThreeVThree, elo.FourVFour, elo.Custom, discordId, guildId)
	if err != nil {
		return fmt.Errorf("error updating user in db: %w", err)
	}
	if updateUser.RowsAffected() != 1 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func GetUser(discordId string, guildId string) (*User, error) {
	row := Db.QueryRow(context.Background(), "select * from users where discord_id = $1 and guild_id = $2", discordId, guildId)

	u := &User{}
	var pgElo [5]pgtype.Int2
	if err := row.Scan(
		&u.DiscordUserID,
		&u.Aoe4Username,
		nil,
		&u.Aoe4Id,
		&pgElo[0],
		&pgElo[1],
		&pgElo[2],
		&pgElo[3],
		&pgElo[4]); err != nil {
		return nil, err
	}

	u.pgToCurrentElo(pgElo)

	return u, nil
}

func GetUsers(guildId string) (users []User, err error) {
	rows, err := Db.Query(context.Background(), "select * from users where guild_id = $1", guildId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u User
		var pgElo [5]pgtype.Int2
		if err := rows.Scan(
			&u.DiscordUserID,
			&u.Aoe4Username,
			nil,
			&u.Aoe4Id,
			&pgElo[0],
			&pgElo[1],
			&pgElo[2],
			&pgElo[3],
			&pgElo[4]); err != nil {
			return nil, err
		}

		u.pgToCurrentElo(pgElo)

		users = append(users, u)
	}

	return
}

func (u *User) pgToCurrentElo(pgElo [5]pgtype.Int2) {
	enabledAndCurrent := [...]struct {
		enabled    bool
		currentElo *int16
	}{
		{config.Cfg.OneVOne.Enabled, &u.CurrentElo.OneVOne},
		{config.Cfg.TwoVTwo.Enabled, &u.CurrentElo.TwoVTwo},
		{config.Cfg.ThreeVThree.Enabled, &u.CurrentElo.ThreeVThree},
		{config.Cfg.FourVFour.Enabled, &u.CurrentElo.FourVFour},
		{config.Cfg.Custom.Enabled, &u.CurrentElo.Custom},
	}

	for i, elo := range pgElo {
		if enabledAndCurrent[i].enabled && elo.Status == pgtype.Present {
			*enabledAndCurrent[i].currentElo = elo.Int
		}
	}
}
