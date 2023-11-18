package soccer_test

import (
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

// TestScaleRating tests the scaleRating function with various input values.
func TestScaleRating(t *testing.T) {
	testCases := []struct {
		originalRating float64
		expectedScaled float64
	}{
		{40, 3},
		{45, 5},
		{50, 8},
		{55, 12},
		{60, 17},
		{65, 23},
		{70, 32},
		{75, 42},
		{80, 54},
		{85, 69},
		{90, 87},
		{91, 91},
		{92, 95},
		{93, 100},
	}

	for _, tc := range testCases {
		scaled := soccer.ScalingFunction(tc.originalRating)
		assert.Equal(t, tc.expectedScaled, math.Floor(scaled))
	}
}
