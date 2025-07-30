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
	// Default aggression level of 50 (medium)
	_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 50, testdata.FixedRandSource(1))
	assert.Equal(t, false, isInjured)
}

// TestApplyInjuries_AggressionEffect tests that higher aggression increases injury probability
func TestApplyInjuries_AggressionEffect(t *testing.T) {
	// Run many iterations to get statistically significant results
	iterations := 10_000

	// Test with no aggression
	noAggressionInjuries := 0
	randSourceNoAgg := testdata.FixedRandSource(42) // Fixed seed for deterministic test
	for i := 0; i < iterations; i++ {
		_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 0, randSourceNoAgg)
		if isInjured {
			noAggressionInjuries++
		}
	}

	// Test with maximum aggression
	maxAggressionInjuries := 0
	randSourceMaxAgg := testdata.FixedRandSource(42) // Same seed for fair comparison
	for i := 0; i < iterations; i++ {
		_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 100, randSourceMaxAgg)
		if isInjured {
			maxAggressionInjuries++
		}
	}

	// Maximum aggression should result in significantly more injuries than no aggression
	assert.Greater(t, maxAggressionInjuries, noAggressionInjuries)

	// Test that injury-prone players get injured more often
	normalPlayerInjuries := 0
	injuryPronePlayerInjuries := 0
	randSourceNormal := testdata.FixedRandSource(42)
	randSourceInjuryProne := testdata.FixedRandSource(42)

	for i := 0; i < iterations; i++ {
		_, isInjuredNormal := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 50, randSourceNormal)
		if isInjuredNormal {
			normalPlayerInjuries++
		}

		_, isInjuredProne := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, true, 50, randSourceInjuryProne)
		if isInjuredProne {
			injuryPronePlayerInjuries++
		}
	}

	// Injury-prone players should get injured more often
	assert.Greater(t, injuryPronePlayerInjuries, normalPlayerInjuries)
}

func TestApplyInjuries_ExpectedFrequencyInRange(t *testing.T) {
	iterations := 100_000
	injuryCount := 0
	randSource := testdata.FixedRandSource(42) // Fixed seed for deterministic test

	// Default aggression level of 50 (medium)
	for i := 0; i < iterations; i++ {
		_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 50, randSource)
		if isInjured {
			injuryCount++
		}
	}

	// Adjusted range based on observed behavior with aggression at 50
	// The actual number of injuries will depend on the implementation of adjustInjuryWeightsByAggression
	// Using the ScalingFunction from scaling.go instead of the mathematical formula
	expectedMin := 2000
	expectedMax := 2500

	assert.GreaterOrEqual(t, injuryCount, expectedMin)
	assert.LessOrEqual(t, injuryCount, expectedMax)
}

func TestApplyInjuries_AggressionImpact(t *testing.T) {
	iterations := 10_000

	// Test with low aggression (10)
	lowAggressionInjuries := 0
	randSourceLow := testdata.FixedRandSource(42) // Fixed seed for deterministic test
	for i := 0; i < iterations; i++ {
		_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 10, randSourceLow)
		if isInjured {
			lowAggressionInjuries++
		}
	}

	// Test with high aggression (90)
	highAggressionInjuries := 0
	randSourceHigh := testdata.FixedRandSource(42) // Same seed for fair comparison
	for i := 0; i < iterations; i++ {
		_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 90, randSourceHigh)
		if isInjured {
			highAggressionInjuries++
		}
	}

	// High aggression should result in significantly more injuries than low aggression
	// We expect at least 50% more injuries with high aggression
	assert.Greater(t, highAggressionInjuries, lowAggressionInjuries+(lowAggressionInjuries/2))
}
