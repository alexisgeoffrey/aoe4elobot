# AOE 4 ELO Bot
This is a Discord bot that automatically retrieves ELO ratings for Age of Empires 4 and gives users custom roles.
Uses the public API for the Age of Empires Leaderboards from https://www.ageofempires.com/stats/ageiv/

Commands:
```
!setELOName {steam username} - Sets your steam username in the bot to retrieve your ELO.

!updateELO - Manually updates ELO for all registered members on the server.
```

Developed with the [DiscordGo](https://github.com/bwmarrin/discordgo) library.

I also used these helpful tools:
- https://mholt.github.io/json-to-go/
- https://mholt.github.io/curl-to-go/
