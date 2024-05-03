package soccer_test

import (
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
)

func TestApplyInjuries_NotInjured(t *testing.T) {
	_, isInjured := soccer.ApplyInjury(false, testdata.FixedRandSource(1))
	assert.Equal(t, false, isInjured)
}

func TestApplyInjuries_Injured(t *testing.T) {
	injury, isInjured := soccer.ApplyInjury(false, testdata.FixedRandSource(83))
	assert.Equal(t, true, isInjured)
	assert.Equal(t, "Squirrel Scare", injury.Name)
}
