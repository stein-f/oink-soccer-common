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
		{40, 1},
		{45, 2},
		{50, 4},
		{55, 7},
		{60, 11},
		{65, 16},
		{70, 24},
		{75, 34},
		{80, 47},
		{85, 63},
		{90, 84},
		{91, 89},
		{92, 94},
		{93, 100},
		{100, 100},
	}

	for _, tc := range testCases {
		scaled := soccer.ScalingFunction(tc.originalRating)
		assert.Equal(t, tc.expectedScaled, math.Floor(scaled))
	}
}
