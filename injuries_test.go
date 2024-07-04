package soccer_test

import (
	"github.com/mroth/weightedrand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, "Squirrel Scare", injury.Name)
}
