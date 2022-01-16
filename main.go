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
	Usernames struct {
		Usernames []Username `json:"usernames"`
	}

	Username struct {
		DiscordUserID string `json:"discord_user_id"`
		SteamUsername string `json:"steam_username"`
	}

	Payload struct {
		Region       string `json:"region"`
		Versus       string `json:"versus"`
		MatchType    string `json:"matchType"`
		TeamSize     string `json:"teamSize"`
		SearchPlayer string `json:"searchPlayer"`
	}

	Response struct {
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
	token     string
	guildID   string
	ELO_TYPES = [...]string{"1v1", "2v2", "3v3", "4v4"} // a constant value, but Go cannot set arrays as constant, so using var
)

const CONFIG_PATH string = "config/usernames.json"

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
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	c := cron.New()
	c.AddFunc("@midnight", func() { updateELO(dg) })

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection to Discord:", err)
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
			s.ChannelMessageSend(m.ChannelID, "Username failed to update.")
			fmt.Println("error updating username:", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s Username has been updated to %s.", m.Author.Mention(), name))
	} else if strings.HasPrefix(m.Content, "!updateELO") {
		err := updateELO(s)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "ELO failed to update.")
			fmt.Println("error updating elo:", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, "ELO updated!")
	}
}

func saveToJSON(s *discordgo.Session, m *discordgo.MessageCreate) (string, error) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	jsonFile, err := openJsonFile()
	if err != nil {
		return "", errors.New(fmt.Sprint("error opening json file:", err))
	}
	defer jsonFile.Close()

	var usernames Usernames

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return "", errors.New(fmt.Sprint("error reading json file:", err))
	}
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
				return "", errors.New(fmt.Sprint("error marshaling usernames:", err))
			}
			os.WriteFile(CONFIG_PATH, jsonUsernames, 0644)
			return usernames.Usernames[i].SteamUsername, nil
		}
	}

	usernames.Usernames = append(
		usernames.Usernames,
		Username{
			DiscordUserID: m.Author.ID,
			SteamUsername: steamUsername,
		},
	)
	jsonUsernames, err := json.Marshal(usernames)
	if err != nil {
		return "", errors.New(fmt.Sprint("error marshaling usernames:", err))
	}
	os.WriteFile(CONFIG_PATH, jsonUsernames, 0644)
	return steamUsername, nil
}

func updateELO(s *discordgo.Session) (err error) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	fmt.Println("Updating ELO...")

	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return errors.New(fmt.Sprint("error getting roles:", err))
	}
	members, err := s.GuildMembers(guildID, "", 100)
	if err != nil {
		return errors.New(fmt.Sprint("error getting members:", err))
	}

	for _, role := range roles { // remove existing roles
		if strings.Contains(role.Name, "ELO:") {
			err = s.GuildRoleDelete(guildID, role.ID)
			if err != nil {
				return errors.New(fmt.Sprint("error removing role:", err))
			}
		}
	}

	jsonFile, err := openJsonFile()
	if err != nil {
		return errors.New(fmt.Sprint("error opening json file:", err))
	}
	defer jsonFile.Close()

	var usernames Usernames
	usernameMap := make(map[string]Username, len(usernames.Usernames))

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return errors.New(fmt.Sprint("error reading json file:", err))
	}
	err = json.Unmarshal(jsonBytes, &usernames)
	if err != nil {
		return errors.New(fmt.Sprint("error unmarshaling json bytes:", err))
	}

	for _, username := range usernames.Usernames {
		usernameMap[username.DiscordUserID] = username
	}

	for _, member := range members { // update elo of each member
		username, ok := usernameMap[member.User.ID]
		if !ok {
			continue
		}
		eloMap, err := curlAPI(username.SteamUsername)
		if err != nil {
			return errors.New(fmt.Sprint("error sending request to api:", err))
		}
		for _, eloType := range ELO_TYPES {
			if elo, ok := eloMap[eloType]; ok {
				role, err := s.GuildRoleCreate(guildID)
				if err != nil {
					return errors.New(fmt.Sprint("error creating guild role:", err))
				}
				role, err = s.GuildRoleEdit(guildID, role.ID, fmt.Sprintf("%s ELO: %s", eloType, elo), 1, false, 0, false)
				if err != nil {
					return errors.New(fmt.Sprint("error editing guild role:", err))
				}
				err = s.GuildMemberRoleAdd(guildID, member.User.ID, role.ID)
				if err != nil {
					return errors.New(fmt.Sprint("error adding guild role:", err))
				}
			}
		}
	}
	fmt.Println("ELO Updated!")

	return nil
}

func curlAPI(username string) (map[string]string, error) {
	respMap := make(map[string]string, 4)
	for _, matchType := range ELO_TYPES {
		data := Payload{
			Region:       "7",
			Versus:       "players",
			MatchType:    "unranked",
			TeamSize:     matchType,
			SearchPlayer: username,
		}
		payloadBytes, err := json.Marshal(data)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error marshaling json payload:", err))
		}
		body := bytes.NewReader(payloadBytes)

		req, err := http.NewRequest("POST", "https://api.ageofempires.com/api/ageiv/Leaderboard", body)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error creating POST request:", err))
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error sending POST to API:", err))
		}

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error reading API response:", err))
		}
		resp.Body.Close()

		if resp.StatusCode == 204 {
			continue
		}

		var respBodyJson Response
		err = json.Unmarshal(respBody, &respBodyJson)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error unmarshaling JSON API response:", err))
		}
		if respBodyJson.Count < 1 {
			continue
		}
		respMap[matchType] = strconv.Itoa(respBodyJson.Items[len(respBodyJson.Items)-1].Elo)
	}

	return respMap, nil
}

func openJsonFile() (*os.File, error) {
	jsonFile, err := os.Open(CONFIG_PATH)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("Config file does not exist. Creating file usernames.json")
		jsonUsernames, err := json.Marshal(Usernames{Usernames: []Username{}})
		if err != nil {
			return nil, errors.New(fmt.Sprint("error marshaling json: ", err))
		}
		os.WriteFile(CONFIG_PATH, jsonUsernames, 0644)
		jsonFile, err = os.Open(CONFIG_PATH)
		if err != nil {
			return nil, errors.New(fmt.Sprint("error opening jsonfile: ", err))
		}
	} else if err != nil {
		return nil, errors.New(fmt.Sprint("error opening jsonfile: ", err))
	}
	return jsonFile, nil
}
