package discordapi

import (
	"fmt"
	"strings"

	"github.com/alexisgeoffrey/aoe4elobot/internal/config"
)

type userElo struct {
	oneVOne     int32
	twoVTwo     int32
	threeVThree int32
	fourVFour   int32
	custom      int32
}

func (elo *userElo) generateEloString(name string) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s:\n", name))

	eloVals := []int32{
		elo.oneVOne,
		elo.twoVTwo,
		elo.threeVThree,
		elo.fourVFour,
		elo.custom,
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
