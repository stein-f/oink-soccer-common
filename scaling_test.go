package soccer_test

import (
	"math"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
)

// TestScaleRating tests the scaleRating function with various input values.
func TestScaleRating(t *testing.T) {
	testCases := []struct {
		originalRating float64
		expectedScaled float64
	}{
		{1, 5},
		{50, 5},
		{65, 5},
		{70, 6},
		{75, 11},
		{80, 20},
		{85, 36},
		{90, 67},
		{91, 75},
		{92, 85},
		{93, 96},
		{100, 100},
	}

	for _, tc := range testCases {
		scaled := soccer.ScalingFunction(tc.originalRating)
		assert.Equal(t, tc.expectedScaled, math.Floor(scaled))
	}
}
