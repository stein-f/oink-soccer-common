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
		{40, 6},
		{45, 9},
		{50, 12},
		{55, 16},
		{60, 21},
		{65, 27},
		{70, 34},
		{75, 42},
		{80, 51},
		{85, 61},
		{90, 72},
		{95, 85},
		{100, 100},
	}

	for _, tc := range testCases {
		scaled := soccer.ScalingFunction(tc.originalRating)
		assert.Equal(t, tc.expectedScaled, math.Floor(scaled))
	}
}

func TestNormalizeRating(t *testing.T) {
	tests := map[string]struct {
		gotSumOfRatings      int
		gotMaxRatings        int
		wantNormalizedRating int
	}{
		"handles zero rating": {
			gotSumOfRatings:      0,
			gotMaxRatings:        0,
			wantNormalizedRating: 0,
		},
		"should be 50 when sum is half of max": {
			gotSumOfRatings:      80,
			gotMaxRatings:        160,
			wantNormalizedRating: 50,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			normalizeRating := soccer.NormalizeRating(test.gotSumOfRatings, test.gotMaxRatings)

			assert.Equal(t, test.wantNormalizedRating, normalizeRating)
		})
	}
}
