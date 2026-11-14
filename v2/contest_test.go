package soccer_test

import (
	"errors"
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runContest(t *testing.T, seed int64, home, away soccer.GameLineup) soccer.MatchOutcome {
	t.Helper()
	out, err := soccer.RunGameWithModel(rand.New(rand.NewSource(seed)), home, away, soccer.ModelContest)
	require.NoError(t, err)
	return out
}

func goals(events []soccer.GameEvent) (home, away int) {
	stats := soccer.CreateGameStats(events)
	return stats.HomeTeamStats.Goals, stats.AwayTeamStats.Goals
}

func TestRunGameWithModel_ClassicIsUnchanged(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.WeakTeam(soccer.FormationTypeBox)
	for seed := int64(0); seed < 50; seed++ {
		want, err := soccer.RunGameWithSeedDetailed(rand.New(rand.NewSource(seed)), home, away)
		require.NoError(t, err)
		got, err := soccer.RunGameWithModel(rand.New(rand.NewSource(seed)), home, away, soccer.ModelClassic)
		require.NoError(t, err)
		assert.Equal(t, want, got, "seed %d", seed)
	}
}

func TestRunGameWithModel_RejectsBadInput(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	_, err := soccer.RunGameWithModel(nil, home, home, soccer.ModelContest)
	assert.ErrorIs(t, err, soccer.ErrNilRandSource)
	_, err = soccer.RunGameWithModel(rand.New(rand.NewSource(1)), home, home, soccer.Model("nope"))
	assert.True(t, errors.Is(err, soccer.ErrUnknownModel))
}

func TestContest_IsDeterministic(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeY)
	away := testdata.WeakTeam(soccer.FormationTypePyramid)
	for seed := int64(0); seed < 20; seed++ {
		assert.Equal(t, runContest(t, seed, home, away), runContest(t, seed, home, away))
	}
}

func TestContest_EmitsShotsInOrder(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeBox)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)
	for seed := int64(0); seed < 200; seed++ {
		out := runContest(t, seed, home, away)
		prev := 0
		for _, e := range out.Events {
			assert.GreaterOrEqual(t, e.Minute, prev, "seed %d: events out of order", seed)
			prev = e.Minute
			assert.NotEmpty(t, e.ChanceType)
			switch e.Event.(type) {
			case soccer.GoalEvent:
				assert.Equal(t, soccer.GameEventTypeGoal, e.Type)
			case soccer.MissEvent:
				assert.Equal(t, soccer.GameEventTypeMiss, e.Type)
			default:
				t.Fatalf("seed %d: unexpected event %T", seed, e.Event)
			}
		}
	}
}

func TestContest_ScoringLooksLikeFootball(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)
	const n = 3000
	var total, draws, nilNil, shots int
	for seed := int64(0); seed < n; seed++ {
		out := runContest(t, seed, home, away)
		h, a := goals(out.Events)
		total += h + a
		shots += len(out.Events)
		if h == a {
			draws++
		}
		if h+a == 0 {
			nilNil++
		}
	}
	perMatch := float64(total) / n
	assert.InDelta(t, 3.0, perMatch, 1.0, "goals per match")
	assert.InDelta(t, 12.5, float64(shots)/n, 3.0, "shots per match")
	assert.Greater(t, float64(draws)/n, 0.15, "draws")
	assert.Greater(t, float64(nilNil)/n, 0.02, "0-0s should happen")
}

func TestContest_StrongerTeamWinsMoreOften(t *testing.T) {
	strong := testdata.StrongTeam(soccer.FormationTypeDiamond)
	weak := testdata.WeakTeam(soccer.FormationTypeDiamond)
	var strongWins, weakWins int
	for seed := int64(0); seed < 2000; seed++ {
		// Weak side at home, so home advantage works against the result.
		h, a := goals(runContest(t, seed, weak, strong).Events)
		switch {
		case a > h:
			strongWins++
		case h > a:
			weakWins++
		}
	}
	assert.Greater(t, strongWins, 2*weakWins, "strong %d vs weak %d", strongWins, weakWins)
}

func TestContest_HomeAdvantage(t *testing.T) {
	team := testdata.StrongTeam(soccer.FormationTypeDiamond)
	const n = 4000
	var homeWins, awayWins int
	for seed := int64(0); seed < n; seed++ {
		h, a := goals(runContest(t, seed, team, team).Events)
		switch {
		case h > a:
			homeWins++
		case a > h:
			awayWins++
		}
	}
	assert.Greater(t, float64(homeWins-awayWins)/n, 0.05,
		"home %d vs away %d: identical sides should favour the home team", homeWins, awayWins)
}

// Set pieces come from fouls, and fouls from aggressive defending: a side
// that tackles recklessly gives away more free kicks and penalties.
func TestContest_AggressiveDefendingConcedesMoreSetPieces(t *testing.T) {
	attackers := testdata.StrongTeam(soccer.FormationTypeDiamond)
	setPiecesAgainst := func(aggression int) int {
		defenders := withAggression(testdata.StrongTeam(soccer.FormationTypeDiamond), aggression)
		n := 0
		for seed := int64(0); seed < 3000; seed++ {
			for _, e := range runContest(t, seed, attackers, defenders).Events {
				isHome := false
				switch ev := e.Event.(type) {
				case soccer.GoalEvent:
					isHome = ev.TeamType == soccer.TeamTypeHome
				case soccer.MissEvent:
					isHome = ev.TeamType == soccer.TeamTypeHome
				}
				if isHome && (e.ChanceType == soccer.ChanceTypeFreeKick || e.ChanceType == soccer.ChanceTypePenalty) {
					n++
				}
			}
		}
		return n
	}
	calm, reckless := setPiecesAgainst(30), setPiecesAgainst(95)
	assert.Greater(t, reckless, calm*3/2, "calm %d, reckless %d", calm, reckless)
}

func withAggression(l soccer.GameLineup, aggression int) soccer.GameLineup {
	players := make([]soccer.SelectedPlayer, len(l.Players))
	copy(players, l.Players)
	for i := range players {
		players[i].Attributes.AggressionRating = aggression
	}
	l.Players = players
	return l
}

// The contest model's own balance harness: every formation pair, identical
// players, home win rates within 3% of each other (strict under
// RUN_BALANCE_STRICT=1, like the classic harness).
func TestContest_FormationBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("balance harness is heavy; rerun without -short")
	}
	formations := []soccer.FormationType{
		soccer.FormationTypePyramid,
		soccer.FormationTypeDiamond,
		soccer.FormationTypeY,
		soccer.FormationTypeBox,
	}
	const trials = 5000
	maxHome, minHome := 0.0, 1.0
	for _, h := range formations {
		for _, a := range formations {
			home, away := testdata.StrongTeam(h), testdata.StrongTeam(a)
			wins := 0
			for seed := int64(0); seed < trials; seed++ {
				hg, ag := goals(runContest(t, seed, home, away).Events)
				if hg > ag {
					wins++
				}
			}
			pct := float64(wins) / trials
			t.Logf("  %-12s vs %-12s home win %5.1f%%", h, a, pct*100)
			if pct > maxHome {
				maxHome = pct
			}
			if pct < minHome {
				minHome = pct
			}
		}
	}
	spread := maxHome - minHome
	t.Logf("contest model home-win-rate spread: %.2f%% (target ≤3%%)", spread*100)
	if shouldRunStrict() {
		assert.LessOrEqual(t, spread, 0.03, "formations not balanced under the contest model")
	}
}
