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
		{"second_use_no_decay", 1},
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

			// Multiplier = max(DRMinMultiplier, DRDecayPerApplication^apps)
			m := math.Pow(soccer.DRDecayPerApplication, float64(tc.apps))
			if m < soccer.DRMinMultiplier {
				m = soccer.DRMinMultiplier
			}
			var expected float64
			if tc.apps <= 1 {
				expected = base
			} else if base >= 1.0 {
				expected = 1.0 + (base-1.0)*m
			} else {
				expected = base * m
			}

			got := b.GetBoost(rGot)
			assert.InDelta(t, expected, got, 1e-12)
		})
	}
}

func TestDiminishingMultiplier(t *testing.T) {
	type tc struct {
		name     string
		apps     int
		expected float64
		floored  bool
	}

	cases := []tc{
		{name: "apps_0_is_1", apps: 0, expected: 1.0},
		{name: "apps_1_is_decay", apps: 1, expected: soccer.DRDecayPerApplication},
		{name: "apps_2_is_decay_sq", apps: 2, expected: math.Pow(soccer.DRDecayPerApplication, 2)},
		{name: "apps_5_above_floor", apps: 5, expected: math.Pow(soccer.DRDecayPerApplication, 5)},
		{name: "apps_38_stuck_at_floor", apps: 38, expected: soccer.DRMinMultiplier, floored: true},
	}

	const eps = 1e-12

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := soccer.DiminishingMultiplier(c.apps)

			// If expected is floored, ensure raw value is indeed below the floor
			if c.floored {
				raw := math.Pow(soccer.DRDecayPerApplication, float64(c.apps))
				if raw >= soccer.DRMinMultiplier {
					t.Fatalf("expected raw multiplier to be below floor for apps=%d; raw=%v floor=%v", c.apps, raw, soccer.DRMinMultiplier)
				}
			}

			if math.Abs(got-c.expected) > eps {
				t.Fatalf("DiminishingMultiplier(%d) = %v, want %v", c.apps, got, c.expected)
			}
		})
	}
}

// TestGetBoost_SampleOutputs logs a small, deterministic table of base, multiplier and effective boost
// for a few application counts and seeds. Run with `go test -run TestGetBoost_SampleOutputs -v` to see the table.
func TestGetBoost_SampleOutputs(t *testing.T) {
	posMin, posMax := 1.05, 1.15
	negMin, negMax := 0.90, 0.95
	appsCases := []int{0, 1, 5, 20, 50}
	seeds := []int64{1, 42, 123456789}

	t.Log("Positive boost samples (base in [1.05,1.15])")
	for _, apps := range appsCases {
		b := soccer.Boost{MinBoost: posMin, MaxBoost: posMax, Applications: apps}
		for _, seed := range seeds {
			rExp := rand.New(rand.NewSource(seed))
			rGot := rand.New(rand.NewSource(seed))
			base := posMin + rExp.Float64()*(posMax-posMin)
			m := math.Pow(soccer.DRDecayPerApplication, float64(apps))
			if m < soccer.DRMinMultiplier {
				m = soccer.DRMinMultiplier
			}
			var expected float64
			if apps <= 1 {
				expected = base
			} else {
				expected = 1.0 + (base-1.0)*m
			}
			got := b.GetBoost(rGot)
			t.Logf("apps=%2d seed=%9d base=%0.6f m=%0.6f eff_expected=%0.6f eff_get=%0.6f", apps, seed, base, m, expected, got)
			assert.InDelta(t, expected, got, 1e-12)
		}
	}

	t.Log("Debuff samples (base in [0.90,0.95])")
	for _, apps := range appsCases {
		b := soccer.Boost{MinBoost: negMin, MaxBoost: negMax, Applications: apps}
		for _, seed := range seeds {
			rExp := rand.New(rand.NewSource(seed))
			rGot := rand.New(rand.NewSource(seed))
			base := negMin + rExp.Float64()*(negMax-negMin)
			m := math.Pow(soccer.DRDecayPerApplication, float64(apps))
			if m < soccer.DRMinMultiplier {
				m = soccer.DRMinMultiplier
			}
			var expected float64
			if apps <= 1 {
				expected = base
			} else {
				expected = base * m
			}
			got := b.GetBoost(rGot)
			t.Logf("apps=%2d seed=%9d base=%0.6f m=%0.6f eff_expected=%0.6f eff_get=%0.6f", apps, seed, base, m, expected, got)
			assert.InDelta(t, expected, got, 1e-12)
		}
	}
}
