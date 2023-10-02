package soccer_test

import (
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDetermineTeamChances_ChancesWithinMinMaxRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		chances, err := soccer.DetermineTeamChances(testdata.StrongTeamPlayers, testdata.StrongTeamPlayers)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(chances), soccer.MaxEvents)
		assert.GreaterOrEqual(t, len(chances), soccer.MinEvents)
	}
}
