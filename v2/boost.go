package soccer

import "github.com/stein-f/oink-soccer-common/v2/internal/tuning"

// DRDecayPerApplication is the per-application decay multiplier for stacked
// boosts. lost-pigs/api/soccer.go uses it to compute how much a boost has
// faded based on how long it's been active. Aliased to the internal tuning
// constant so the value stays in lockstep with the engine.
const DRDecayPerApplication = tuning.BoostDecay

type Boost struct {
	BoostType     BoostType      `json:"boost_type"`
	BoostPosition PlayerPosition `json:"boost_position"`
	MinBoost      float64        `json:"min_boost"`
	MaxBoost      float64        `json:"max_boost"`
	Note          string         `json:"note"`
	Applications  int            `json:"applications"`
}
