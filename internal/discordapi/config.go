package discordapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const configPath = "config/config.json"

func parseRegistration(content string, id string) (string, string, error) {
	input := strings.SplitN(content, " ", 2)
	if len(input) <= 1 {
		return "", "", fmt.Errorf("invalid input for info: %s", content)
	}
	infoInput := strings.Split(input[1], ",")
	if len(infoInput) <= 1 {
		return "", "", fmt.Errorf("invalid input for info: %s", content)
	}
	aoe4Username, aoe4Id := strings.TrimSpace(infoInput[0]), strings.TrimSpace(infoInput[1])

	registerUser := user{
		DiscordUserID: id,
		Aoe4Username:  aoe4Username,
		Aoe4Id:        aoe4Id,
	}

	if err := saveToConfig(registerUser); err != nil {
		return "", "", fmt.Errorf("error saving to config file: %w", err)
	}

	return aoe4Username, aoe4Id, nil
}

func saveToConfig(newU user) error {
	configBytes, err := configFileToBytes()
	if err != nil {
		return fmt.Errorf("error converting config file to bytes: %w", err)
	}

	var us users
	json.Unmarshal(configBytes, &us)

	// check if user is already in config file, if so, modify that entry
	for i, u := range us.Users {
		if u.DiscordUserID == newU.DiscordUserID {
			us.Users[i].Aoe4Username = newU.Aoe4Username
			us.Users[i].Aoe4Id = newU.Aoe4Id
			jsonUsers, err := json.Marshal(us)
			if err != nil {
				return fmt.Errorf("error marshaling users: %w", err)
			}
			os.WriteFile(configPath, jsonUsers, 0644)
			return nil
		}
	}

	// if user is not in config file, create a new entry
	us.Users = append(us.Users, newU)
	jsonUsers, err := json.Marshal(us)
	if err != nil {
		return fmt.Errorf("error marshaling users: %w", err)
	}

	if err := os.WriteFile(configPath, jsonUsers, 0644); err != nil {
		return fmt.Errorf("error writing to config file: %w", err)
	}

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
			log.Println("Config file does not exist. Creating file config.json")
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
