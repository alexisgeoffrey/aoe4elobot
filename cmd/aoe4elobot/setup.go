package main

import (
	"context"
	"log"

	"github.com/alexisgeoffrey/aoe4elobot/internal/discordapi"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
)

func setupDb(db *pgxpool.Pool) error {
	if _, err := db.Exec(context.Background(),
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
		return err
	}

	return nil
}

func eloUpdateCron(dg *discordgo.Session) {
	log.Println("Running scheduled Elo update.")

	for _, g := range getGuilds(dg.State) {
		err := discordapi.UpdateAllElo(dg, g.ID)
		if err != nil {
			log.Printf("error updating elo on server %s: %v\n", g.ID, err)
			continue
		}
	}

	log.Println("Scheduled Elo update complete.")
}

func getGuilds(st *discordgo.State) (guilds []*discordgo.Guild) {
	st.RLock()
	defer st.RUnlock()
	copy(guilds, st.Guilds)
	return
}
