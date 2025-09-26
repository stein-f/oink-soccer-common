package soccer

import (
	"math"
	"math/rand"
)

const (
	// DRDecayPerApplication defines the decay factor, which multiplies the effect by this factor.
	// Example: apps=0 => 1.0x, apps=1 => 0.98x, apps=2 => 0.98^2, ...
	DRDecayPerApplication = 0.97
	// DRMinMultiplier is a floor so boosts never become useless
	DRMinMultiplier = 0.35
)

type Boost struct {
	BoostType     BoostType      `json:"boost_type"`
	BoostPosition PlayerPosition `json:"boost_position"`
	MinBoost      float64        `json:"min_boost"`
	MaxBoost      float64        `json:"max_boost"`
	Note          string         `json:"note"`
	Applications  int            `json:"applications"`
}

func DiminishingMultiplier(apps int) float64 {
	m := math.Pow(DRDecayPerApplication, float64(apps))
	if m < DRMinMultiplier {
		return DRMinMultiplier
	}
	return m
}

func (b Boost) GetBoost(source *rand.Rand) float64 {
	base := b.MinBoost + source.Float64()*(b.MaxBoost-b.MinBoost)

	// Only apply diminishing when applications are more than 1.
	if b.Applications <= 1 {
		return base
	}

	m := DiminishingMultiplier(b.Applications)

	// Apply diminishing returns to the excess over 1.0 only so positive boosts never become nerfs.
	if base >= 1.0 {
		return 1.0 + (base-1.0)*m
	}
	// For debuffs (base < 1.0), keep existing behavior of scaling the whole value.
	return base * m
}
