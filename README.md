# AOE 4 Elo Bot
[![builds.sr.ht status](https://builds.sr.ht/~alexisgeoffrey/aoe4elobot.svg)](https://builds.sr.ht/~alexisgeoffrey/aoe4elobot?)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexisgeoffrey/aoe4elobot/v2)](https://goreportcard.com/report/github.com/alexisgeoffrey/aoe4elobot/v2)

A Discord bot that automatically retrieves Elo ratings for Age of Empires 4 and gives users custom roles.

Uses the public API for the [Age of Empires Leaderboards](https://www.ageofempires.com/stats/ageiv/) via my [aoe4api](https://git.sr.ht/~alexisgeoffrey/aoe4api) library.

## Build Instructions
Before using the AOE 4 Elo Bot backend, a Discord application and bot must be set up and added to a Discord server [here](https://discord.com/developers/applications).
### *Go CLI*
The simplest way to compile and run the bot is directly with the Go CLI. Make sure Go v1.17 or higher is installed.

First, clone the repo and navigate into its directory:
```bash
$ git clone https://git.sr.ht/~alexisgeoffrey/aoe4elobot
$ cd aoe4elobot
```
Then, run the project, replacing the placeholders with their proper values:
```bash
$ go run . -t DISCORD_BOT_TOKEN
```
### *Docker*
A Dockerfile is included in this repo so the bot can be run in a Docker container. First, clone the repo and navigate into its directory as before. Then, build the Docker image:
```bash
$ sudo docker build . -t aoe4elobot
```
Then, create, a volume for the bot config file:
```bash
$ sudo docker volume create aoe4elobot-config
```
Finally, run the image, replacing `token` and `id` with the proper values:
```bash
$ sudo docker run -v aoe4elobot-config:/app/config \
  -e BOT_TOKEN=token \
  aoe4elobot
```
Alternatively, Docker Compose can be used. Here is a sample `docker-compose.yml`:
```yml
services:
  aoe4elobot:
    image: aoe4elobot
    container_name: aoe4elobot
    environment:
      - BOT_TOKEN=token
    volumes:
      - config:/app/config
volumes:
  config:
```
## Server Commands
- `!setEloInfo AOE4_USERNAME, AOE4_ID` - Registers your AOE4 username and ID in the bot to retrieve your Elo rating.
- `!updateElo` - Manually updates Elo ratings for all registered members on the server.
#
Developed with [DiscordGo](https://github.com/bwmarrin/discordgo).

I also used these helpful tools from [Matt Holt](https://github.com/mholt):
- https://mholt.github.io/json-to-go/
- https://mholt.github.io/curl-to-go/
