package soccer

import (
	"math"
	"testing"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
	"github.com/stretchr/testify/assert"
)

// scoring_test.go uses package-internal access (no _test suffix on the
// package name) so it can drive the unexported scoring helpers directly.
// These functions are the engine's core math; testing them in isolation
// lets us pin behavior independently of the eventual phase-based engine.

func legacyAttrs() PlayerAttributes {
	return PlayerAttributes{
		ControlRating:   80,
		AttackRating:    70,
		DefenseRating:   60,
		SpeedRating:     90,
		PrimaryPosition: PlayerPositionMidfield,
		Positions:       []PlayerPosition{PlayerPositionMidfield},
	}
}

// Backfill: legacy data with only SpeedRating must produce the same scores
// as the explicit-attribute math when Pace/Recovery/WorkRate fall back to
// SpeedRating. This is what lets us migrate rosters incrementally.
func TestEffectiveAccessors_FallBackToSpeedRating(t *testing.T) {
	p := legacyAttrs()
	assert.Equal(t, 90, p.EffectivePace())
	assert.Equal(t, 90, p.EffectiveRecovery())
	assert.Equal(t, 90, p.EffectiveWorkRate())
}

func TestEffectiveAccessors_PreferExplicitFields(t *testing.T) {
	p := legacyAttrs()
	p.Pace = 50
	p.Recovery = 40
	p.WorkRate = 30

	assert.Equal(t, 50, p.EffectivePace())
	assert.Equal(t, 40, p.EffectiveRecovery())
	assert.Equal(t, 30, p.EffectiveWorkRate())
}

// The whole point of Phase 2 — bumping one physical attribute must affect
// only the corresponding score, not all three. v1 violated this by folding
// SpeedRating into every score; v2 keeps them orthogonal.
func TestRawScores_PhysicalAttributesAreOrthogonal(t *testing.T) {
	base := PlayerAttributes{
		ControlRating: 80, AttackRating: 80, DefenseRating: 80,
		Pace: 50, Recovery: 50, WorkRate: 50,
	}

	baseControl := rawControl(base, Tactics{})
	baseAttack := rawAttack(base)
	baseDefense := rawDefense(base, Tactics{})

	t.Run("pace bump affects attack only", func(t *testing.T) {
		bumped := base
		bumped.Pace = 90
		assert.Equal(t, baseControl, rawControl(bumped, Tactics{}))
		assert.Greater(t, rawAttack(bumped), baseAttack)
		assert.Equal(t, baseDefense, rawDefense(bumped, Tactics{}))
	})

	t.Run("recovery bump affects defense only", func(t *testing.T) {
		bumped := base
		bumped.Recovery = 90
		assert.Equal(t, baseControl, rawControl(bumped, Tactics{}))
		assert.Equal(t, baseAttack, rawAttack(bumped))
		assert.Greater(t, rawDefense(bumped, Tactics{}), baseDefense)
	})

	t.Run("work rate bump affects control only", func(t *testing.T) {
		bumped := base
		bumped.WorkRate = 90
		assert.Greater(t, rawControl(bumped, Tactics{}), baseControl)
		assert.Equal(t, baseAttack, rawAttack(bumped))
		assert.Equal(t, baseDefense, rawDefense(bumped, Tactics{}))
	})
}

func TestRawScores_LegacySpeedRatingMatchesExplicit(t *testing.T) {
	legacy := legacyAttrs()
	explicit := legacyAttrs()
	explicit.SpeedRating = 0
	explicit.Pace = legacy.SpeedRating
	explicit.Recovery = legacy.SpeedRating
	explicit.WorkRate = legacy.SpeedRating

	assert.Equal(t, rawControl(legacy, Tactics{}), rawControl(explicit, Tactics{}))
	assert.Equal(t, rawAttack(legacy), rawAttack(explicit))
	assert.Equal(t, rawDefense(legacy, Tactics{}), rawDefense(explicit, Tactics{}))
}

func TestRawDefense_GoalkeeperUsesGoalkeeperRating(t *testing.T) {
	gk := PlayerAttributes{
		GoalkeeperRating: 88,
		DefenseRating:    20, // low — must be ignored
		Recovery:         70,
		PrimaryPosition:  PlayerPositionGoalkeeper,
		Positions:        []PlayerPosition{PlayerPositionGoalkeeper},
	}
	outfield := gk
	outfield.PrimaryPosition = PlayerPositionDefense
	outfield.Positions = []PlayerPosition{PlayerPositionDefense}

	// Same physicals + recovery, but the GK uses GoalkeeperRating(88) and
	// the outfielder uses DefenseRating(20). The GK score must be much higher.
	assert.Greater(t, rawDefense(gk, Tactics{}), rawDefense(outfield, Tactics{}))
}

func TestPlayerScore_OutOfPositionPenaltyApplies(t *testing.T) {
	mid := SelectedPlayer{
		Attributes:       legacyAttrs(),
		SelectedPosition: PlayerPositionMidfield,
	}
	misplaced := mid
	misplaced.SelectedPosition = PlayerPositionAttack

	inPos := playerControl(mid, Tactics{})
	outOfPos := playerControl(misplaced, Tactics{})

	assert.InDelta(t, inPos*0.85, outOfPos, 1e-9, "out-of-position must scale by 0.85")
}

func TestPlayerScore_AnyPositionNeverPenalised(t *testing.T) {
	utility := SelectedPlayer{
		Attributes: PlayerAttributes{
			ControlRating: 80, AttackRating: 80, DefenseRating: 80, SpeedRating: 70,
			PrimaryPosition: PlayerPositionAny,
			Positions:       []PlayerPosition{PlayerPositionAny},
		},
		SelectedPosition: PlayerPositionAttack,
	}
	assert.False(t, utility.IsOutOfPosition())
	// Score should equal the curved raw value — no 0.85 out-of-position scale.
	assert.Equal(t, tuning.SkillCurve(rawControl(utility.Attributes, Tactics{})), playerControl(utility, Tactics{}))
}

func TestPlayerScore_InjuryReducesScore(t *testing.T) {
	healthy := SelectedPlayer{
		Attributes:       legacyAttrs(),
		SelectedPosition: PlayerPositionMidfield,
	}
	injured := healthy
	injured.Injury = &InjuryEvent{Injury: Injury{StatsReduction: 0.85}}

	expected := playerControl(healthy, Tactics{}) * 0.85
	assert.InDelta(t, expected, playerControl(injured, Tactics{}), 1e-9)
}

func TestPlayerScore_NilInjuryIsIdentity(t *testing.T) {
	sp := SelectedPlayer{
		Attributes:       legacyAttrs(),
		SelectedPosition: PlayerPositionMidfield,
	}
	// With no injury, playerControl is the curved raw value (no further multipliers).
	assert.Equal(t, tuning.SkillCurve(rawControl(sp.Attributes, Tactics{})), playerControl(sp, Tactics{}))
}

// Position weights sum to 1.0 (verified in tuning_test.go), so a team where
// every player has the same per-action score X should produce a team score
// of X. With the skill curve applied at the player level, X is the curved
// value of the raw rating (e.g. 80 → ~51.2 at exponent 3.0). This is the
// smell test that the weighted-average math is right.
func TestTeamScore_UniformPlayersProduceUniformTeamScore(t *testing.T) {
	mk := func(pos PlayerPosition) SelectedPlayer {
		return SelectedPlayer{
			Attributes: PlayerAttributes{
				ControlRating: 80, AttackRating: 80, DefenseRating: 80,
				GoalkeeperRating: 80, SpeedRating: 80,
				PrimaryPosition: pos,
				Positions:       []PlayerPosition{pos},
			},
			SelectedPosition: pos,
		}
	}
	lineup := GameLineup{Players: []SelectedPlayer{
		mk(PlayerPositionGoalkeeper),
		mk(PlayerPositionDefense),
		mk(PlayerPositionMidfield),
		mk(PlayerPositionAttack),
	}}

	want := tuning.SkillCurve(80)
	assert.InDelta(t, want, teamControl(lineup), 1e-9)
	assert.InDelta(t, want, teamDefense(lineup), 1e-9)
}

// v1 special-cased the Box formation in team scoring (Box gave attackers
// 60% of the control weight instead of 15%, compounding the formation's
// dominance). v2 must use the same position weights regardless of formation.
func TestTeamScore_FormationDoesNotChangePositionWeights(t *testing.T) {
	mk := func(pos PlayerPosition, ctrl int) SelectedPlayer {
		return SelectedPlayer{
			Attributes: PlayerAttributes{
				ControlRating:   ctrl,
				PrimaryPosition: pos,
				Positions:       []PlayerPosition{pos},
			},
			SelectedPosition: pos,
		}
	}
	// Two attackers with high control, no midfielders — Box-shaped lineup.
	players := []SelectedPlayer{
		mk(PlayerPositionGoalkeeper, 50),
		mk(PlayerPositionDefense, 60),
		mk(PlayerPositionDefense, 60),
		mk(PlayerPositionAttack, 95),
		mk(PlayerPositionAttack, 95),
	}

	withDiamond := GameLineup{Team: Team{Formation: FormationTypeDiamond}, Players: players}
	withBox := GameLineup{Team: Team{Formation: FormationTypeBox}, Players: players}

	// Same players + same position-weights ⇒ same score regardless of formation.
	assert.Equal(t, teamControl(withDiamond), teamControl(withBox),
		"formation must not change the position-weight calculation")
}

// Sentinel: pin the attribute math so we don't drift accidentally during
// tuning. If these change intentionally, update the constants in tuning.go
// in the same PR and explain why.
func TestRawScores_LockedAgainstV1Math(t *testing.T) {
	p := PlayerAttributes{
		ControlRating: 80, AttackRating: 70, DefenseRating: 60, SpeedRating: 90,
		PrimaryPosition: PlayerPositionDefense,
		Positions:       []PlayerPosition{PlayerPositionDefense},
	}
	// (80*4 + 90) / 5 = 82 — control unchanged from v1.
	assert.Equal(t, math.Round(82.0), rawControl(p, Tactics{}))
	// (70*3 + 90) / 4 = 75 — attack unchanged from v1 (open-play default).
	assert.Equal(t, math.Round(75.0), rawAttack(p))
	// Outfield defense adds tackling weight: (def*5 + tackling*2 + recovery) / 8.
	// With Tackling unset, EffectiveTackling backfills to DefenseRating (60), so
	// (60*5 + 60*2 + 90) / 8 = 510 / 8 = 63.75 ⇒ 64.
	assert.Equal(t, math.Round(64.0), rawDefense(p, Tactics{}))

	// Goalkeeper still uses the legacy formula (gk*5 + recovery) / 6.
	gk := p
	gk.GoalkeeperRating = 88
	gk.PrimaryPosition = PlayerPositionGoalkeeper
	gk.Positions = []PlayerPosition{PlayerPositionGoalkeeper}
	// (88*5 + 90) / 6 = 88.33 ⇒ 88.
	assert.Equal(t, math.Round(88.0), rawDefense(gk, Tactics{}))
}
