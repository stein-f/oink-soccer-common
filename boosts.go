package soccer

import (
	"math"
	"math/rand"
)

const (
	// DRDecayPerApplication defines the decay factor, which multiplies the effect by this factor.
	// Example: apps=0 => 1.0x, apps=1 => 0.85x, apps=2 => 0.85^2, ...
	DRDecayPerApplication = 0.85
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

func diminishingMultiplier(apps int) float64 {
	m := math.Pow(DRDecayPerApplication, float64(apps))
	if m < DRMinMultiplier {
		return DRMinMultiplier
	}
	return m
}

func (b Boost) GetBoost(source *rand.Rand) float64 {
	base := b.MinBoost + source.Float64()*(b.MaxBoost-b.MinBoost)
	return base * diminishingMultiplier(b.Applications)
}
