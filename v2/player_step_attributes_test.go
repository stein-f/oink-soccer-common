package soccer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepAttributes_FallBackToCoreRatings(t *testing.T) {
	// A player from before season 19 carries none of the player-on-player
	// attributes; each one reads the core rating that used to stand in for it.
	p := PlayerAttributes{
		GoalkeeperRating: 81, DefenseRating: 70, SpeedRating: 77, ControlRating: 84, AttackRating: 60,
		Technique: 66, PrimaryPosition: PlayerPositionMidfield,
	}

	assert.Equal(t, 84, p.EffectiveDribbling(), "control")
	assert.Equal(t, 84, p.EffectivePassing(), "control")
	assert.Equal(t, 66, p.EffectiveCrossing(), "technique")
	assert.Equal(t, 77, p.EffectivePace(), "speed")
	assert.Equal(t, 65, p.EffectiveStrength(), "a midfielder: mean of defense and attack")
	assert.Equal(t, 70, p.EffectiveInterceptions(), "defense")
	assert.Equal(t, 60, p.EffectivePositioning(), "attack")
	assert.Equal(t, 70, p.EffectiveMarking(), "defense")
	assert.Equal(t, 66, p.EffectiveShotPower(), "technique")
	assert.Equal(t, 81, p.EffectiveReflexes(), "goalkeeper rating")
	assert.Equal(t, 81, p.EffectiveHandling(), "goalkeeper rating")
	assert.Equal(t, 81, p.EffectiveOneOnOne(), "goalkeeper rating")

	p.Technique = 0
	assert.Equal(t, 84, p.EffectiveCrossing(), "no technique either: control")
	assert.Equal(t, 84, p.EffectiveShotPower(), "no technique either: control")

	p.PrimaryPosition = PlayerPositionDefense
	assert.Equal(t, 70, p.EffectiveStrength(), "a defender's strength reads from defense")
	p.PrimaryPosition = PlayerPositionAttack
	assert.Equal(t, 60, p.EffectiveStrength(), "an attacker's from attack")
}

func TestStepAttributes_UseTheirOwnValueWhenSet(t *testing.T) {
	p := PlayerAttributes{
		GoalkeeperRating: 10, DefenseRating: 10, SpeedRating: 10, ControlRating: 10, AttackRating: 10,
		Dribbling: 91, Passing: 82, Crossing: 73, Pace: 94, Strength: 65, Interceptions: 56,
		Positioning: 87, Marking: 48, ShotPower: 79, Reflexes: 31, Handling: 32, OneOnOne: 33,
	}

	assert.Equal(t, 91, p.EffectiveDribbling())
	assert.Equal(t, 82, p.EffectivePassing())
	assert.Equal(t, 73, p.EffectiveCrossing())
	assert.Equal(t, 94, p.EffectivePace())
	assert.Equal(t, 65, p.EffectiveStrength())
	assert.Equal(t, 56, p.EffectiveInterceptions())
	assert.Equal(t, 87, p.EffectivePositioning())
	assert.Equal(t, 48, p.EffectiveMarking())
	assert.Equal(t, 79, p.EffectiveShotPower())
	assert.Equal(t, 31, p.EffectiveReflexes())
	assert.Equal(t, 32, p.EffectiveHandling())
	assert.Equal(t, 33, p.EffectiveOneOnOne())
}
