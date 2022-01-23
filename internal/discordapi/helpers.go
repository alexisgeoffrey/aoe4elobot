package discordapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func formatUpdateMessage(st *discordgo.State, u []user, guildId string) (string, error) {
	var updateMessage strings.Builder
	updateMessage.WriteString("Elo updated!\n\n")

	st.RLock()
	defer st.RUnlock()

	for _, u := range u {
		if u.newElo == u.oldElo {
			continue
		}

		member, err := st.Member(guildId, u.DiscordUserID)
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
			if err := os.MkdirAll("config", 0744); err != nil {
				return nil, fmt.Errorf("error creating config directory: %w", err)
			}
			if err := os.WriteFile(configPath, jsonUsers, 0644); err != nil {
				return nil, fmt.Errorf("error creating config file: %w", err)
			}
			if configFile, err = os.Open(configPath); err != nil {
				return nil, fmt.Errorf("error opening config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error opening config file: %w", err)
		}
	}
	return configFile, nil
}
