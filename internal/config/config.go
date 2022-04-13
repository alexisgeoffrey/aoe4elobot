package config

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

type (
	ConfigFile struct {
		BotToken      string   `yaml:"bot_token" env:"BOT_TOKEN" env-required:"true"`
		DbUrl         string   `yaml:"db_url" env:"DB_URL" env-required:"true"`
		OneVOne       *EloType `yaml:"1v1"`
		TwoVTwo       *EloType `yaml:"2v2"`
		ThreeVThree   *EloType `yaml:"3v3"`
		FourVFour     *EloType `yaml:"4v4"`
		Custom        *EloType
		AdminRoles    []string        `yaml:"admin_roles,flow"`
		AdminRolesMap map[string]bool `yaml:"admin_roles_map,omitempty"`
		BotChannelId  string          `yaml:"bot_channel_id" env-required:"true"`
	}

	EloType struct {
		Enabled bool
		Roles   []*EloRole       `yaml:"roles,omitempty"`
		RoleMap map[string]int32 `yaml:"role_map,omitempty"`
	}

	EloRole struct {
		RoleId       string `yaml:"role_id"`
		StartingElo  int32  `yaml:"starting_elo"`
		EndingElo    int32  `yaml:"ending_elo"`
		RolePriority int32  `yaml:"role_priority"`
	}
)

var (
	Config ConfigFile

	sampleEloRoles = []*EloRole{
		{
			RoleId:       "eloRoleId1",
			StartingElo:  500,
			EndingElo:    1000,
			RolePriority: 200,
		},
		{
			RoleId:       "eloRoleId2",
			StartingElo:  1001,
			EndingElo:    2000,
			RolePriority: 100,
		},
	}

	sampleAdminRoles = []string{"adminRoleId1", "adminRoleId2"}
)

func init() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yml"
	}

	err := cleanenv.ReadConfig(configPath, &Config)
	if errors.Is(err, os.ErrNotExist) {
		genConfig(configPath)
		return
	} else if err != nil {
		log.Fatalf("error reading config file: %v\n", err)
	}

	for _, roleSet := range GetEloTypes() {
		if roleSet.Enabled {
			roleSet.RoleMap = make(map[string]int32, len(roleSet.Roles))
			for _, role := range roleSet.Roles {
				roleSet.RoleMap[role.RoleId] = role.RolePriority
			}
		}
	}

	Config.AdminRolesMap = make(map[string]bool, len(Config.AdminRoles))
	for _, role := range Config.AdminRoles {
		Config.AdminRolesMap[role] = true
	}
}

func genConfig(path string) error {
	Config.OneVOne = &EloType{Enabled: true, Roles: sampleEloRoles}
	Config.TwoVTwo = &EloType{}
	Config.ThreeVThree = &EloType{}
	Config.FourVFour = &EloType{}
	Config.Custom = &EloType{}
	Config.AdminRoles = sampleAdminRoles
	Config.BotChannelId = "botChannelId"

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating config file: %v", err)
	}
	defer file.Close()

	yamlBytes, err := yaml.Marshal(&Config)
	if err != nil {
		return fmt.Errorf("error marshaling yaml struct: %v", err)
	}

	if _, err := file.Write(yamlBytes); err != nil {
		return fmt.Errorf("error writing config file: %v", err)
	}

	log.Println("Config file does not exist. Creating...")
	return nil
}

func GetEloTypes() []*EloType {
	return []*EloType{
		Config.OneVOne,
		Config.TwoVTwo,
		Config.ThreeVThree,
		Config.FourVFour,
		Config.Custom,
	}
}
