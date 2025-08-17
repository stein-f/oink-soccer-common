package soccer_test

import (
	"github.com/stretchr/testify/assert"
	"math"
	"math/rand"
	"testing"
	"time"

	soccer "github.com/stein-f/oink-soccer-common"
)

func TestGetBoost(t *testing.T) {
	tests := []struct {
		name  string
		boost soccer.Boost
	}{
		{"in range", soccer.Boost{MinBoost: 5, MaxBoost: 10}},
		{"handles min 0", soccer.Boost{MinBoost: 0, MaxBoost: 1}},
		{"handles negative", soccer.Boost{MinBoost: -10, MaxBoost: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.boost.GetBoost(rand.New(rand.NewSource(time.Now().UnixNano())))
			if got < tt.boost.MinBoost || got > tt.boost.MaxBoost {
				t.Errorf("GetBoost() = %v, want range [%v, %v]", got, tt.boost.MinBoost, tt.boost.MaxBoost)
			}
		})
	}
}

func TestGetBoost_Applications(t *testing.T) {
	minVal, maxVal := 1.0, 3.0

	cases := []struct {
		name string
		apps int
	}{
		{"first_use_no_penalty", 0},
		{"second_use_decay", 1},
		{"deep_decay_clamped_to_floor", 50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := soccer.Boost{
				MinBoost:     minVal,
				MaxBoost:     maxVal,
				Applications: tc.apps,
			}

			// Use two RNGs with the same seed so we can compute the expected base roll
			seed := int64(123456789)
			rExp := rand.New(rand.NewSource(seed))
			rGot := rand.New(rand.NewSource(seed))

			base := minVal + rExp.Float64()*(maxVal-minVal)

			// Multiplier = maxVal(DRMinMultiplier, DRDecayPerApplication^apps)
			m := math.Pow(soccer.DRDecayPerApplication, float64(tc.apps))
			if m < soccer.DRMinMultiplier {
				m = soccer.DRMinMultiplier
			}
			expected := base * m

			got := b.GetBoost(rGot)
			assert.InDelta(t, expected, got, 1e-12)
		})
	}
}
