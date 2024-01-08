package soccer

import (
	"math/rand"
	"time"
)

type Boost struct {
	BoostType     BoostType      `json:"boost_type"`
	BoostPosition PlayerPosition `json:"boost_position"`
	MinBoost      float64        `json:"min_boost"`
	MaxBoost      float64        `json:"max_boost"`
}

func (b Boost) GetBoost() float64 {
	src := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(src)
	randomFloat := rnd.Float64()
	return b.MinBoost + randomFloat*(b.MaxBoost-b.MinBoost)
}
