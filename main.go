package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type (
	usernames struct {
		Usernames []username `json:"usernames"`
	}

	username struct {
		DiscordUserID string `json:"discord_user_id"`
		SteamUsername string `json:"steam_username"`
	}

	payload struct {
		Region       string `json:"region"`
		Versus       string `json:"versus"`
		MatchType    string `json:"matchType"`
		TeamSize     string `json:"teamSize"`
		SearchPlayer string `json:"searchPlayer"`
	}

	response struct {
		Count int `json:"count"`
		Items []struct {
			GameID       string      `json:"gameId"`
			UserID       string      `json:"userId"`
			RlUserID     int         `json:"rlUserId"`
			UserName     string      `json:"userName"`
			AvatarURL    interface{} `json:"avatarUrl"`
			PlayerNumber interface{} `json:"playerNumber"`
			Elo          int         `json:"elo"`
			EloRating    int         `json:"eloRating"`
			Rank         int         `json:"rank"`
			Region       string      `json:"region"`
			Wins         int         `json:"wins"`
			WinPercent   float64     `json:"winPercent"`
			Losses       int         `json:"losses"`
			WinStreak    int         `json:"winStreak"`
		} `json:"items"`
	}
)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.StringVar(&guildID, "g", "", "Guild ID")
	flag.Parse()
}

var (
	token    string
	guildID  string
	eloTypes = [...]string{"1v1", "2v2", "3v3", "4v4"} // a constant value, but Go cannot set arrays as constant, so using var
)

const configPath string = "config/usernames.json"

func main() {
	if token == "" {
		fmt.Println("No token provided.")
		return
	}

	if guildID == "" {
		fmt.Println("No Guild ID provided.")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	c := cron.New()
	c.AddFunc("@midnight", func() { updateAllELO(dg) })

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection to Discord: ", err)
		return
	}

	c.Start()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("AOE4 ELO Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Cron job and Discord session.
	fmt.Println("Shutting down...")
	c.Stop()
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!setELOName") {
		name, err := saveToJSON(s, m)
		if err != nil {
			s.ChannelMessageSendReply(m.ChannelID, "Your Steam username failed to update.", m.MessageReference)
			fmt.Println("error updating username: ", err)
			return
		}
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprint("Steam username for ", m.Author.Username, " has been updated to ", name, "."), m.MessageReference)
	} else if strings.HasPrefix(m.Content, "!updateELO") {
		err := updateAllELO(s)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "ELO failed to update.")
			fmt.Println("error updating elo: ", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, "ELO updated!")
	}
}

func saveToJSON(s *discordgo.Session, m *discordgo.MessageCreate) (string, error) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	jsonBytes, err := configFileToBytes()
	if err != nil {
		return "", errors.New(fmt.Sprint("error converting config file to bytes: ", err))
	}

	var usernames usernames
	json.Unmarshal(jsonBytes, &usernames)

	input := strings.SplitN(m.Content, " ", 2)
	if len(input) <= 1 {
		return "", errors.New("invalid input for username")
	}
	steamUsername := input[1]

	for i, username := range usernames.Usernames {
		if username.DiscordUserID == m.Author.ID {
			usernames.Usernames[i].SteamUsername = steamUsername
			jsonUsernames, err := json.Marshal(usernames)
			if err != nil {
				return "", errors.New(fmt.Sprint("error marshaling usernames: ", err))
			}
			os.WriteFile(configPath, jsonUsernames, 0644)
			return usernames.Usernames[i].SteamUsername, nil
		}
	}

	usernames.Usernames = append(
		usernames.Usernames,
		username{
			DiscordUserID: m.Author.ID,
			SteamUsername: steamUsername,
		},
	)
	jsonUsernames, err := json.Marshal(usernames)
	if err != nil {
		return "", errors.New(fmt.Sprint("error marshaling usernames: ", err))
	}
	os.WriteFile(configPath, jsonUsernames, 0644)
	return steamUsername, nil
}

func updateAllELO(s *discordgo.Session) (err error) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	fmt.Println("Updating ELO...")

	err = removeExistingRoles(s)
	if err != nil {
		return errors.New(fmt.Sprint("error removing existing roles: ", err))
	}

	jsonBytes, err := configFileToBytes()
	if err != nil {
		return errors.New(fmt.Sprint("error converting config file to bytes: ", err))
	}

	var usernames usernames
	err = json.Unmarshal(jsonBytes, &usernames)
	if err != nil {
		return errors.New(fmt.Sprint("error unmarshaling json bytes: ", err))
	}

	for _, username := range usernames.Usernames {
		err = updateMemberELO(username, s, username.DiscordUserID)
		if err != nil {
			return errors.New(fmt.Sprint("error updating member ELO: ", err))
		}
	}
	fmt.Println("ELO Updated!")

	return nil
}

func updateMemberELO(username username, s *discordgo.Session, memberID string) error {
	eloMap, err := curlAPI(username.SteamUsername)
	if err != nil {
		return errors.New(fmt.Sprint("error sending request to api: ", err))
	}
	for _, eloType := range eloTypes {
		if elo, ok := eloMap[eloType]; ok {
			role, err := s.GuildRoleCreate(guildID)
			if err != nil {
				return errors.New(fmt.Sprint("error creating guild role: ", err))
			}
			role, err = s.GuildRoleEdit(guildID, role.ID, fmt.Sprintf("%s ELO: %s", eloType, elo), 1, false, 0, false)
			if err != nil {
				return errors.New(fmt.Sprint("error editing guild role: ", err))
			}
			err = s.GuildMemberRoleAdd(guildID, memberID, role.ID)
			if err != nil {
				return errors.New(fmt.Sprint("error adding guild role: ", err))
			}
		}
	}
	return nil
}

func removeExistingRoles(s *discordgo.Session) error {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return errors.New(fmt.Sprint("error getting roles: ", err))
	}

	for _, role := range roles {
		if strings.Contains(role.Name, "ELO:") {
			err = s.GuildRoleDelete(guildID, role.ID)
			if err != nil {
				return errors.New(fmt.Sprint("error removing role: ", err))
			}
		}
	}
	return nil
}

func curlAPI(username string) (map[string]string, error) {
	respMap := make(map[string]string, 4)
	for _, matchType := range eloTypes {
		data := payload{
			Region:       "7",
			Versus:       "players",
			MatchType:    "unranked",
			TeamSize:     matchType,
			SearchPlayer: username,
		}
		payloadBytes, err := json.Marshal(data)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error marshaling json payload: ", err))
		}
		body := bytes.NewReader(payloadBytes)

		req, err := http.NewRequest("POST", "https://api.ageofempires.com/api/ageiv/Leaderboard", body)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error creating POST request: ", err))
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error sending POST to API: ", err))
		}

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error reading API response: ", err))
		}
		resp.Body.Close()

		if resp.StatusCode == 204 {
			continue
		}

		var respBodyJson response
		err = json.Unmarshal(respBody, &respBodyJson)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error unmarshaling JSON API response: ", err))
		}
		if respBodyJson.Count < 1 {
			continue
		}
		respMap[matchType] = strconv.Itoa(respBodyJson.Items[len(respBodyJson.Items)-1].Elo)
	}

	return respMap, nil
}

func openConfigFile() (*os.File, error) {
	configFile, err := os.Open(configPath)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("Config file does not exist. Creating file usernames.json")
		jsonUsernames, err := json.Marshal(usernames{Usernames: []username{}})
		if err != nil {
			return nil, errors.New(fmt.Sprint("error marshaling json: ", err))
		}
		os.WriteFile(configPath, jsonUsernames, 0644)
		configFile, err = os.Open(configPath)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error opening config file: ", err))
		}
	} else if err != nil {
		return nil, errors.New(fmt.Sprint("error opening config file: ", err))
	}
	return configFile, nil
}

func configFileToBytes() ([]byte, error) {
	configFile, err := openConfigFile()
	if err != nil {
		return nil, errors.New(fmt.Sprint("error opening json file: ", err))
	}
	defer configFile.Close()

	jsonBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, errors.New(fmt.Sprint("error reading json file: ", err))
	}

	return jsonBytes, nil
}
