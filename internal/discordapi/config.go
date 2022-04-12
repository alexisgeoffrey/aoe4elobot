package discordapi

type (
	ConfigFile struct {
		BotToken    string  `yaml:"bot_token" env:"BOT_TOKEN" env-required:"true"`
		DbUrl       string  `yaml:"db_url" env:"DB_URL" env-required:"true"`
		OneVOne     EloType `yaml:"1v1"`
		TwoVTwo     EloType `yaml:"2v2"`
		ThreeVThree EloType `yaml:"3v3"`
		FourVFour   EloType `yaml:"4v4"`
		Custom      EloType
		AdminRoles  []string `yaml:"admin_roles,flow"`
	}

	EloType struct {
		Enabled bool
		Roles   []*EloRole `yaml:"roles,omitempty"`
		roleMap map[string]bool
	}

	EloRole struct {
		RoleId      string `yaml:"role_id"`
		StartingElo int32  `yaml:"starting_elo"`
		EndingElo   int32  `yaml:"ending_elo"`
	}
)

const configPath = "config/config.json"

var Config ConfigFile
