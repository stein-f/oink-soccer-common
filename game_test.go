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

func TestDetermineTeamChances_RespectsFormationTruthTableRanges(t *testing.T) {
	// mapping of directional HOME/AWAY style pairs to expected inclusive ranges
	type rng struct{ min, max int }

	// style resolution here mirrors formationStyle in the production code:
	toStyle := func(f soccer.FormationType) string {
		switch f {
		case soccer.FormationTypePyramid:
			return "DEF"
		case soccer.FormationTypeDiamond, soccer.FormationTypeBox:
			return "BAL"
		case soccer.FormationTypeY:
			return "ATT"
		default:
			return "BAL"
		}
	}

	styleKey := func(homeStyle, awayStyle string) string {
		return "HOME:" + homeStyle + "|AWAY:" + awayStyle
	}

	ranges := map[string]rng{
		"HOME:ATT|AWAY:ATT": {min: 7, max: 15},
		"HOME:ATT|AWAY:BAL": {min: 6, max: 12},
		"HOME:ATT|AWAY:DEF": {min: 5, max: 11},

		"HOME:BAL|AWAY:ATT": {min: 7, max: 12},
		"HOME:BAL|AWAY:BAL": {min: 4, max: 9},
		"HOME:BAL|AWAY:DEF": {min: 3, max: 8},

		"HOME:DEF|AWAY:ATT": {min: 6, max: 11},
		"HOME:DEF|AWAY:BAL": {min: 3, max: 8},
		"HOME:DEF|AWAY:DEF": {min: 1, max: 6},
	}

	formations := []soccer.FormationType{
		soccer.FormationTypePyramid, // DEF
		soccer.FormationTypeDiamond, // BAL
		soccer.FormationTypeBox,     // ATT
		soccer.FormationTypeY,       // ATT
	}

	// run several trials to cover randomness
	trials := 100

	src := testdata.TimeNowRandSource()

	for _, homeF := range formations {
		for _, awayF := range formations {
			t.Run(string(homeF)+" vs "+string(awayF), func(t *testing.T) {
				home := soccer.GameLineup{Team: soccer.Team{Formation: homeF}, Players: testdata.StrongTeam()}
				away := soccer.GameLineup{Team: soccer.Team{Formation: awayF}, Players: testdata.StrongTeam()}

				key := styleKey(toStyle(homeF), toStyle(awayF))
				expected, ok := ranges[key]
				if !ok {
					t.Fatalf("missing expected range for key %s", key)
				}

				for i := 0; i < trials; i++ {
					chances, err := soccer.DetermineTeamChances(src, home, away)
					assert.NoError(t, err)
					if !assert.GreaterOrEqual(t, len(chances), expected.min) {
						t.Logf("got %d chances; expected >= %d for %s", len(chances), expected.min, key)
					}
					if !assert.LessOrEqual(t, len(chances), expected.max) {
						t.Logf("got %d chances; expected <= %d for %s", len(chances), expected.max, key)
					}
				}
			})
		}
	}
}
