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

// SetPieceTaker: every *direct* set-piece chance (free kick / penalty) for a
// team must go to the named player when the field is populated. Corners are
// deliberately excluded — for a corner the named taker delivers the ball but
// doesn't head it home, so the finisher is picked normally.
func TestTactics_SetPieceTakerReceivesDirectSetPieces(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)
	home.Team.Tactics = soccer.Tactics{SetPieceTaker: "3"} // midfielder

	for seed := int64(0); seed < 20; seed++ {
		events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
		require.NoError(t, err)
		for _, e := range events {
			if e.ChanceType != soccer.ChanceTypeFreeKick &&
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
				"direct set-piece chance type %s went to %q, not the designated taker", e.ChanceType, attacker)
		}
	}
}

// SetPieceTaker on Corner: the named taker delivers, so they must never be
// the player credited with the corner shot. Heading + position weights pick
// the finisher.
func TestTactics_SetPieceTakerNeverFinishesCorners(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)
	home.Team.Tactics = soccer.Tactics{SetPieceTaker: "3"} // midfielder

	var cornerCount int
	for seed := int64(0); seed < 60; seed++ {
		events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
		require.NoError(t, err)
		for _, e := range events {
			if e.ChanceType != soccer.ChanceTypeCorner {
				continue
			}
			var attacker string
			var team soccer.TeamType
			switch e.Type {
			case soccer.GameEventTypeGoal:
				g := e.GetGoalEvent()
				attacker, team = g.PlayerID, g.TeamType
			case soccer.GameEventTypeMiss:
				m := e.GetMissEvent()
				attacker, team = m.PlayerID, m.TeamType
			}
			if team != soccer.TeamTypeHome {
				continue
			}
			cornerCount++
			assert.NotEqual(t, "3", attacker,
				"corner went to the named taker %q — taker delivers, finisher is somebody else", attacker)
		}
	}
	require.Greater(t, cornerCount, 0, "no home corners were generated — test would silently pass")
}

// SetPieceTaker on Corner: changing only the taker's Technique must not
// change which player finishes the corner — Technique drives delivery
// quality (chance conversion) but NOT finisher selection. The exclusion
// pool is identical between the two arms (same taker ID), so any shift in
// finisher distribution would mean Technique was leaking into the picker.
func TestTactics_CornerFinisherDistributionIndependentOfTakerTechnique(t *testing.T) {
	if testing.Short() {
		t.Skip("distribution test runs many trials; skip under -short")
	}

	const trials = 4000
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	finisherShares := func(takerTechnique int) map[string]float64 {
		home := cloneLineup(testdata.StrongTeam(soccer.FormationTypeDiamond))
		home.Players[2].ID = "taker"
		home.Players[2].Attributes.Technique = takerTechnique
		home.Team.Tactics = soccer.Tactics{SetPieceTaker: "taker"}

		counts := map[string]int{}
		var total int
		for seed := int64(0); seed < int64(trials); seed++ {
			events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
			require.NoError(t, err)
			for _, e := range events {
				if e.ChanceType != soccer.ChanceTypeCorner {
					continue
				}
				var attacker string
				var team soccer.TeamType
				switch e.Type {
				case soccer.GameEventTypeGoal:
					g := e.GetGoalEvent()
					attacker, team = g.PlayerID, g.TeamType
				case soccer.GameEventTypeMiss:
					m := e.GetMissEvent()
					attacker, team = m.PlayerID, m.TeamType
				}
				if team != soccer.TeamTypeHome {
					continue
				}
				counts[attacker]++
				total++
			}
		}
		require.Greater(t, total, 200, "not enough home corners")
		out := map[string]float64{}
		for id, c := range counts {
			out[id] = float64(c) / float64(total)
		}
		return out
	}

	low := finisherShares(30)
	high := finisherShares(95)

	for id, lowShare := range low {
		highShare := high[id]
		t.Logf("finisher %s: low-tech=%.3f high-tech=%.3f", id, lowShare, highShare)
		// 4pp tolerance — any larger drift would imply Technique leaked
		// into the picker. Both arms share the same exclusion set, so the
		// only way these can diverge is sampling noise.
		assert.InDelta(t, lowShare, highShare, 0.04,
			"finisher %s share moved with taker Technique (low=%.3f → high=%.3f) — Technique should only affect conversion, not finisher pick",
			id, lowShare, highShare)
	}
}

// SetPieceTaker on Corner: a high-Technique taker must produce a higher
// corner conversion rate than a low-Technique taker, all else equal.
func TestTactics_HighTechniqueTakerImprovesCornerConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("conversion test runs many trials; skip under -short")
	}

	const trials = 4000
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	cornerConversion := func(takerTechnique int) float64 {
		home := cloneLineup(testdata.StrongTeam(soccer.FormationTypeDiamond))
		// Slot the named taker at index 2 (a midfielder in StrongTeam).
		home.Players[2].ID = "taker"
		home.Players[2].Attributes.Technique = takerTechnique
		home.Team.Tactics = soccer.Tactics{SetPieceTaker: "taker"}

		var corners, goals int
		for seed := int64(0); seed < int64(trials); seed++ {
			events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
			require.NoError(t, err)
			for _, e := range events {
				if e.ChanceType != soccer.ChanceTypeCorner {
					continue
				}
				var team soccer.TeamType
				switch e.Type {
				case soccer.GameEventTypeGoal:
					team = e.GetGoalEvent().TeamType
				case soccer.GameEventTypeMiss:
					team = e.GetMissEvent().TeamType
				}
				if team != soccer.TeamTypeHome {
					continue
				}
				corners++
				if e.Type == soccer.GameEventTypeGoal {
					goals++
				}
			}
		}
		require.Greater(t, corners, 100, "not enough home corners")
		return float64(goals) / float64(corners)
	}

	low := cornerConversion(30)
	high := cornerConversion(95)

	t.Logf("home corner conversion: technique=30 → %.3f | technique=95 → %.3f", low, high)
	assert.Greater(t, high, low,
		"high-Technique corner taker must lift conversion (low=%.3f, high=%.3f)", low, high)
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

// Roles: tagging your *weakest* midfielder as Playmaker must HURT win rate
// vs the same lineup with no Playmaker. This is the focal-point property —
// the role is a real choice with a real downside, not a free lift. Pairs
// with TestPlaymaker_FocalPointShiftsAroundPositionMean (unit test on the
// math) by validating the integration impact end-to-end.
func TestRole_WeakPlaymakerHurtsWinRate(t *testing.T) {
	if testing.Short() {
		t.Skip("role impact test runs many trials; skip under -short")
	}

	const trials = 1500
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	// Replace one midfielder with a deliberately weak controller, then tag
	// THAT player as Playmaker. The team mean should drag down toward them.
	weakened := cloneLineup(home)
	for i := range weakened.Players {
		if weakened.Players[i].SelectedPosition == soccer.PlayerPositionMidfield {
			weakened.Players[i].Attributes.ControlRating = 30
			weakened.Players[i].Attributes.WorkRate = 30
			weakened.Players[i].Role = soccer.PlayerRolePlaymaker
			break
		}
	}

	// Control arm: the same weakened lineup but no Playmaker tag.
	control := cloneLineup(home)
	for i := range control.Players {
		if control.Players[i].SelectedPosition == soccer.PlayerPositionMidfield {
			control.Players[i].Attributes.ControlRating = 30
			control.Players[i].Attributes.WorkRate = 30
			break
		}
	}

	winControl := homeWinRate(t, trials, control, away)
	winWeakPM := homeWinRate(t, trials, weakened, away)

	t.Logf("home win rate: weak-mid no role=%.1f%% | weak-mid Playmaker=%.1f%%",
		winControl*100, winWeakPM*100)
	assert.Less(t, winWeakPM, winControl,
		"tagging the weakest controller as Playmaker must hurt win rate, not help it")
}

// Roles: tagging your *weakest* mid-tier defender as Ball Winner must HURT
// win rate vs the same lineup with no Ball Winner tag. Mirror of the
// Playmaker test on the defensive side — the fix replaces a flat ×1.10
// per-player boost with a focal-point weighting that has a real downside
// for poor picks.
//
// We tag a midfielder rather than a centre-back: StrongTeam has 2 mids,
// which is the smallest case where focal-point weighting actually shifts
// the group mean. (A Ball Winner who's the only player in their position
// group is a no-op — there's no group mean to drag — which matches the
// real-football intuition that the role only matters when you have a
// midfield to organise.)
func TestRole_WeakBallWinnerHurtsWinRate(t *testing.T) {
	if testing.Short() {
		t.Skip("role impact test runs many trials; skip under -short")
	}

	const trials = 1500
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	weakened := cloneLineup(home)
	for i := range weakened.Players {
		if weakened.Players[i].SelectedPosition == soccer.PlayerPositionMidfield {
			weakened.Players[i].Attributes.DefenseRating = 30
			weakened.Players[i].Attributes.Tackling = 30
			weakened.Players[i].Attributes.SpeedRating = 30
			weakened.Players[i].Role = soccer.PlayerRoleBallWinner
			break
		}
	}

	control := cloneLineup(home)
	for i := range control.Players {
		if control.Players[i].SelectedPosition == soccer.PlayerPositionMidfield {
			control.Players[i].Attributes.DefenseRating = 30
			control.Players[i].Attributes.Tackling = 30
			control.Players[i].Attributes.SpeedRating = 30
			break
		}
	}

	winControl := homeWinRate(t, trials, control, away)
	winWeakBW := homeWinRate(t, trials, weakened, away)

	t.Logf("home win rate: weak-mid no role=%.1f%% | weak-mid Ball Winner=%.1f%%",
		winControl*100, winWeakBW*100)
	assert.Less(t, winWeakBW, winControl,
		"tagging the weakest player in their position group as Ball Winner must hurt win rate, not help it")
}

// Roles: a quality Captain provides a small team-wide boost AND lifts their
// own play. Verify a strong-leader captain shifts win rate up.
func TestRole_QualityCaptainShiftsWinRate(t *testing.T) {
	if testing.Short() {
		t.Skip("role impact test runs many trials; skip under -short")
	}

	const trials = 1500
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	withCaptain := cloneLineup(home)
	// Player "3" is a midfielder with ControlRating=85 — a real leader.
	// Captain quality ≈ 85, well above neutral (60).
	for i := range withCaptain.Players {
		if withCaptain.Players[i].ID == "3" {
			withCaptain.Players[i].Role = soccer.PlayerRoleCaptain
			break
		}
	}

	winBase := homeWinRate(t, trials, home, away)
	winCaptain := homeWinRate(t, trials, withCaptain, away)

	t.Logf("home win rate: base=%.1f%%, with quality captain=%.1f%%", winBase*100, winCaptain*100)
	assert.Greater(t, winCaptain, winBase, "a quality captain should shift win rate in their team's favour")
}

// Roles: a poor-quality Captain (low ControlRating + low Composure) must
// HURT win rate vs the same lineup with no captain. The captain's burden
// is real — this is the symmetric downside that replaced the old flat
// +3% identity-agnostic boost.
func TestRole_PoorCaptainHurtsWinRate(t *testing.T) {
	if testing.Short() {
		t.Skip("role impact test runs many trials; skip under -short")
	}

	const trials = 1500
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.StrongTeam(soccer.FormationTypeDiamond)

	withPoorCaptain := cloneLineup(home)
	// Player "5" (attacker) has ControlRating=85 in StrongTeam — that's
	// not a poor captain. Drag it down explicitly so this test isolates
	// the quality scaling. Composure unset → backfills to ControlRating.
	for i := range withPoorCaptain.Players {
		if withPoorCaptain.Players[i].ID == "5" {
			withPoorCaptain.Players[i].Attributes.ControlRating = 25
			withPoorCaptain.Players[i].Role = soccer.PlayerRoleCaptain
			break
		}
	}

	// Control arm: same control-25 attacker but no captain tag.
	control := cloneLineup(home)
	for i := range control.Players {
		if control.Players[i].ID == "5" {
			control.Players[i].Attributes.ControlRating = 25
			break
		}
	}

	winControl := homeWinRate(t, trials, control, away)
	winPoorCap := homeWinRate(t, trials, withPoorCaptain, away)

	t.Logf("home win rate: weak-mid no captain=%.1f%% | weak-mid as captain=%.1f%%",
		winControl*100, winPoorCap*100)
	assert.Less(t, winPoorCap, winControl,
		"tagging a poor-quality player as captain must hurt win rate, not help it")
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
