package soccer_test

import (
	"math"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
)

func TestScaleRating(t *testing.T) {
	testCases := []struct {
		originalRating float64
		expectedScaled float64
	}{
		{1, 1},
		{50, 5},
		{65, 18},
		{70, 26},
		{75, 35},
		{80, 50},
		{85, 64},
		{90, 80},
		{90.5, 82},
		{95, 99},
		{100, 100},
	}
	for _, tc := range testCases {
		scaled := soccer.ScalingFunction(tc.originalRating)
		assert.Equal(t, tc.expectedScaled, math.Floor(scaled))
	}
}
