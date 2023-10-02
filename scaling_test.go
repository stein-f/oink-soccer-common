package soccer_test

import (
	"fmt"
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Scaling function:
// Rating In: 40,  Rating Out: 6
// Rating In: 45,  Rating Out: 8
// Rating In: 50,  Rating Out: 10
// Rating In: 55,  Rating Out: 12
// Rating In: 60,  Rating Out: 15
// Rating In: 65,  Rating Out: 18
// Rating In: 70,  Rating Out: 22
// Rating In: 75,  Rating Out: 27
// Rating In: 80,  Rating Out: 33
// Rating In: 85,  Rating Out: 39
// Rating In: 90,  Rating Out: 47
// Rating In: 95,  Rating Out: 55
// Rating In: 100, Rating Out: 63
func TestScalingFunction(t *testing.T) {
	for i := 40; i <= 100; i++ {
		if i%5 == 0 {
			fmt.Println(fmt.Sprintf("Rating In: %d, Rating Out: %d", i, soccer.ScalingFunction(i)))
		}
	}

	assert.Equal(t, 6, soccer.ScalingFunction(40))
	assert.Equal(t, 8, soccer.ScalingFunction(45))
	assert.Equal(t, 10, soccer.ScalingFunction(50))
	assert.Equal(t, 12, soccer.ScalingFunction(55))
	assert.Equal(t, 15, soccer.ScalingFunction(60))
	assert.Equal(t, 18, soccer.ScalingFunction(65))
	assert.Equal(t, 22, soccer.ScalingFunction(70))
	assert.Equal(t, 27, soccer.ScalingFunction(75))
	assert.Equal(t, 33, soccer.ScalingFunction(80))
	assert.Equal(t, 39, soccer.ScalingFunction(85))
	assert.Equal(t, 47, soccer.ScalingFunction(90))
	assert.Equal(t, 55, soccer.ScalingFunction(95))
	assert.Equal(t, 63, soccer.ScalingFunction(100))
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
