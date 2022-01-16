# AOE 4 ELO Bot
[![Docker Build & Push](https://github.com/alexisgeoffrey/aoe4elobot/actions/workflows/build-push.yml/badge.svg)](https://github.com/alexisgeoffrey/aoe4elobot/actions/workflows/build-push.yml)
[![CodeQL](https://github.com/alexisgeoffrey/aoe4elobot/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/alexisgeoffrey/aoe4elobot/actions/workflows/codeql-analysis.yml)
[![Total alerts](https://img.shields.io/lgtm/alerts/g/alexisgeoffrey/aoe4elobot.svg?logo=lgtm&logoWidth=18)](https://lgtm.com/projects/g/alexisgeoffrey/aoe4elobot/alerts/)

A Discord bot that automatically retrieves ELO ratings for Age of Empires 4 and gives users custom roles.
Uses the public API for the Age of Empires Leaderboards from https://www.ageofempires.com/stats/ageiv/

Commands:
- `!setELOName {Steam username}` - Registers your Steam username in the bot to retrieve your ELO.
- `!updateELO` - Manually updates ELO for all registered members on the server.

Developed with the [DiscordGo](https://github.com/bwmarrin/discordgo) library.

I also used these helpful tools from [Matt Holt](https://github.com/mholt):
- https://mholt.github.io/json-to-go/
- https://mholt.github.io/curl-to-go/
