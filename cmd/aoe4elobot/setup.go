package main

import (
	"log"

	"github.com/alexisgeoffrey/aoe4elobot/internal/discordapi"
	"github.com/bwmarrin/discordgo"
)

func eloUpdateCron(dg *discordgo.Session) {
	log.Println("Running scheduled Elo update.")

	for _, id := range getGuildIds(dg.State) {
		err := discordapi.UpdateGuildElo(dg, id)
		if err != nil {
			log.Printf("error updating elo on server %s: %v\n", id, err)
			continue
		}
	}

	log.Println("Scheduled Elo update complete.")
}

func getGuildIds(st *discordgo.State) []string {
	st.RLock()
	defer st.RUnlock()

	guildIds := make([]string, len(st.Guilds))
	for _, guild := range st.Guilds {
		guildIds = append(guildIds, guild.ID)
	}
	return guildIds
}
