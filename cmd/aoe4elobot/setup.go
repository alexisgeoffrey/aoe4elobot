package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alexisgeoffrey/aoe4elobot/internal/discordapi"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v4/pgxpool"
	"gopkg.in/yaml.v3"
)

var (
	sampleEloRoles = []*discordapi.EloRole{
		{
			RoleId:      "eloRoleId1",
			StartingElo: 500,
			EndingElo:   1000,
		},
		{
			RoleId:      "eloRoleId2",
			StartingElo: 1001,
			EndingElo:   2000,
		},
	}

	sampleAdminRoles = []string{"adminRoleId1", "adminRoleId2"}
)

func setupDb(db *pgxpool.Pool) error {
	if _, err := db.Exec(context.Background(),
		`create table if not exists users(
		discord_id	varchar(20) primary key,
		username	text not null,
		guild_id	char(20) not null,
		aoe_id		char(40) not null,
		elo_1v1		int,
		elo_2v2		int,
		elo_3v3		int,
		elo_4v4		int,
		elo_custom	int
		)`); err != nil {
		return err
	}

	return nil
}

func genConfig(path string) error {
	discordapi.Config.OneVOne.Enabled = true
	discordapi.Config.OneVOne.Roles = sampleEloRoles
	discordapi.Config.AdminRoles = sampleAdminRoles

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating config file: %v", err)
	}
	defer file.Close()

	yamlBytes, err := yaml.Marshal(&discordapi.Config)
	if err != nil {
		return fmt.Errorf("error marshaling yaml struct: %v", err)
	}

	if _, err := file.Write(yamlBytes); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	log.Println("Config file does not exist. Creating...")
	return nil
}

// func eloUpdateCron(dg *discordgo.Session) {
// 	log.Println("Running scheduled Elo update.")

// 	for _, g := range getGuilds(dg.State) {
// 		updateMessage, err := discordapi.UpdateAllElo(dg, g.ID)
// 		if err != nil {
// 			log.Printf("error updating elo on server %s: %v\n", g.ID, err)
// 			continue
// 		}
// 		dg.ChannelMessageSend(g.Channels[0].ID, updateMessage)
// 	}

// 	log.Println("Scheduled Elo update complete.")
// }

func getGuilds(st *discordgo.State) (guilds []*discordgo.Guild) {
	st.RLock()
	defer st.RUnlock()
	copy(guilds, st.Guilds)
	return
}
