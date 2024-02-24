package soccer

import (
	"math/rand"
)

type Boost struct {
	BoostType     BoostType      `json:"boost_type"`
	BoostPosition PlayerPosition `json:"boost_position"`
	MinBoost      float64        `json:"min_boost"`
	MaxBoost      float64        `json:"max_boost"`
	Note          float64        `json:"note"`
}

func (b Boost) GetBoost(source *rand.Rand) float64 {
	randomFloat := source.Float64()
	return b.MinBoost + randomFloat*(b.MaxBoost-b.MinBoost)
}
