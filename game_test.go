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
		homeLineup := soccer.GameLineup{Players: testdata.StrongTeam()}
		awayLineup := soccer.GameLineup{Players: testdata.StrongTeam()}
		chances, err := soccer.DetermineTeamChances(source, homeLineup, awayLineup)
		assert.NoError(t, err)
		assert.LessOrEqual(t, len(chances), 12)
		assert.GreaterOrEqual(t, len(chances), 1)
	}
}

func TestDetermineTeamChances_TeamWithInjuriesWinsLessFrequently(t *testing.T) {
	source := testdata.TimeNowRandSource()
	players := testdata.StrongTeam()

	injuredTeam := soccer.GameLineup{Players: players}
	injuredTeam.Players[4].Injury = &soccer.InjuryEvent{Injury: soccer.Injury{StatsReduction: 0.85}}

	healthyTeam := soccer.GameLineup{Players: testdata.StrongTeam()}

	var injuredTeamWins, healthyTeamWins int
	for i := 0; i < 1000; i++ {
		events, _, err := soccer.RunGameWithSeed(source, injuredTeam, healthyTeam)
		stats := soccer.CreateGameStats(events)
		if stats.HomeTeamStats.Goals > stats.AwayTeamStats.Goals {
			injuredTeamWins++
		} else if stats.HomeTeamStats.Goals < stats.AwayTeamStats.Goals {
			healthyTeamWins++
		}
		assert.NoError(t, err)
	}

	assert.Greater(t, healthyTeamWins, injuredTeamWins)
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

func TestGetTeamBoost_CanStackMultipleBoosts(t *testing.T) {
	lineup := soccer.GameLineup{Players: testdata.StrongTeam()}
	lineup.ItemBoosts = []soccer.Boost{
		{BoostType: soccer.BoostTypeTeam, MinBoost: 2, MaxBoost: 2},
		{BoostType: soccer.BoostTypeTeam, MinBoost: 2, MaxBoost: 2},
	}

	teamBoost := soccer.GetTeamBoost(testdata.TimeNowRandSource(), lineup)

	assert.Equal(t, 4.0, teamBoost)
}

func TestGetTeamBoost_HandlesNoBoosts(t *testing.T) {
	lineup := soccer.GameLineup{Players: testdata.StrongTeam()}

	teamBoost := soccer.GetTeamBoost(testdata.TimeNowRandSource(), lineup)

	assert.Equal(t, 1.0, teamBoost)
}
