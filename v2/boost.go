package soccer

import "github.com/stein-f/oink-soccer-common/v2/internal/tuning"

// DRDecayPerApplication is the decay multiplier applied per unit of decay
// charge a team has accrued. lost-pigs/api/soccer.go uses it to compute how
// much a boost has faded. Aliased to the internal tuning constant so the
// value stays in lockstep with the engine.
const DRDecayPerApplication = tuning.BoostDecay

// DRMinMultiplier is the floor on boost decay — however much a team has
// boosted, its items still land at this fraction of their rolled strength.
// Exported so the effectiveness UI shows the floor the engine actually
// applies rather than a raw decay curve that runs away below it.
const DRMinMultiplier = tuning.BoostMinMultiplier

type Boost struct {
	BoostType     BoostType      `json:"boost_type"`
	BoostPosition PlayerPosition `json:"boost_position"`
	MinBoost      float64        `json:"min_boost"`
	MaxBoost      float64        `json:"max_boost"`
	Note          string         `json:"note"`
	// DecayCharge is how much of the team's season decay budget was already
	// spent when this boost is rolled. It replaces the old integer
	// Applications count, which charged one unit per boosted day regardless
	// of item strength — that made the strongest item the cheapest to the
	// budget. The caller weights each boosted day by item strength (see
	// lost-pigs decay_weight metadata) so delivered boost per unit of charge
	// is constant across item tiers. 0 means no decay.
	DecayCharge float64 `json:"decay_charge"`
}
