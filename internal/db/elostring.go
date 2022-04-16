package db

import (
	"fmt"
	"strings"

	"github.com/alexisgeoffrey/aoe4elobot/v2/internal/config"
)

func (elo *UserElo) GenerateEloString(name string) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s:\n", name))

	eloVals := []int32{
		elo.OneVOne,
		elo.TwoVTwo,
		elo.ThreeVThree,
		elo.FourVFour,
		elo.Custom,
	}

	for i, label := range [...]string{"1v1", "2v2", "3v3", "4v4", "Custom"} {
		if config.Cfg.EloTypes[i].Enabled {
			if eloVals[i] == 0 {
				builder.WriteString(fmt.Sprintf("%s: None\n", label))
			} else {
				builder.WriteString(fmt.Sprintf("%s: %d\n", label, eloVals[i]))
			}
		}
	}

	return builder.String()
}
