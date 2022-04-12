package discordapi

import (
	"fmt"
	"strings"
)

type userElo struct {
	oneVOne     int32
	twoVTwo     int32
	threeVThree int32
	fourVFour   int32
	custom      int32
}

var eloLabels = []string{"1v1", "2v2", "3v3", "4v4", "Custom"}

func (elo *userElo) generateEloString(mention string) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s:\n", mention))

	eloVals := []int32{
		elo.oneVOne,
		elo.twoVTwo,
		elo.threeVThree,
		elo.fourVFour,
		elo.custom,
	}

	for i, et := range EloTypes {
		if et.Enabled && eloVals[i] != 0 {
			builder.WriteString(fmt.Sprintf("%s: %d\n", eloLabels[i], eloVals[i]))
		} else if et.Enabled {
			builder.WriteString(fmt.Sprintf("%s: None\n", eloLabels[i]))
		}
	}

	return builder.String()
}
