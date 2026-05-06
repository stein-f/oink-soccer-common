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

// Backfill: when WorkRate isn't set, EffectiveWorkRate falls back to
// SpeedRating (the only physical-style accessor still in the model after
// Pace + Recovery were consolidated back into SpeedRating).
func TestEffectiveWorkRate_FallsBackToSpeedRating(t *testing.T) {
	p := legacyAttrs()
	assert.Equal(t, 90, p.EffectiveWorkRate())
}

func TestEffectiveWorkRate_PrefersExplicitField(t *testing.T) {
	p := legacyAttrs()
	p.WorkRate = 30
	assert.Equal(t, 30, p.EffectiveWorkRate())
}

// SpeedRating drives both attack and defense scoring. Bumping it must lift
// both raw scores; ControlRating-driven control must be unaffected.
func TestRawScores_SpeedRatingDrivesAttackAndDefense(t *testing.T) {
	base := PlayerAttributes{
		ControlRating: 80, AttackRating: 80, DefenseRating: 80,
		SpeedRating: 50,
	}

	baseControl := rawControl(base, Tactics{})
	baseAttack := rawAttack(base)
	baseDefense := rawDefense(base, Tactics{})

	t.Run("speed bump lifts attack and defense", func(t *testing.T) {
		bumped := base
		bumped.SpeedRating = 90
		assert.Greater(t, rawAttack(bumped), baseAttack)
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

func TestRawDefense_GoalkeeperUsesGoalkeeperRating(t *testing.T) {
	gk := PlayerAttributes{
		GoalkeeperRating: 88,
		DefenseRating:    20, // low — must be ignored
		SpeedRating:      70,
		PrimaryPosition:  PlayerPositionGoalkeeper,
		Positions:        []PlayerPosition{PlayerPositionGoalkeeper},
	}
	outfield := gk
	outfield.PrimaryPosition = PlayerPositionDefense
	outfield.Positions = []PlayerPosition{PlayerPositionDefense}

	// Same physicals, but the GK uses GoalkeeperRating(88) and the outfielder
	// uses DefenseRating(20). The GK score must be much higher.
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

// Playmaker is a focal-point lever, NOT a flat per-player boost. It must
// satisfy: (1) tagging the strongest controller in a position group raises
// team control, (2) tagging a weak player drops it, (3) tagging any player
// in a uniform group is neutral. Earlier implementation (×1.10 on the
// player's score) failed (2) — it gave even weak playmakers a free lift.
func TestPlaymaker_FocalPointShiftsAroundPositionMean(t *testing.T) {
	mk := func(pos PlayerPosition, ctrl int) SelectedPlayer {
		return SelectedPlayer{
			Attributes: PlayerAttributes{
				ControlRating:   ctrl,
				WorkRate:        ctrl, // keep work rate matched so rawControl is 1:1
				PrimaryPosition: pos,
				Positions:       []PlayerPosition{pos},
			},
			SelectedPosition: pos,
		}
	}
	withRole := func(p SelectedPlayer, role PlayerRole) SelectedPlayer {
		p.Role = role
		return p
	}

	gk := mk(PlayerPositionGoalkeeper, 70)
	def := mk(PlayerPositionDefense, 70)
	atk := mk(PlayerPositionAttack, 70)

	// Asymmetric midfield: strong (90), average (70), weak (50).
	t.Run("asymmetric midfield", func(t *testing.T) {
		strong := mk(PlayerPositionMidfield, 90)
		avg := mk(PlayerPositionMidfield, 70)
		weak := mk(PlayerPositionMidfield, 50)

		build := func(strongRole, weakRole PlayerRole) GameLineup {
			return GameLineup{Players: []SelectedPlayer{
				gk, def,
				withRole(strong, strongRole),
				avg,
				withRole(weak, weakRole),
				atk,
			}}
		}

		noPlaymaker := teamControl(build("", ""))
		strongPlaymaker := teamControl(build(PlayerRolePlaymaker, ""))
		weakPlaymaker := teamControl(build("", PlayerRolePlaymaker))

		t.Logf("teamControl: none=%.3f strong-PM=%.3f weak-PM=%.3f",
			noPlaymaker, strongPlaymaker, weakPlaymaker)

		assert.Greater(t, strongPlaymaker, noPlaymaker,
			"tagging the strongest mid (90) as Playmaker must raise team control")
		assert.Less(t, weakPlaymaker, noPlaymaker,
			"tagging the weakest mid (50) as Playmaker must lower team control — no free boost for poor picks")
	})

	// Uniform midfield: tagging any of three identically-rated mids is
	// neutral. This is the cleanest pin on the math — when there's no
	// score gap, there's nothing for the focal-point weighting to shift.
	t.Run("uniform midfield is neutral", func(t *testing.T) {
		uniform := func(role PlayerRole) GameLineup {
			a := mk(PlayerPositionMidfield, 70)
			b := mk(PlayerPositionMidfield, 70)
			c := mk(PlayerPositionMidfield, 70)
			return GameLineup{Players: []SelectedPlayer{
				gk, def,
				withRole(a, role),
				b, c, atk,
			}}
		}
		noPlaymaker := teamControl(uniform(""))
		withPlaymaker := teamControl(uniform(PlayerRolePlaymaker))
		assert.InDelta(t, noPlaymaker, withPlaymaker, 1e-9,
			"with all mids equal, Playmaker should not move team control")
	})
}

// Ball Winner: same focal-point semantics as Playmaker, but on the defensive
// side. A strong defender tagged as Ball Winner amplifies team defense; a
// weak one drags it down. Earlier implementation (×1.10 on the player's
// score) gave even poor Ball Winners a free lift.
func TestBallWinner_FocalPointShiftsAroundPositionMean(t *testing.T) {
	mk := func(pos PlayerPosition, def int) SelectedPlayer {
		return SelectedPlayer{
			Attributes: PlayerAttributes{
				DefenseRating:   def,
				SpeedRating:     def, // matched so rawDefense scales 1:1
				Tackling:        def,
				PrimaryPosition: pos,
				Positions:       []PlayerPosition{pos},
			},
			SelectedPosition: pos,
		}
	}
	withRole := func(p SelectedPlayer, role PlayerRole) SelectedPlayer {
		p.Role = role
		return p
	}

	gk := SelectedPlayer{
		Attributes: PlayerAttributes{
			GoalkeeperRating: 70, SpeedRating: 70,
			PrimaryPosition: PlayerPositionGoalkeeper,
			Positions:       []PlayerPosition{PlayerPositionGoalkeeper},
		},
		SelectedPosition: PlayerPositionGoalkeeper,
	}
	mid := mk(PlayerPositionMidfield, 70)
	atk := mk(PlayerPositionAttack, 70)

	t.Run("asymmetric defense", func(t *testing.T) {
		strong := mk(PlayerPositionDefense, 90)
		avg := mk(PlayerPositionDefense, 70)
		weak := mk(PlayerPositionDefense, 50)

		build := func(strongRole, weakRole PlayerRole) GameLineup {
			return GameLineup{Players: []SelectedPlayer{
				gk,
				withRole(strong, strongRole),
				avg,
				withRole(weak, weakRole),
				mid, atk,
			}}
		}

		none := teamDefense(build("", ""))
		strongBW := teamDefense(build(PlayerRoleBallWinner, ""))
		weakBW := teamDefense(build("", PlayerRoleBallWinner))

		t.Logf("teamDefense: none=%.3f strong-BW=%.3f weak-BW=%.3f", none, strongBW, weakBW)

		assert.Greater(t, strongBW, none,
			"tagging the strongest defender (90) as Ball Winner must raise team defense")
		assert.Less(t, weakBW, none,
			"tagging the weakest defender (50) as Ball Winner must lower team defense — no free boost for poor picks")
	})

	t.Run("uniform defense is neutral", func(t *testing.T) {
		uniform := func(role PlayerRole) GameLineup {
			a := mk(PlayerPositionDefense, 70)
			b := mk(PlayerPositionDefense, 70)
			c := mk(PlayerPositionDefense, 70)
			return GameLineup{Players: []SelectedPlayer{
				gk,
				withRole(a, role),
				b, c, mid, atk,
			}}
		}
		none := teamDefense(uniform(""))
		withBW := teamDefense(uniform(PlayerRoleBallWinner))
		assert.InDelta(t, none, withBW, 1e-9,
			"with all defenders equal, Ball Winner should not move team defense")
	})
}

// Captain quality: pin the helper that maps a player's attributes to a
// 0-100 captain quality score. Outfield uses ControlRating; keepers use
// GoalkeeperRating. Composure (or its backfill to ControlRating) is the
// other half. This is the input to the team-wide and self multipliers.
func TestCaptainQuality_PositionAware(t *testing.T) {
	mid := PlayerAttributes{
		ControlRating: 90, Composure: 80,
		PrimaryPosition: PlayerPositionMidfield,
		Positions:       []PlayerPosition{PlayerPositionMidfield},
	}
	assert.Equal(t, 85, captainQuality(mid),
		"outfield uses ControlRating + Composure; (90+80)/2 = 85")

	gk := PlayerAttributes{
		GoalkeeperRating: 90, ControlRating: 30, Composure: 80,
		PrimaryPosition: PlayerPositionGoalkeeper,
		Positions:       []PlayerPosition{PlayerPositionGoalkeeper},
	}
	assert.Equal(t, 85, captainQuality(gk),
		"keeper uses GoalkeeperRating instead of ControlRating; (90+80)/2 = 85, ignoring the low ctrl")

	legacy := PlayerAttributes{
		ControlRating:   80, // Composure unset → backfills to ControlRating
		PrimaryPosition: PlayerPositionMidfield,
		Positions:       []PlayerPosition{PlayerPositionMidfield},
	}
	assert.Equal(t, 80, captainQuality(legacy),
		"legacy roster (no Composure) backfills to ControlRating; (80+80)/2 = 80")
}

// Captain self-boost: a quality captain plays above themselves; a poor
// captain feels the burden. Pin both directions vs no-role as a smell test
// on the per-action multiplication.
func TestCaptain_SelfBoostShiftsWithQuality(t *testing.T) {
	mk := func(ctrl int) SelectedPlayer {
		return SelectedPlayer{
			Attributes: PlayerAttributes{
				ControlRating: ctrl, WorkRate: ctrl,
				PrimaryPosition: PlayerPositionMidfield,
				Positions:       []PlayerPosition{PlayerPositionMidfield},
			},
			SelectedPosition: PlayerPositionMidfield,
		}
	}

	strong := mk(95) // quality = 95, well above neutral 60
	weak := mk(25)   // quality = 25, well below neutral 60

	strongBase := playerControl(strong, Tactics{})
	weakBase := playerControl(weak, Tactics{})

	strong.Role = PlayerRoleCaptain
	weak.Role = PlayerRoleCaptain
	strongAsCaptain := playerControl(strong, Tactics{})
	weakAsCaptain := playerControl(weak, Tactics{})

	t.Logf("strong (q=95): base=%.3f as-captain=%.3f | weak (q=25): base=%.3f as-captain=%.3f",
		strongBase, strongAsCaptain, weakBase, weakAsCaptain)

	assert.Greater(t, strongAsCaptain, strongBase,
		"strong-quality captain must lift their own score (armband motivates)")
	assert.Less(t, weakAsCaptain, weakBase,
		"poor-quality captain must reduce their own score (burden of armband)")
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
