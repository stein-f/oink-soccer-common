package soccer_test

import (
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
)

func TestDetermineTeamChances_ChancesWithinMinMaxRange(t *testing.T) {
	source := testdata.TimeNowRandSource()
	for i := 0; i < 100; i++ {
		homeLineup := soccer.GameLineup{Players: testdata.StrongTeamPlayers}
		awayLineup := soccer.GameLineup{Players: testdata.StrongTeamPlayers}
		chances, err := soccer.DetermineTeamChances(source, homeLineup, awayLineup)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(chances), 12)
		assert.GreaterOrEqual(t, len(chances), 1)
	}
}

func TestGetRandomMinutes(t *testing.T) {
	source := testdata.TimeNowRandSource()
	for i := 0; i < 100; i++ {
		minutes, err := soccer.GetRandomMinutes(source, 10)
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
