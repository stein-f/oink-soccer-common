package soccer_test

import (
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDetermineTeamChances_ChancesWithinMinMaxRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		homeLineup := soccer.GameLineup{Players: testdata.StrongTeamPlayers}
		awayLineup := soccer.GameLineup{Players: testdata.StrongTeamPlayers}
		chances, err := soccer.DetermineTeamChances(homeLineup, awayLineup)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(chances), 12)
		assert.GreaterOrEqual(t, len(chances), 1)
	}
}

func TestGetRandomMinutes(t *testing.T) {
	for i := 0; i < 100; i++ {
		minutes, err := soccer.GetRandomMinutes(10)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range minutes {
			assert.GreaterOrEqual(t, m, 1)
			assert.LessOrEqual(t, m, 98)
		}
		assert.Len(t, minutes, 10)
	}
}
