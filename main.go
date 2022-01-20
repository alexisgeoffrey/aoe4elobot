package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
		oldElo        userElo
		newElo        userElo
	}

	userElo struct {
		Elo1v1    string
		Elo2v2    string
		Elo3v3    string
		Elo4v4    string
		EloCustom string
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

	safeMap struct {
		respMap map[string]string
		mu      sync.Mutex
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
	eloTypes   = [...]string{"1v1", "2v2", "3v3", "4v4", "Custom"} // a constant value, but Go cannot set arrays as constant, so using var
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

	dg.LogLevel = 2

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
			s.ChannelMessageSendReply(m.ChannelID, "Your Steam username failed to update.", m.Reference())
			fmt.Printf("error updating username: %v\n", err)
			return
		}
		// Send response as a reply to message
		s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Steam username for %s has been updated to %s.", m.Author.Username, name), m.Reference())
	} else if strings.HasPrefix(m.Content, "!updateElo") {
		eventMutex.Lock()
		defer eventMutex.Unlock()

		s.ChannelMessageSend(m.ChannelID, "Updating elo...")
		updateMessage, err := updateAllElo(s)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Elo failed to update.")
			fmt.Printf("error updating elo: %v\n", err)
			return
		}
		s.ChannelMessageSend(m.ChannelID, updateMessage)
	}
}

func saveToConfig(m *discordgo.MessageCreate) (string, error) {
	configBytes, err := configFileToBytes()
	if err != nil {
		return "", fmt.Errorf("error converting config file to bytes: %w", err)
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
				return "", fmt.Errorf("error marshaling users: %w", err)
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
		return "", fmt.Errorf("error marshaling users: %w", err)
	}
	os.WriteFile(configPath, jsonUsers, 0644)
	return steamUsername, nil
}

func updateAllElo(s *discordgo.Session) (string, error) {
	fmt.Println("Updating Elo...")

	configBytes, err := configFileToBytes()
	if err != nil {
		return "", fmt.Errorf("error converting config file to bytes: %w", err)
	}

	var u users
	err = json.Unmarshal(configBytes, &u)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling config bytes: %w", err)
	}

	for _, u := range u.Users {
		memberElo, err := getMemberElo(s.State, u)
		if err != nil {
			return "", fmt.Errorf("error retrieving existing member roles: %w", err)
		}
		u.oldElo = memberElo
	}

	err = removeAllExistingRoles(s)
	if err != nil {
		return "", fmt.Errorf("error removing existing roles: %w", err)
	}

	for _, u := range u.Users {
		memberElo, err := updateMemberElo(s, u)
		if err != nil {
			return "", fmt.Errorf("error updating member Elo: %w", err)
		}
		u.newElo = memberElo
	}

	updateMessage, err := formatUpdateMessage(s.State, u.Users)
	if err != nil {
		return "", fmt.Errorf("error formatting update message: %w", err)
	}

	fmt.Println(updateMessage)

	return updateMessage, nil
}

func getMemberElo(st *discordgo.State, u user) (userElo, error) {
	st.RLock()
	defer st.RUnlock()

	member, err := st.Member(guildID, u.DiscordUserID)
	if err != nil {
		return userElo{}, fmt.Errorf("error retrieving member: %w", err)
	}

	var memberElo userElo
	for _, roleID := range member.Roles {
		role, err := st.Role(guildID, roleID)
		if err != nil {
			fmt.Printf("error retrieving role %s for member %s: %v\n ", roleID, member.User.Username, err)
			continue
		}

		roleName := role.Name
		if err != nil {
			return userElo{}, fmt.Errorf("error getting role info: %w", err)
		}

		if strings.Contains(roleName, "1v1 Elo:") {
			memberElo.Elo1v1 = strings.Split(roleName, " ")[2]
		} else if strings.Contains(roleName, "2v2 Elo:") {
			memberElo.Elo2v2 = strings.Split(roleName, " ")[2]
		} else if strings.Contains(roleName, "3v3 Elo:") {
			memberElo.Elo3v3 = strings.Split(roleName, " ")[2]
		} else if strings.Contains(roleName, "4v4 Elo:") {
			memberElo.Elo4v4 = strings.Split(roleName, " ")[2]
		} else if strings.Contains(roleName, "Custom Elo:") {
			memberElo.Elo4v4 = strings.Split(roleName, " ")[2]
		}
	}

	return memberElo, nil
}

func updateMemberElo(s *discordgo.Session, u user) (userElo, error) {
	eloMap, err := queryAoeApi(u.SteamUsername)
	if err != nil {
		return userElo{}, fmt.Errorf("error sending request to AOE api: %w", err)
	}
	// convert elo map to userElo struct
	var ue userElo
	if elo, ok := eloMap["1v1"]; ok {
		ue.Elo1v1 = elo
	}
	if elo, ok := eloMap["2v2"]; ok {
		ue.Elo2v2 = elo
	}
	if elo, ok := eloMap["3v3"]; ok {
		ue.Elo3v3 = elo
	}
	if elo, ok := eloMap["4v4"]; ok {
		ue.Elo4v4 = elo
	}
	if elo, ok := eloMap["Custom"]; ok {
		ue.EloCustom = elo
	}

	for _, eloType := range eloTypes {
		if elo, ok := eloMap[eloType]; ok {
			role, err := s.GuildRoleCreate(guildID)
			if err != nil {
				return userElo{}, fmt.Errorf("error creating guild role: %w", err)
			}
			role, err = s.GuildRoleEdit(guildID, role.ID, fmt.Sprintf("%s Elo: %s", eloType, elo), 1, false, 0, false)
			if err != nil {
				return userElo{}, fmt.Errorf("error editing guild role: %w", err)
			}
			err = s.GuildMemberRoleAdd(guildID, u.DiscordUserID, role.ID)
			if err != nil {
				return userElo{}, fmt.Errorf("error adding guild role: %w", err)
			}
		}
	}

	return ue, nil
}

func removeAllExistingRoles(s *discordgo.Session) error {
	s.State.RLock()
	defer s.State.RUnlock()

	guild, err := s.State.Guild(guildID)
	if err != nil {
		return fmt.Errorf("error getting guild from state: %w", err)
	}

	for _, role := range guild.Roles {
		if strings.Contains(role.Name, "Elo:") {
			err = s.GuildRoleDelete(guildID, role.ID)
			if err != nil {
				fmt.Printf("error removing role %s: %v\n", role.ID, err)
				continue
			}
		}
	}

	return nil
}

func formatUpdateMessage(st *discordgo.State, u []user) (string, error) {
	var updateMessage strings.Builder
	updateMessage.WriteString("Elo updated!\n\n")

	st.RLock()
	defer st.RUnlock()

	for _, u := range u {
		if u.newElo == u.oldElo {
			continue
		}

		member, err := st.Member(guildID, u.DiscordUserID)
		if err != nil {
			return "", fmt.Errorf("error retrieving member %s name: %w", u.DiscordUserID, err)
		}

		var memberName string
		// check if nickname is assigned
		if member.Nick != "" {
			memberName = member.Nick
		} else {
			memberName = member.User.Username
		}

		updateMessage.WriteString(fmt.Sprint(memberName, ":\n"))
		if oldElo, newElo := u.oldElo.Elo1v1, u.newElo.Elo1v1; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("1v1 Elo:", oldElo, "->", newElo))
		}
		if oldElo, newElo := u.oldElo.Elo2v2, u.newElo.Elo2v2; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("2v2 Elo:", oldElo, "->", newElo))
		}
		if oldElo, newElo := u.oldElo.Elo3v3, u.newElo.Elo3v3; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("3v3 Elo:", oldElo, "->", newElo))
		}
		if oldElo, newElo := u.oldElo.Elo4v4, u.newElo.Elo4v4; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("4v4 Elo:", oldElo, "->", newElo))
		}
		if oldElo, newElo := u.oldElo.EloCustom, u.newElo.EloCustom; oldElo != "" && oldElo != newElo {
			updateMessage.WriteString(fmt.Sprintln("Custom Elo:", oldElo, "->", newElo))
		}
		updateMessage.WriteString("\n")
	}

	return updateMessage.String(), nil
}

func queryAoeApi(username string) (map[string]string, error) {
	respMap := make(map[string]string)
	safeMap := safeMap{respMap: respMap}
	var wg sync.WaitGroup

	for _, matchType := range eloTypes {
		var data payload
		if matchType == "Custom" {
			data = payload{
				Region:       "7",
				Versus:       "players",
				MatchType:    matchType,
				SearchPlayer: username,
			}
		} else {
			data = payload{
				Region:       "7",
				Versus:       "players",
				MatchType:    "unranked",
				TeamSize:     matchType,
				SearchPlayer: username,
			}
		}
		wg.Add(1)
		go func() {
			if err := querySpecificEloAoeApi(data, &safeMap); err != nil {
				fmt.Printf("error retrieving Elo from AOE api for %s: %v", username, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	return respMap, nil
}

func querySpecificEloAoeApi(data payload, respMap *safeMap) error {
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling json payload: %w", err)
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", "https://api.ageofempires.com/api/ageiv/Leaderboard", body)
	if err != nil {
		return fmt.Errorf("error creating POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AOE 4 Elo Bot/0.0.0 (github.com/alexisgeoffrey/aoe4elobot; alexisgeoffrey1@gmail.com)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending POST to API: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading API response: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}

	var respBodyJson response
	err = json.Unmarshal(respBody, &respBodyJson)
	if err != nil {
		return fmt.Errorf("error unmarshaling json API response: %w", err)
	}
	if respBodyJson.Count < 1 {
		return nil
	}

	respMap.mu.Lock()
	respMap.respMap[data.MatchType] = strconv.Itoa(respBodyJson.Items[0].Elo)
	respMap.mu.Unlock()

	return nil
}

func configFileToBytes() ([]byte, error) {
	configFile, err := openOrCreateConfigFile()
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %w", err)
	}
	defer configFile.Close()

	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading json file: %w", err)
	}

	return configBytes, nil
}

func openOrCreateConfigFile() (*os.File, error) {
	configFile, err := os.Open(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("Config file does not exist. Creating file config.json")
			jsonUsers, err := json.Marshal(users{Users: []user{}})
			if err != nil {
				return nil, fmt.Errorf("error marshaling json: %w", err)
			}
			os.MkdirAll("config", 0744)
			if err != nil {
				return nil, fmt.Errorf("error creating config directory: %w", err)
			}
			os.WriteFile(configPath, jsonUsers, 0644)
			if err != nil {
				return nil, fmt.Errorf("error creating config file: %w", err)
			}
			configFile, err = os.Open(configPath)
			if err != nil {
				return nil, fmt.Errorf("error opening config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error opening config file: %w", err)
		}
	}
	return configFile, nil
}
