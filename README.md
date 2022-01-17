# AOE 4 ELO Bot
[![build](https://img.shields.io/github/workflow/status/alexisgeoffrey/aoe4elobot/Docker%20Build%20&%20Push/main)](https://github.com/alexisgeoffrey/aoe4elobot/actions/workflows/build-push.yml)
[![CodeQL](https://img.shields.io/github/workflow/status/alexisgeoffrey/aoe4elobot/CodeQL/main?label=code%20QL)](https://github.com/alexisgeoffrey/aoe4elobot/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexisgeoffrey/aoe4elobot)](https://goreportcard.com/report/github.com/alexisgeoffrey/aoe4elobot)
[![Docker Image Size](https://img.shields.io/docker/image-size/alexisgeoffrey/aoe4elobot?sort=date)](https://github.com/alexisgeoffrey/aoe4elobot/pkgs/container/aoe4elobot)

A Discord bot that automatically retrieves ELO ratings for Age of Empires 4 and gives users custom roles.
Uses the public API for the Age of Empires Leaderboards from https://www.ageofempires.com/stats/ageiv/

Commands:
- `!setELOName {Steam username}` - Registers your Steam username in the bot to retrieve your ELO.
- `!updateELO` - Manually updates ELO for all registered members on the server.

Developed with the [DiscordGo](https://github.com/bwmarrin/discordgo) library.

I also used these helpful tools from [Matt Holt](https://github.com/mholt):
- https://mholt.github.io/json-to-go/
- https://mholt.github.io/curl-to-go/
