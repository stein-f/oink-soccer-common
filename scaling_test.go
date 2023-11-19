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
		{40, 7},
		{45, 10},
		{50, 15},
		{55, 20},
		{60, 26},
		{65, 33},
		{70, 41},
		{75, 50},
		{80, 61},
		{85, 73},
		{90, 87},
		{91, 90},
		{92, 93},
		{93, 96},
		{100, 100},
	}

	for _, tc := range testCases {
		scaled := soccer.ScalingFunction(tc.originalRating)
		assert.Equal(t, tc.expectedScaled, math.Floor(scaled))
	}
}
