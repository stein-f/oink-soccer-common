package soccer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// tactics_attribute_test.go pins the invariant that team Tactics changes
// which player attributes are most valuable on the pitch. Press shifts
// control toward work rate; line height shifts defense toward recovery.
// This is what makes a high-press Y-formation lineup *want* different
// midfielders than a possession-style Diamond.

// A high-press team values a high-WorkRate midfielder more than a passive
// team does. Same player, same rating, two different tactical contexts.
func TestRawControl_PressShiftsValueOfWorkRate(t *testing.T) {
	// Two midfielders with the same composite control but very different
	// physical profiles.
	technician := PlayerAttributes{ControlRating: 90, WorkRate: 60}
	workhorse := PlayerAttributes{ControlRating: 75, WorkRate: 95}

	// Under low press (passive), the technician's skill dominates.
	lowPress := Tactics{Press: PressLevelLow}
	techLow := rawControl(technician, lowPress)
	workLow := rawControl(workhorse, lowPress)
	assert.Greater(t, techLow, workLow,
		"low press should favour the technician (skill-heavy weighting)")

	// Under high press, the workhorse's stamina matters far more.
	highPress := Tactics{Press: PressLevelHigh}
	techHigh := rawControl(technician, highPress)
	workHigh := rawControl(workhorse, highPress)

	// Tilt: under high press the workhorse closes the gap or overtakes.
	// We assert the *swing* is positive (workhorse gains relative to technician).
	swingLow := techLow - workLow
	swingHigh := techHigh - workHigh
	assert.Greater(t, swingLow, swingHigh,
		"high press should narrow or overturn the technician's edge — got swingLow=%v, swingHigh=%v",
		swingLow, swingHigh)
}

// A high defensive line punishes a positionally-strong but slow defender
// (low Recovery) and rewards a fast one. Deep line is the opposite — a
// recovery-poor but well-positioned defender holds up fine.
func TestRawDefense_LineHeightShiftsValueOfRecovery(t *testing.T) {
	libero := PlayerAttributes{
		DefenseRating:   92,
		Tackling:        80,
		Recovery:        55, // slow but reads the game
		PrimaryPosition: PlayerPositionDefense,
		Positions:       []PlayerPosition{PlayerPositionDefense},
	}
	chaser := PlayerAttributes{
		DefenseRating:   75,
		Tackling:        80,
		Recovery:        95, // fast, average reader
		PrimaryPosition: PlayerPositionDefense,
		Positions:       []PlayerPosition{PlayerPositionDefense},
	}

	// Deep line: positioning rules; libero outperforms chaser.
	deep := Tactics{LineHeight: LineHeightDeep}
	liberoDeep := rawDefense(libero, deep)
	chaserDeep := rawDefense(chaser, deep)
	assert.Greater(t, liberoDeep, chaserDeep,
		"deep line should favour the positionally-strong libero")

	// High line: chaser overtakes — recovery becomes the load-bearing attribute.
	high := Tactics{LineHeight: LineHeightHigh}
	liberoHigh := rawDefense(libero, high)
	chaserHigh := rawDefense(chaser, high)
	assert.Greater(t, chaserHigh, liberoHigh,
		"high line should favour the fast chaser — recovery matters more than positioning")
}

// Goalkeepers shouldn't be affected by line height — they're not the ones
// chasing balls behind the defensive line. Sanity check that the GK formula
// is tactic-invariant.
func TestRawDefense_GoalkeeperIgnoresLineHeight(t *testing.T) {
	gk := PlayerAttributes{
		GoalkeeperRating: 88,
		Recovery:         70,
		PrimaryPosition:  PlayerPositionGoalkeeper,
		Positions:        []PlayerPosition{PlayerPositionGoalkeeper},
	}

	deep := rawDefense(gk, Tactics{LineHeight: LineHeightDeep})
	high := rawDefense(gk, Tactics{LineHeight: LineHeightHigh})
	assert.Equal(t, deep, high, "goalkeeper defense score must not depend on line height")
}

// Empty Tactics{} ⇒ legacy / neutral weighting. This guard keeps existing
// callers (and the v1-baseline snapshots) behaving unchanged.
func TestRawScores_NeutralTacticsMatchLegacyWeights(t *testing.T) {
	mid := PlayerAttributes{ControlRating: 80, WorkRate: 70}
	// Neutral: (80*4 + 70) / 5 = 78.
	assert.Equal(t, 78.0, rawControl(mid, Tactics{}))

	def := PlayerAttributes{
		DefenseRating:   80,
		Tackling:        80,
		Recovery:        70,
		PrimaryPosition: PlayerPositionDefense,
		Positions:       []PlayerPosition{PlayerPositionDefense},
	}
	// Neutral: (80*5 + 80*2 + 70) / 8 = (400 + 160 + 70) / 8 = 78.75 ⇒ 79.
	assert.Equal(t, 79.0, rawDefense(def, Tactics{}))
}
