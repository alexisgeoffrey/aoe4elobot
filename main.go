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
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

type (
	users struct {
		Users []user `json:"users"`
	}

	user struct {
		DiscordUserID string `json:"discord_user_id"`
		SteamUsername string `json:"steam_username"`
	}

	userElo struct {
		Elo1v1 string
		Elo2v2 string
		Elo3v3 string
		Elo4v4 string
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
	token      string
	guildID    string
	eloTypes   = [...]string{"1v1", "2v2", "3v3", "4v4"} // a constant value, but Go cannot set arrays as constant, so using var
	eventMutex sync.Mutex
)

const configPath string = "config/config.json"

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

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMessages

	c := cron.New()
	c.AddFunc("@midnight", func() {
		fmt.Println("Running scheduled Elo update.")
		updateAllElo(dg)
	})

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection to Discord: ", err)
		return
	}

	c.Start()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("AOE4 Elo Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Cron job and Discord session.
	fmt.Println("Shutting down...")
	c.Stop()
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!setEloName") {
		eventMutex.Lock()
		defer eventMutex.Unlock()

		name, err := saveToConfig(m)
		if err != nil {
			s.ChannelMessageSendReply(m.ChannelID, "Your Steam username failed to update.", m.MessageReference)
			fmt.Println("error updating username: ", err)
			return
		}
		// Send response as a reply to message
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprint("Steam username for ", m.Author.Username, " has been updated to ", name, "."), m.MessageReference)
	} else if strings.HasPrefix(m.Content, "!updateElo") {
		eventMutex.Lock()
		defer eventMutex.Unlock()

		s.ChannelMessageSend(m.ChannelID, "Updating elo...")
		updateMessage, err := updateAllElo(s)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Elo failed to update.")
			fmt.Println("error updating elo: ", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, updateMessage)
	}
}

func saveToConfig(m *discordgo.MessageCreate) (string, error) {
	configBytes, err := configFileToBytes()
	if err != nil {
		return "", errors.New(fmt.Sprint("error converting config file to bytes: ", err))
	}

	var users users
	json.Unmarshal(configBytes, &users)

	input := strings.SplitN(m.Content, " ", 2)
	if len(input) <= 1 {
		return "", errors.New("invalid input for username")
	}
	steamUsername := input[1]

	// check if user is already in config file, if so, modify that entry
	for i, user := range users.Users {
		if user.DiscordUserID == m.Author.ID {
			users.Users[i].SteamUsername = steamUsername
			jsonUsers, err := json.Marshal(users)
			if err != nil {
				return "", errors.New(fmt.Sprint("error marshaling users: ", err))
			}
			os.WriteFile(configPath, jsonUsers, 0644)
			return users.Users[i].SteamUsername, nil
		}
	}

	// if user is not in config file, create a new entry
	users.Users = append(
		users.Users,
		user{
			DiscordUserID: m.Author.ID,
			SteamUsername: steamUsername,
		},
	)
	jsonUsers, err := json.Marshal(users)
	if err != nil {
		return "", errors.New(fmt.Sprint("error marshaling users: ", err))
	}
	os.WriteFile(configPath, jsonUsers, 0644)
	return steamUsername, nil
}

func updateAllElo(s *discordgo.Session) (string, error) {
	fmt.Println("Updating Elo...")

	configBytes, err := configFileToBytes()
	if err != nil {
		return "", errors.New(fmt.Sprint("error converting config file to bytes: ", err))
	}

	var users users
	err = json.Unmarshal(configBytes, &users)
	if err != nil {
		return "", errors.New(fmt.Sprint("error unmarshaling json bytes: ", err))
	}

	allMemberOldElo := make(map[string]userElo)
	for _, user := range users.Users {
		memberElo, err := getMemberElo(s.State, user)
		if err != nil {
			return "", errors.New(fmt.Sprint("error retrieving existing member roles: ", err))
		}
		allMemberOldElo[user.DiscordUserID] = memberElo
	}

	err = removeAllExistingRoles(s)
	if err != nil {
		return "", errors.New(fmt.Sprint("error removing existing roles: ", err))
	}

	allMemberNewElo := make(map[string]userElo)
	for _, user := range users.Users {
		memberElo, err := updateMemberElo(s, user)
		if err != nil {
			return "", errors.New(fmt.Sprint("error updating member Elo: ", err))
		}
		allMemberNewElo[user.DiscordUserID] = memberElo
	}

	updateMessage, err := formatUpdateMessage(s.State, allMemberOldElo, allMemberNewElo)
	if err != nil {
		return "", errors.New(fmt.Sprint("error formatting update message: ", err))
	}

	fmt.Println(updateMessage)

	return updateMessage, nil
}

func formatUpdateMessage(st *discordgo.State, oldElo map[string]userElo, newElo map[string]userElo) (string, error) {
	var updateMessage strings.Builder
	updateMessage.WriteString("Elo updated!\n\n")

	st.RLock()
	defer st.RUnlock()

	for userID, oldMemberElo := range oldElo {
		if newElo[userID] == oldMemberElo {
			continue
		}

		member, err := st.Member(guildID, userID)
		if err != nil {
			return "", errors.New(fmt.Sprint("error retrieving member name: ", err))
		}

		var memberName string
		// check if nickname is assigned
		if member.Nick != "" {
			memberName = member.Nick
		} else {
			memberName = member.User.Username
		}

		updateMessage.WriteString(fmt.Sprint(memberName, ":\n"))
		if oldElo, newElo := oldMemberElo.Elo1v1, newElo[userID].Elo1v1; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("1v1 Elo:", oldElo, "->", newElo))
		}
		if oldElo, newElo := oldMemberElo.Elo2v2, newElo[userID].Elo2v2; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("2v2 Elo:", oldElo, "->", newElo))
		}
		if oldElo, newElo := oldMemberElo.Elo3v3, newElo[userID].Elo3v3; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("3v3 Elo:", oldElo, "->", newElo))
		}
		if oldElo, newElo := oldMemberElo.Elo4v4, newElo[userID].Elo4v4; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("4v4 Elo:", oldElo, "->", newElo))
		}
		updateMessage.WriteString("\n")
	}

	return updateMessage.String(), nil
}

func updateMemberElo(s *discordgo.Session, u user) (userElo, error) {
	eloMap, err := queryElo(u.SteamUsername)
	if err != nil {
		return userElo{}, errors.New(fmt.Sprint("error sending request to api: ", err))
	}
	for _, eloType := range eloTypes {
		if elo, ok := eloMap[eloType]; ok {
			role, err := s.GuildRoleCreate(guildID)
			if err != nil {
				return userElo{}, errors.New(fmt.Sprint("error creating guild role: ", err))
			}
			role, err = s.GuildRoleEdit(guildID, role.ID, fmt.Sprintf("%s Elo: %s", eloType, elo), 1, false, 0, false)
			if err != nil {
				return userElo{}, errors.New(fmt.Sprint("error editing guild role: ", err))
			}
			err = s.GuildMemberRoleAdd(guildID, u.DiscordUserID, role.ID)
			if err != nil {
				return userElo{}, errors.New(fmt.Sprint("error adding guild role: ", err))
			}
		}
	}
	// convert elo map to userElo struct
	var userElo userElo
	if elo, ok := eloMap["1v1"]; ok {
		userElo.Elo1v1 = elo
	}
	if elo, ok := eloMap["2v2"]; ok {
		userElo.Elo2v2 = elo
	}
	if elo, ok := eloMap["3v3"]; ok {
		userElo.Elo3v3 = elo
	}
	if elo, ok := eloMap["4v4"]; ok {
		userElo.Elo4v4 = elo
	}
	return userElo, nil
}

func getMemberElo(st *discordgo.State, u user) (userElo, error) {
	st.RLock()
	defer st.RUnlock()

	member, err := st.Member(guildID, u.DiscordUserID)
	if err != nil {
		return userElo{}, errors.New(fmt.Sprint("error retrieving member: ", err))
	}

	var memberElo userElo
	for _, roleID := range member.Roles {
		role, err := st.Role(guildID, roleID)
		if err != nil {
			fmt.Println("error retrieving role ", roleID, " for member ", member.User.Username, ": ", err)
			continue
		}

		roleName := role.Name
		if err != nil {
			return userElo{}, errors.New(fmt.Sprint("error getting role info: ", err))
		}

		if strings.Contains(roleName, "1v1 Elo:") {
			memberElo.Elo1v1 = strings.Split(roleName, " ")[2]
		} else if strings.Contains(roleName, "2v2 Elo:") {
			memberElo.Elo2v2 = strings.Split(roleName, " ")[2]
		} else if strings.Contains(roleName, "3v3 Elo:") {
			memberElo.Elo3v3 = strings.Split(roleName, " ")[2]
		} else if strings.Contains(roleName, "4v4 Elo:") {
			memberElo.Elo4v4 = strings.Split(roleName, " ")[2]
		}
	}

	return memberElo, nil
}

func removeAllExistingRoles(s *discordgo.Session) error {
	s.State.RLock()
	defer s.State.RUnlock()

	guild, err := s.State.Guild(guildID)
	if err != nil {
		return errors.New(fmt.Sprint("error getting guild from state: ", err))
	}

	for _, role := range guild.Roles {
		if strings.Contains(role.Name, "Elo:") {
			err = s.GuildRoleDelete(guildID, role.ID)
			if err != nil {
				fmt.Println("error removing role ", role.ID+":", err)
				continue
			}
		}
	}

	return nil
}

func queryElo(username string) (map[string]string, error) {
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
		req.Header.Set("User-Agent", "AOE 4 Elo Bot/0.0.0 (github.com/alexisgeoffrey/aoe4elobot; alexisgeoffrey1@gmail.com)")

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
			return nil, errors.New(fmt.Sprint("error unmarshaling json API response: ", err))
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
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Config file does not exist. Creating file config.json")
			jsonUsers, err := json.Marshal(users{Users: []user{}})
			if err != nil {
				return nil, errors.New(fmt.Sprint("error marshaling json: ", err))
			}
			os.WriteFile(configPath, jsonUsers, 0644)
			configFile, err = os.Open(configPath)
			if err != nil {
				return nil, errors.New(fmt.Sprint("error opening config file: ", err))
			}
		} else {
			return nil, errors.New(fmt.Sprint("error opening config file: ", err))
		}
	}
	return configFile, nil
}

func configFileToBytes() ([]byte, error) {
	configFile, err := openConfigFile()
	if err != nil {
		return nil, errors.New(fmt.Sprint("error opening json file: ", err))
	}
	defer configFile.Close()

	configBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, errors.New(fmt.Sprint("error reading json file: ", err))
	}

	return configBytes, nil
}
