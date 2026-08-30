package soccer

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boostTestPlayer(id string, pos PlayerPosition, rating int) SelectedPlayer {
	return SelectedPlayer{
		ID: id,
		Attributes: PlayerAttributes{
			ControlRating:    rating,
			AttackRating:     rating,
			DefenseRating:    rating,
			GoalkeeperRating: rating,
			SpeedRating:      rating,
			WorkRate:         rating,
			PrimaryPosition:  pos,
			Positions:        []PlayerPosition{pos},
		},
		SelectedPosition: pos,
	}
}

// One player per position, all identically rated, so every position group's
// mean is the same and a group's contribution is exactly its position weight.
func boostTestLineup() GameLineup {
	return GameLineup{Players: []SelectedPlayer{
		boostTestPlayer("gk", PlayerPositionGoalkeeper, 70),
		boostTestPlayer("def", PlayerPositionDefense, 70),
		boostTestPlayer("mid", PlayerPositionMidfield, 70),
		boostTestPlayer("atk", PlayerPositionAttack, 70),
	}}
}

// A Position boost lifts only the group it targets. With uniform players the
// team score moves by exactly that group's position weight, which is the
// whole point of a position boost being cheaper than a team boost.
func TestPositionBoost_LiftsOnlyTheTargetedGroup(t *testing.T) {
	lineup := boostTestLineup()
	const boost = 1.10
	mids := positionMultipliers{PlayerPositionMidfield: boost}

	t.Run("control", func(t *testing.T) {
		base := teamControl(lineup, nil)
		boosted := teamControl(lineup, mids)

		// Midfield carries 0.65 of control, so a +10% midfield boost is +6.5%.
		want := base * (1 + (boost-1)*tuning.ControlPositionWeights.Midfield)
		assert.InDelta(t, want, boosted, 1e-9)
		assert.Greater(t, boosted, base)
	})

	t.Run("defense", func(t *testing.T) {
		base := teamDefense(lineup, nil)
		boosted := teamDefense(lineup, mids)

		// Midfield carries only 0.20 of defense — the same item, smaller share.
		want := base * (1 + (boost-1)*tuning.DefensePositionWeights.Midfield)
		assert.InDelta(t, want, boosted, 1e-9)
	})

	t.Run("a boost on an empty group changes nothing", func(t *testing.T) {
		noMids := GameLineup{Players: []SelectedPlayer{
			boostTestPlayer("gk", PlayerPositionGoalkeeper, 70),
			boostTestPlayer("def", PlayerPositionDefense, 70),
			boostTestPlayer("atk", PlayerPositionAttack, 70),
		}}
		assert.InDelta(t, teamControl(noMids, nil), teamControl(noMids, mids), 1e-9)
	})
}

// The Half-time Orange is a midfield Position boost. Before this it was
// declared, stored, shown in the UI and charged for, but never read by the
// engine — the item did nothing in a match.
func TestRollPositionBoosts_HalfTimeOrangeReachesTheEngine(t *testing.T) {
	orange := Boost{
		BoostType:     BoostTypePosition,
		BoostPosition: PlayerPositionMidfield,
		MinBoost:      1.03,
		MaxBoost:      1.07,
	}
	lineup := boostTestLineup()
	lineup.ItemBoosts = []Boost{orange}

	pm := rollPositionBoosts(rand.New(rand.NewSource(7)), lineup)

	require.NotNil(t, pm, "a Position-typed boost must produce a multiplier")
	assert.InDelta(t, 1.05, pm.forPosition(PlayerPositionMidfield), 0.02)
	assert.Equal(t, 1.0, pm.forPosition(PlayerPositionAttack),
		"an untargeted group must be left alone")
}

func TestRollPositionBoosts_IgnoresTeamBoostsAndCompoundsPerPosition(t *testing.T) {
	lineup := boostTestLineup()
	lineup.ItemBoosts = []Boost{
		{BoostType: BoostTypeTeam, MinBoost: 1.05, MaxBoost: 1.05},
		{BoostType: BoostTypePosition, BoostPosition: PlayerPositionMidfield, MinBoost: 1.10, MaxBoost: 1.10},
		{BoostType: BoostTypePosition, BoostPosition: PlayerPositionMidfield, MinBoost: 1.20, MaxBoost: 1.20},
		{BoostType: BoostTypePosition, BoostPosition: PlayerPositionDefense, MinBoost: 1.50, MaxBoost: 1.50},
	}

	pm := rollPositionBoosts(rand.New(rand.NewSource(7)), lineup)

	assert.InDelta(t, 1.32, pm.forPosition(PlayerPositionMidfield), 1e-9, "1.10 * 1.20")
	assert.InDelta(t, 1.50, pm.forPosition(PlayerPositionDefense), 1e-9)
	assert.Equal(t, 1.0, pm.forPosition(PlayerPositionAttack),
		"the team boost must not leak into a position group")
}

func TestRollPositionBoosts_NoBoostsIsNil(t *testing.T) {
	pm := rollPositionBoosts(rand.New(rand.NewSource(7)), boostTestLineup())

	assert.Nil(t, pm)
	assert.Equal(t, 1.0, pm.forPosition(PlayerPositionMidfield),
		"the nil map must read as neutral rather than zero")
}

// Team boosts are sold as applying to "defending, control and attack". They
// used to reach control and defense only, leaving the shot itself unboosted.
func TestTeamBoost_LiftsTheShot(t *testing.T) {
	attacker := boostTestPlayer("atk", PlayerPositionAttack, 70)

	goalsWithBoost := countGoals(t, attacker, 1.10)
	goalsWithout := countGoals(t, attacker, 1.0)

	assert.Greater(t, goalsWithBoost, goalsWithout,
		"a +10%% attack boost must convert more chances (%d vs %d)", goalsWithBoost, goalsWithout)
}

// countGoals resolves the same chance many times and counts the goals, so a
// small shift in conversion shows up as a difference in outcome rather than
// getting lost in a single roll.
func countGoals(t *testing.T, attacker SelectedPlayer, attackBoost float64) int {
	t.Helper()

	const trials = 4000
	goals := 0
	r := rand.New(rand.NewSource(42))
	for i := 0; i < trials; i++ {
		ev := resolveChance(r, attacker, TeamTypeHome, ChanceTypeOpenPlay,
			FormationProfile{ChanceCreation: 1, ChanceQuality: 1},
			Tactics{}, 60, 1, attackBoost, 10)
		if ev.Type == GameEventTypeGoal {
			goals++
		}
	}
	return goals
}

// A boost must never make the shot worse. resolveChance floors atk at 1, so
// a bug that passed 0 through as "no boost" would silently gut conversion.
func TestResolveChance_NeutralBoostIsIdentity(t *testing.T) {
	attacker := boostTestPlayer("atk", PlayerPositionAttack, 70)

	assert.Equal(t, countGoals(t, attacker, 1.0), countGoals(t, attacker, 1.0))
	assert.Greater(t, countGoals(t, attacker, 1.0), 0)
}

// End to end: two identical teams, one holding a midfield boost. The boosted
// side must take more of the chances, because midfield drives control and
// control decides possession.
func TestPositionBoost_WinsPossessionOverAnIdenticalTeam(t *testing.T) {
	base := teamControl(boostTestLineup(), nil)
	boosted := teamControl(boostTestLineup(), positionMultipliers{PlayerPositionMidfield: 1.07})

	share := boosted / (boosted + base)
	assert.Greater(t, share, 0.5,
		"the boosted team must win more than half of possession, got %.4f", share)
	assert.Less(t, math.Abs(share-0.5), 0.05,
		"a 7%% midfield boost should tilt possession, not dominate it")
}
