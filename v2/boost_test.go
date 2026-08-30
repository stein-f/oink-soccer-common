package soccer

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
)

// A fixed-value boost isolates the decay multiplier from the [min,max] roll.
func fixedBoost(v float64, charge float64) Boost {
	return Boost{BoostType: BoostTypeTeam, MinBoost: v, MaxBoost: v, DecayCharge: charge}
}

func TestRollBoost_DecaysExcessByCharge(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	tests := []struct {
		name   string
		charge float64
	}{
		{name: "no charge", charge: 0},
		{name: "one unit", charge: 1},
		{name: "fractional charge", charge: 0.6},
		{name: "weighted charge", charge: 3.33},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := math.Pow(tuning.BoostDecay, tc.charge)
			if m < tuning.BoostMinMultiplier {
				m = tuning.BoostMinMultiplier
			}
			want := 1.0 + 0.10*m

			got := rollBoost(r, fixedBoost(1.10, tc.charge))
			if math.Abs(got-want) > 1e-9 {
				t.Fatalf("charge %v: got %v, want %v", tc.charge, got, want)
			}
		})
	}
}

// The point of weighting charge by item strength: every item delivers the
// same boost per unit of charge it burns, so no tier is stronger-for-cheaper.
func TestRollBoost_DeliveredBoostPerChargeIsFlatAcrossTiers(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	// (average boost, decay weight) for the three truffle tiers, as the
	// caller weights them. Charge is normalised on the mid tier.
	tiers := []struct {
		name   string
		boost  float64
		weight float64
	}{
		{name: "magic", boost: 1.03, weight: 0.6},
		{name: "golden", boost: 1.05, weight: 1.0},
		{name: "iridium", boost: 1.10, weight: 2.0},
	}

	// Charge accrued from one boosted day of each tier, at a fresh season.
	var ratios []float64
	for _, tier := range tiers {
		delivered := rollBoost(r, fixedBoost(tier.boost, 0)) - 1.0
		ratios = append(ratios, delivered/tier.weight)
	}

	for i := 1; i < len(ratios); i++ {
		if math.Abs(ratios[i]-ratios[0]) > 0.002 {
			t.Fatalf("%s delivers %.4f boost per unit of charge, %s delivers %.4f — tiers must be within 0.002",
				tiers[i].name, ratios[i], tiers[0].name, ratios[0])
		}
	}
}

func TestRollBoost_ClampsToMinMultiplier(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	// A charge high enough that raw decay would fall under the floor.
	charge := math.Log(tuning.BoostMinMultiplier)/math.Log(tuning.BoostDecay) + 10
	got := rollBoost(r, fixedBoost(1.10, charge))
	want := 1.0 + 0.10*tuning.BoostMinMultiplier
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Decay shrinks a debuff toward neutral just like a boost. The old formula
// multiplied a debuff whole (0.9 * 0.94 = 0.846), pushing it further from
// 1.0 — decay made a struggling team worse, which is the same inversion bug
// that made positive boosts into nerfs.
func TestRollBoost_DebuffDecaysTowardNeutral(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	got := rollBoost(r, fixedBoost(0.9, 2))
	want := 1.0 - 0.10*math.Pow(tuning.BoostDecay, 2)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got <= 0.9 {
		t.Fatalf("decay made the debuff harsher: %v <= %v", got, 0.9)
	}
}

// The regression guard for the bug that had decay switched off in prod: a
// boost the player paid for must never come out below neutral, at any charge.
func TestRollBoost_NeverInvertsAPositiveBoost(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	// The weakest boost in the shop (+1%) against charges far past the floor.
	for _, charge := range []float64{0.6, 1, 2, 5, 12, 40, 200, 10000} {
		got := rollBoost(r, fixedBoost(1.01, charge))
		if got < 1.0 {
			t.Fatalf("charge %v turned a +1%% boost into a %v nerf", charge, got)
		}
	}
}

// A Boost with no range set must leave the team score alone rather than
// multiplying it by zero.
func TestRollBoost_UnpopulatedBoostIsNeutral(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	if got := rollBoost(r, Boost{BoostType: BoostTypeTeam}); got != 1.0 {
		t.Fatalf("got %v, want 1.0", got)
	}
}
