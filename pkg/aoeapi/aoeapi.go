package aoeapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
)

type (
	safeMap struct {
		sync.Mutex
		respMap map[string]string
	}

	Payload struct {
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

func GetEloTypes() [5]string {
	return [...]string{"1v1", "2v2", "3v3", "4v4", "Custom"}
}

func Query(data Payload, matchType string) (string, error) {
	sm := &safeMap{respMap: make(map[string]string)}

	if err := queryToMap(data, matchType, sm); err != nil {
		return "", fmt.Errorf("error querying aoe api and/or inserting in map: %w", err)
	}

	if elo, ok := sm.respMap[matchType]; ok {
		return elo, nil
	}
	return "", fmt.Errorf("no elo value found for match type %s for username %s", matchType, data.SearchPlayer)
}

func QueryAll(username string) (map[string]string, error) {
	var wg sync.WaitGroup
	sm := &safeMap{respMap: make(map[string]string)}

	for _, matchType := range GetEloTypes() {
		var data Payload
		if matchType == "Custom" {
			data = Payload{
				Region:       "7",
				Versus:       "players",
				MatchType:    matchType,
				SearchPlayer: username,
			}
		} else {
			data = Payload{
				Region:       "7",
				Versus:       "players",
				MatchType:    "unranked",
				TeamSize:     matchType,
				SearchPlayer: username,
			}
		}

		wg.Add(1)
		go func(mt string) {
			if err := queryToMap(data, mt, sm); err != nil {
				fmt.Printf("error retrieving Elo from AOE api for %s: %v", username, err)
			}
			wg.Done()
		}(matchType)
	}
	wg.Wait()

	return sm.respMap, nil
}

func queryToMap(data Payload, matchType string, sm *safeMap) error {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNoContent {
			return nil
		}
		return fmt.Errorf("error from API, received status code %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading API response: %w", err)
	}

	var respBodyJson response
	if err := json.Unmarshal(respBody, &respBodyJson); err != nil {
		return fmt.Errorf("error unmarshaling json API response: %w", err)
	}
	if respBodyJson.Count < 1 {
		return nil
	}

	sm.Lock()
	defer sm.Unlock()
	sm.respMap[matchType] = strconv.Itoa(respBodyJson.Items[0].Elo)

	return nil
}
