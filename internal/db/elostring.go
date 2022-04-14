package db

import (
	"fmt"
	"strings"

	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
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

	eloLabels := getEloLabels()
	for i, et := range config.GetEloTypes() {
		if et.Enabled && eloVals[i] != 0 {
			builder.WriteString(fmt.Sprintf("%s: %d\n", eloLabels[i], eloVals[i]))
		} else if et.Enabled {
			builder.WriteString(fmt.Sprintf("%s: None\n", eloLabels[i]))
		}
	}

	return builder.String()
}

func getEloLabels() []string {
	return []string{"1v1", "2v2", "3v3", "4v4", "Custom"}
}
