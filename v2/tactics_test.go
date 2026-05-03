package soccer_test

import (
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Backward-compat: a lineup with no Tactics field set must produce the same
// match as one where Tactics is the explicit zero value. This is the
// guarantee that v1-style consumers don't need to care about the new field.
func TestTactics_ZeroValueIsNeutral(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	withExplicitNeutral := home
	withExplicitNeutral.Team.Tactics = soccer.Tactics{} // explicit zero

	const seed = int64(99)
	a, _, _ := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
	b, _, _ := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), withExplicitNeutral, away)

	assert.Equal(t, a, b, "explicit zero-value Tactics must behave identically to unset Tactics")
}

// Each tactic must have a measurable effect over many trials. Direction
// matters: pressing should reduce the opponent's win rate; high tempo
// should produce more chances; etc.
func TestTactics_PressReducesOpponentScoring(t *testing.T) {
	if testing.Short() {
		t.Skip("tactics impact test runs many trials; skip under -short")
	}

	const trials = 1000
	homeBase := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	homePressing := homeBase
	homePressing.Team.Tactics = soccer.Tactics{Press: soccer.PressLevelHigh}

	awayGoalsBase := awayGoalAvg(t, trials, homeBase, away)
	awayGoalsPressed := awayGoalAvg(t, trials, homePressing, away)

	t.Logf("away goals/game: base=%.2f, vs high press=%.2f", awayGoalsBase, awayGoalsPressed)
	assert.Less(t, awayGoalsPressed, awayGoalsBase,
		"pressing high should reduce opponent goals; base=%.2f pressed=%.2f", awayGoalsBase, awayGoalsPressed)
}

func TestTactics_FastTempoProducesMoreChances(t *testing.T) {
	if testing.Short() {
		t.Skip("tactics impact test runs many trials; skip under -short")
	}

	const trials = 1000
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	homeFast := home
	homeFast.Team.Tactics = soccer.Tactics{Tempo: soccer.TempoLevelFast}

	chancesBase := totalChancesAvg(t, trials, home, away)
	chancesFast := totalChancesAvg(t, trials, homeFast, away)

	t.Logf("total chances/game: base=%.2f, with fast tempo=%.2f", chancesBase, chancesFast)
	assert.Greater(t, chancesFast, chancesBase,
		"fast tempo should produce more chances; base=%.2f fast=%.2f", chancesBase, chancesFast)
}

// SetPieceTaker: every set-piece chance for a team must go to the named
// player when the field is populated.
func TestTactics_SetPieceTakerReceivesAllSetPieces(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)
	home.Team.Tactics = soccer.Tactics{SetPieceTaker: "3"} // midfielder

	for seed := int64(0); seed < 20; seed++ {
		events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
		require.NoError(t, err)
		for _, e := range events {
			if e.ChanceType != soccer.ChanceTypeFreeKick &&
				e.ChanceType != soccer.ChanceTypeCorner &&
				e.ChanceType != soccer.ChanceTypePenalty {
				continue
			}
			var attacker string
			switch e.Type {
			case soccer.GameEventTypeGoal:
				g := e.GetGoalEvent()
				if g.TeamType != soccer.TeamTypeHome {
					continue
				}
				attacker = g.PlayerID
			case soccer.GameEventTypeMiss:
				m := e.GetMissEvent()
				if m.TeamType != soccer.TeamTypeHome {
					continue
				}
				attacker = m.PlayerID
			}
			assert.Equal(t, "3", attacker,
				"set-piece chance type %s went to %q, not the designated taker", e.ChanceType, attacker)
		}
	}
}

// Press fatigue: high-pressing teams must score *fewer* goals in the late
// game (60+min) than they would on neutral press. Without this in-match
// cost, high press would be a free lunch — managers would always pick it
// since the only other downside is a next-game injury risk that recovery
// items can mitigate.
func TestTactics_PressFatigueReducesLateGoals(t *testing.T) {
	if testing.Short() {
		t.Skip("fatigue impact test runs many trials; skip under -short")
	}

	const trials = 2000
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	homeNeutral := home
	homeHighPress := home
	homeHighPress.Team.Tactics = soccer.Tactics{Press: soccer.PressLevelHigh}

	earlyN, lateN := homeGoalsByPhase(t, trials, homeNeutral, away)
	earlyP, lateP := homeGoalsByPhase(t, trials, homeHighPress, away)

	t.Logf("home goals/game: neutral early=%.2f late=%.2f | high-press early=%.2f late=%.2f",
		earlyN, lateN, earlyP, lateP)

	// Early-game goals should be similar (fatigue hasn't kicked in yet).
	// Late-game goals must drop measurably under high press.
	assert.Less(t, lateP, lateN,
		"high press should fatigue: late goals must drop (%.3f → %.3f)", lateN, lateP)

	// Sanity: early goals don't drop disproportionately. Allow some variance
	// from the press's other effects (control reduction shifts possession),
	// but the *late*-game gap must be wider than the early-game gap.
	earlyGap := earlyN - earlyP
	lateGap := lateN - lateP
	assert.Greater(t, lateGap, earlyGap,
		"the high-press scoring penalty must be larger late than early — got early gap=%.3f, late gap=%.3f",
		earlyGap, lateGap)
}

// Roles: a Captain provides a small team-wide boost. Verify it shifts win
// rate in the captain's favour over enough trials.
func TestRole_CaptainShiftsWinRate(t *testing.T) {
	if testing.Short() {
		t.Skip("role impact test runs many trials; skip under -short")
	}

	const trials = 1500
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	withCaptain := cloneLineup(home)
	// Make the GK the captain (any player works — the boost is team-wide).
	withCaptain.Players[0].Role = soccer.PlayerRoleCaptain

	winBase := homeWinRate(t, trials, home, away)
	winCaptain := homeWinRate(t, trials, withCaptain, away)

	t.Logf("home win rate: base=%.1f%%, with captain=%.1f%%", winBase*100, winCaptain*100)
	assert.Greater(t, winCaptain, winBase, "captain should shift win rate in their team's favour")
}

// homeGoalsByPhase returns (avg early-game goals, avg late-game goals) per
// match for the home team. "Late" is minute >= 60, where pressFatigueFactor
// kicks in.
func homeGoalsByPhase(t *testing.T, trials int, home, away soccer.GameLineup) (early, late float64) {
	t.Helper()
	var earlyTotal, lateTotal int
	for i := 0; i < trials; i++ {
		events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(int64(i))), home, away)
		require.NoError(t, err)
		for _, e := range events {
			if e.Type != soccer.GameEventTypeGoal {
				continue
			}
			g := e.GetGoalEvent()
			if g.TeamType != soccer.TeamTypeHome {
				continue
			}
			if e.Minute < 60 {
				earlyTotal++
			} else {
				lateTotal++
			}
		}
	}
	n := float64(trials)
	return float64(earlyTotal) / n, float64(lateTotal) / n
}

func awayGoalAvg(t *testing.T, trials int, home, away soccer.GameLineup) float64 {
	t.Helper()
	var total int
	for i := 0; i < trials; i++ {
		events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(int64(i))), home, away)
		require.NoError(t, err)
		stats := soccer.CreateGameStats(events)
		total += stats.AwayTeamStats.Goals
	}
	return float64(total) / float64(trials)
}

func totalChancesAvg(t *testing.T, trials int, home, away soccer.GameLineup) float64 {
	t.Helper()
	var total int
	for i := 0; i < trials; i++ {
		events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(int64(i))), home, away)
		require.NoError(t, err)
		total += len(events)
	}
	return float64(total) / float64(trials)
}
