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
		AdminRolesMap map[string]bool `yaml:"-"`
		DbUrl         string          `yaml:"db_url" env:"DB_URL" env-required:"true"`
		BotToken      string          `yaml:"bot_token" env:"BOT_TOKEN" env-required:"true"`
		BotChannelId  string          `yaml:"bot_channel_id" env-required:"true"`
		AdminRoles    []string        `yaml:"admin_roles,flow"`
		EloTypes      []EloType       `yaml:"-"`
		OneVOne       EloType         `yaml:"1v1"`
		TwoVTwo       EloType         `yaml:"2v2"`
		ThreeVThree   EloType         `yaml:"3v3"`
		FourVFour     EloType         `yaml:"4v4"`
		Custom        EloType
	}

	EloType struct {
		RoleMap map[string]int16 `yaml:"-"`
		Roles   []EloRole        `yaml:"roles,omitempty"`
		Enabled bool
	}

	EloRole struct {
		RoleId       string `yaml:"role_id"`
		RolePriority int16  `yaml:"role_priority"`
		StartingElo  int16  `yaml:"starting_elo"`
		EndingElo    int16  `yaml:"ending_elo"`
	}
)

const UserAgent = "AOE 4 Elo Bot/2.0.0 (github.com/alexisgeoffrey/aoe4elobot; alexisgeoffrey1@gmail.com)"

var Cfg ConfigFile

var (
	sampleEloRoles = []EloRole{
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

	err := cleanenv.ReadConfig(configPath, &Cfg)
	if errors.Is(err, os.ErrNotExist) {
		if err := genConfig(configPath); err != nil {
			log.Printf("error generating config file: %v", err)
		}
		os.Exit(1)
	} else if err != nil {
		log.Fatalf("error reading config file: %v\n", err)
	}

	for _, eloType := range []*EloType{
		&Cfg.OneVOne,
		&Cfg.TwoVTwo,
		&Cfg.ThreeVThree,
		&Cfg.FourVFour,
		&Cfg.Custom,
	} {
		if eloType.Enabled && len(eloType.Roles) != 0 {
			eloType.RoleMap = make(map[string]int16, len(eloType.Roles))
			for _, role := range eloType.Roles {
				eloType.RoleMap[role.RoleId] = role.RolePriority
			}
		}
	}

	Cfg.EloTypes = []EloType{
		Cfg.OneVOne,
		Cfg.TwoVTwo,
		Cfg.ThreeVThree,
		Cfg.FourVFour,
		Cfg.Custom,
	}

	Cfg.AdminRolesMap = make(map[string]bool, len(Cfg.AdminRoles))
	for _, role := range Cfg.AdminRoles {
		Cfg.AdminRolesMap[role] = true
	}
}

func genConfig(path string) error {
	log.Println("Config file does not exist. Creating...")

	Cfg.OneVOne = EloType{Enabled: true, Roles: sampleEloRoles}
	Cfg.AdminRoles = sampleAdminRoles
	Cfg.BotChannelId = "botChannelId"

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating config file: %w", err)
	}
	defer file.Close()

	yamlBytes, err := yaml.Marshal(Cfg)
	if err != nil {
		return fmt.Errorf("error marshaling yaml struct: %w", err)
	}

	if _, err := file.Write(yamlBytes); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
