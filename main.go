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

type Usernames struct {
	Usernames []Username `json:"usernames"`
}

type Username struct {
	DiscordUsername string `json:"discord_username"`
	SteamUsername   string `json:"steam_username"`
}

type Payload struct {
	Region       string `json:"region"`
	Versus       string `json:"versus"`
	MatchType    string `json:"matchType"`
	TeamSize     string `json:"teamSize"`
	SearchPlayer string `json:"searchPlayer"`
}

type Response struct {
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

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.StringVar(&guildID, "g", "", "Guild ID")
	flag.Parse()
}

var (
	token   string
	guildID string
)

func main() {
	if token == "" {
		fmt.Println("No token provided.")
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
	c.AddFunc("@midnight", func() { updateELO(dg) })

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	c.Start()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("AOE4 ELO Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Cron job and Discord session.
	c.Stop()
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.HasPrefix(m.Content, "!setELOName") {
		name, err := saveToJSON(s, m)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Username failed to update.")
			fmt.Println("error updating username: ", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("@%s Username has been updated to %s.", m.Author.Username, name))
	} else if strings.HasPrefix(m.Content, "!updateELO") {
		err := updateELO(s)
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

	jsonFile, err := openJsonFile()
	if err != nil {
		return "", err
	}
	defer jsonFile.Close()

	var usernames Usernames

	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(jsonBytes, &usernames)

	for _, username := range usernames.Usernames {
		if username.DiscordUsername == m.Author.Username {
			input := strings.SplitAfterN(m.Content, " ", 2)
			if len(input) <= 1 {
				return "", errors.New("invalid input for username")
			}
			username.SteamUsername = strings.SplitAfterN(m.Content, " ", 2)[1]
			jsonUsernames, _ := json.Marshal(usernames)
			os.WriteFile("usernames.json", jsonUsernames, 0644)
			return username.SteamUsername, nil
		}
	}

	return "", nil
}

func updateELO(s *discordgo.Session) (err error) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	roles, err := s.GuildRoles(guildID)
	if err != nil {
		// fmt.Println("error getting roles: ", err)
		return
	}
	members, err := s.GuildMembers(guildID, "", 100)
	if err != nil {
		// fmt.Println("error getting members: ", err)
		return
	}

	for _, role := range roles { // remove existing roles
		if strings.Contains(role.Name, "ELO:") {
			err = s.GuildRoleDelete(guildID, role.ID)
			if err != nil {
				fmt.Println("error removing role: ", err)
				return
			}
		}
	}

	jsonFile, err := openJsonFile()
	if err != nil {
		return
	}
	defer jsonFile.Close()

	var usernames Usernames
	usernameMap := make(map[string]Username, len(usernames.Usernames))

	for _, username := range usernames.Usernames {
		usernameMap[username.DiscordUsername] = username
	}

	jsonBytes, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(jsonBytes, &usernames)

	for _, member := range members { // update elo of each member
		username, ok := usernameMap[member.User.Username]
		if !ok {
			continue
		}
		eloMap, err := curlAPI(username.SteamUsername)
		if err != nil {
			return err
		}
		for eloType, elo := range eloMap {
			role, err := s.GuildRoleCreate(guildID)
			if err != nil {
				return err
			}
			role, err = s.GuildRoleEdit(guildID, role.ID, fmt.Sprintf("%s ELO: %s", eloType, elo), 1, false, 0, false)
			if err != nil {
				return err
			}
			err = s.GuildMemberRoleAdd(guildID, member.User.ID, role.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func curlAPI(username string) (map[string]string, error) {
	respMap := make(map[string]string, 4)
	for _, matchType := range []string{"1v1", "2v2", "3v3", "4v4"} {
		data := Payload{
			Region:       "7",
			Versus:       "players",
			MatchType:    "unranked",
			TeamSize:     matchType,
			SearchPlayer: username,
		}
		payloadBytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body := bytes.NewReader(payloadBytes)

		req, err := http.NewRequest("POST", "https://api.ageofempires.com/api/ageiv/Leaderboard", body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()

		var respBodyJson Response
		err = json.Unmarshal(respBody, &respBodyJson)
		if err != nil {
			return nil, err
		}
		if respBodyJson.Count < 1 {
			continue
		}
		respMap[matchType] = strconv.Itoa(respBodyJson.Items[len(respBodyJson.Items)-1].Elo)
	}

	return respMap, nil
}

func openJsonFile() (*os.File, error) {
	jsonFile, err := os.Open("usernames.json")
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("File does not exist. Creating file usernames.json")
		jsonUsernames, err := json.Marshal(Usernames{Usernames: []Username{}})
		if err != nil {
			return nil, errors.New(fmt.Sprint("error marshaling json: ", err))
		}
		os.WriteFile("usernames.json", jsonUsernames, 0644)
		jsonFile, err = os.Open("usernames.json")
		if err != nil {
			return nil, errors.New(fmt.Sprint("error opening jsonfile: ", err))
		}
	}
	if err != nil {
		return nil, errors.New(fmt.Sprint("error opening jsonfile: ", err))
	}
	return jsonFile, nil
}
