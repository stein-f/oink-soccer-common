package soccer_test

import (
	"github.com/mroth/weightedrand"
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
	"testing"
)

var injuryWeightsDefaults = []weightedrand.Choice{
	{Item: false, Weight: 50},
	{Item: true, Weight: 1},
}

var injuryWeightsInjuryPronePlayers = []weightedrand.Choice{
	{Item: false, Weight: 25},
	{Item: true, Weight: 1},
}

func TestApplyInjuries_NotInjured(t *testing.T) {
	_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, testdata.FixedRandSource(1))
	assert.Equal(t, false, isInjured)
}

func TestApplyInjuries_Injured(t *testing.T) {
	injury, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, testdata.FixedRandSource(83))
	assert.Equal(t, true, isInjured)
	assert.Equal(t, "Mascot Mishap", injury.Name)
}

func TestApplyInjuries_ExpectedFrequencyInRange(t *testing.T) {
	iterations := 100_000
	injuryCount := 0
	randSource := testdata.FixedRandSource(42) // Fixed seed for deterministic test
	for i := 0; i < iterations; i++ {
		_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, randSource)
		if isInjured {
			injuryCount++
		}
	}

	// Tightened range based on expected value and standard deviation
	expectedMin := 1829
	expectedMax := 2091

	assert.GreaterOrEqual(t, injuryCount, expectedMin)
	assert.LessOrEqual(t, injuryCount, expectedMax)
}
