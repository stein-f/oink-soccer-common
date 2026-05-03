package soccer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// build_test.go covers the chance-type-aware attack score. The point of
// per-chance attribute weights is that different player archetypes (target
// man, speedster, technical) each have chance types they shine on — managers
// can pick a build with their tactics in mind, and no single physical
// attribute makes a player obsolete.

type buildArchetype struct {
	name    string
	attrs   PlayerAttributes
	bestOn  ChanceType
	worstOn ChanceType
}

func buildFixtures() []buildArchetype {
	mk := func(atk, pace, ctrl int) PlayerAttributes {
		return PlayerAttributes{
			AttackRating:    atk,
			ControlRating:   ctrl,
			Pace:            pace,
			PrimaryPosition: PlayerPositionAttack,
			Positions:       []PlayerPosition{PlayerPositionAttack},
		}
	}
	return []buildArchetype{
		{
			name:    "target man (atk 88, pace 55, ctrl 78)",
			attrs:   mk(88, 55, 78),
			bestOn:  ChanceTypeCorner,
			worstOn: ChanceTypeGoalKeeperShot,
		},
		{
			name:  "speedster (atk 85, pace 92, ctrl 65)",
			attrs: mk(85, 92, 65),
			// Pace*3 in the GoalkeeperShot formula crushes everything else.
			bestOn: ChanceTypeGoalKeeperShot,
			// FreeKick is technique-pure ((atk + technique*3) / 4 with technique
			// backfilling to ctrl=65) — speedsters with low control are awful at it.
			worstOn: ChanceTypeFreeKick,
		},
		{
			name:  "technical (atk 80, pace 72, ctrl 90)",
			attrs: mk(80, 72, 90),
			// FreeKick is the most technique-pure chance type ((atk + technique*3) / 4)
			// — a high-control, mid-attack technician edges Long Range here.
			bestOn: ChanceTypeFreeKick,
			// 1-on-1 breakaways punish the technical player's mid-range pace
			// (pace*3 dominates the formula).
			worstOn: ChanceTypeGoalKeeperShot,
		},
	}
}

// Each archetype must score highest on the chance type their build is suited
// for, and lowest on the chance type that punishes their weakness. This is
// the "no useless players" invariant — every build has a chance type where
// it shines.
func TestRawAttackForChance_ArchetypeBestAndWorst(t *testing.T) {
	chances := []ChanceType{
		ChanceTypeOpenPlay,
		ChanceTypeCross,
		ChanceTypeCorner,
		ChanceTypeLongRange,
		ChanceTypeFreeKick,
		ChanceTypePenalty,
		ChanceTypeGoalKeeperShot,
	}

	for _, b := range buildFixtures() {
		t.Run(b.name, func(t *testing.T) {
			scores := make(map[ChanceType]float64)
			for _, ct := range chances {
				scores[ct] = rawAttackForChance(b.attrs, ct)
			}

			best, worst := chances[0], chances[0]
			for _, ct := range chances[1:] {
				if scores[ct] > scores[best] {
					best = ct
				}
				if scores[ct] < scores[worst] {
					worst = ct
				}
			}

			assert.Equal(t, b.bestOn, best,
				"%s should score highest on %s, got %s (scores: %v)",
				b.name, b.bestOn, best, scores)
			assert.Equal(t, b.worstOn, worst,
				"%s should score lowest on %s, got %s (scores: %v)",
				b.name, b.worstOn, worst, scores)
		})
	}
}

// Direct comparison: on a corner, the target man must out-score the speedster
// even though their AttackRatings are similar — that's the whole point of the
// per-chance weights.
func TestRawAttackForChance_TargetManBeatsSpeedsterOnCorners(t *testing.T) {
	targetMan := PlayerAttributes{AttackRating: 88, Pace: 55, ControlRating: 78}
	speedster := PlayerAttributes{AttackRating: 85, Pace: 92, ControlRating: 65}

	tmCorner := rawAttackForChance(targetMan, ChanceTypeCorner)
	spCorner := rawAttackForChance(speedster, ChanceTypeCorner)

	assert.Greater(t, tmCorner, spCorner,
		"target man (atk %d ctrl %d) should out-score speedster (atk %d ctrl %d) on corners — got %v vs %v",
		targetMan.AttackRating, targetMan.ControlRating, speedster.AttackRating, speedster.ControlRating, tmCorner, spCorner)

	tmBreak := rawAttackForChance(targetMan, ChanceTypeGoalKeeperShot)
	spBreak := rawAttackForChance(speedster, ChanceTypeGoalKeeperShot)

	assert.Greater(t, spBreak, tmBreak,
		"speedster should out-score target man on 1-on-1 breakaways — got %v vs %v",
		spBreak, tmBreak)
}

// Backward-compat: rawAttack (the chance-type-agnostic helper) must still
// match the v1 (skill*3 + pace*1) / 4 formula. Sentinel against accidental
// drift in the default weights.
func TestRawAttack_DefaultWeightsLockedAgainstV1Math(t *testing.T) {
	p := PlayerAttributes{AttackRating: 70, SpeedRating: 90}
	// (70*3 + 90) / 4 = 75
	assert.Equal(t, 75.0, rawAttack(p))
}

// A chance type that doesn't declare attack weights (or declares Divisor=0)
// must fall back to the defaults so the engine never divides by zero.
func TestRawAttackForChance_FallsBackOnUnknownChanceType(t *testing.T) {
	p := PlayerAttributes{AttackRating: 70, SpeedRating: 90}
	// "" isn't in the chanceTypeProfiles map ⇒ defaultAttackScore ⇒ 75.
	assert.Equal(t, rawAttack(p), rawAttackForChance(p, ChanceType("")))
}

// With explicit specialist attributes populated, a player who looks "average"
// on the legacy composites can dominate a single chance type. This is the
// payoff for adding Heading / Composure / Technique / Finishing — managers
// can now build a corner specialist or a penalty specialist who out-performs
// a generally-better striker on their preferred chance type.
func TestRawAttackForChance_SpecialistAttributesUnlockTopBuilds(t *testing.T) {
	t.Run("aerial specialist beats a generic striker on corners", func(t *testing.T) {
		// Note: explicit Heading on the generic striker — without it, Effective-
		// Heading backfills to AttackRating and a high-attack generic gets a
		// free aerial boost. With realistic FIFA-populated rosters (where the
		// allocation pipeline sets Heading explicitly) this isn't a concern.
		generic := PlayerAttributes{AttackRating: 88, Pace: 70, ControlRating: 70, Heading: 70}
		aerial := PlayerAttributes{AttackRating: 80, Pace: 60, ControlRating: 65, Heading: 92}

		assert.Greater(t, rawAttackForChance(aerial, ChanceTypeCorner),
			rawAttackForChance(generic, ChanceTypeCorner),
			"aerial specialist (heading=92) should out-score a higher-attack generic striker (heading=70) on corners")
	})

	t.Run("clutch finisher beats a generic striker on penalties", func(t *testing.T) {
		generic := PlayerAttributes{AttackRating: 88, Pace: 70, ControlRating: 70}
		clutch := PlayerAttributes{AttackRating: 80, Pace: 60, ControlRating: 70, Composure: 95}

		assert.Greater(t, rawAttackForChance(clutch, ChanceTypePenalty),
			rawAttackForChance(generic, ChanceTypePenalty),
			"clutch finisher (composure=95) should out-score a higher-attack generic striker on penalties")
	})

	t.Run("free-kick specialist beats a generic striker on free kicks", func(t *testing.T) {
		generic := PlayerAttributes{AttackRating: 88, Pace: 70, ControlRating: 70}
		fkSpec := PlayerAttributes{AttackRating: 78, Pace: 65, ControlRating: 75, Technique: 94}

		assert.Greater(t, rawAttackForChance(fkSpec, ChanceTypeFreeKick),
			rawAttackForChance(generic, ChanceTypeFreeKick),
			"free-kick specialist (technique=94) should out-score a generic striker on free kicks")
	})

	t.Run("poacher beats a generic striker on open play", func(t *testing.T) {
		// Same backfill caveat as the aerial test — explicit Finishing on the
		// generic so its high AttackRating doesn't double-count via EffectiveFinishing.
		generic := PlayerAttributes{AttackRating: 85, Pace: 80, ControlRating: 70, Finishing: 70}
		poacher := PlayerAttributes{AttackRating: 80, Pace: 75, ControlRating: 65, Finishing: 95}

		assert.Greater(t, rawAttackForChance(poacher, ChanceTypeOpenPlay),
			rawAttackForChance(generic, ChanceTypeOpenPlay),
			"poacher (finishing=95) should out-score a higher-attack generic striker (finishing=70) on open play")
	})

	t.Run("specialist edge does NOT carry to chance types that don't reward their attribute", func(t *testing.T) {
		// An aerial specialist with otherwise modest attributes shouldn't
		// out-score a faster, higher-attack striker on a 1-on-1 breakaway —
		// pace and finishing matter there, not heading.
		generic := PlayerAttributes{AttackRating: 85, Pace: 92, ControlRating: 70}
		aerial := PlayerAttributes{AttackRating: 80, Pace: 60, ControlRating: 65, Heading: 95}

		assert.Greater(t, rawAttackForChance(generic, ChanceTypeGoalKeeperShot),
			rawAttackForChance(aerial, ChanceTypeGoalKeeperShot),
			"a fast striker should still beat an aerial specialist on 1-on-1 breakaways")
	})
}

// Defense gets the same treatment via the new Tackling attribute. A "smart
// but soft" defender (high DefenseRating, low Tackling) should now score
// lower than an enforcer-style defender (high Tackling) — splitting these
// is the whole point of the new attribute.
func TestRawDefense_TacklingAttributeShapesOutfieldDefense(t *testing.T) {
	soft := PlayerAttributes{
		DefenseRating:   90,
		Tackling:        60, // smart positioning, doesn't tackle
		Recovery:        85,
		PrimaryPosition: PlayerPositionDefense,
		Positions:       []PlayerPosition{PlayerPositionDefense},
	}
	enforcer := PlayerAttributes{
		DefenseRating:   80,
		Tackling:        92, // crunching dispossessor
		Recovery:        85,
		PrimaryPosition: PlayerPositionDefense,
		Positions:       []PlayerPosition{PlayerPositionDefense},
	}

	assert.Greater(t, rawDefense(enforcer, Tactics{}), rawDefense(soft, Tactics{}),
		"enforcer (tackling=92) should out-score a higher-DefenseRating, lower-tackling defender")
}

// Goalkeepers don't use Tackling — saves come from GoalkeeperRating + Recovery.
// Setting Tackling on a GK must have no effect.
func TestRawDefense_GoalkeeperIgnoresTackling(t *testing.T) {
	gk := PlayerAttributes{
		GoalkeeperRating: 88,
		Recovery:         70,
		PrimaryPosition:  PlayerPositionGoalkeeper,
		Positions:        []PlayerPosition{PlayerPositionGoalkeeper},
	}
	withTackling := gk
	withTackling.Tackling = 95

	assert.Equal(t, rawDefense(gk, Tactics{}), rawDefense(withTackling, Tactics{}),
		"goalkeepers must ignore Tackling — saves are driven by GoalkeeperRating + Recovery")
}
