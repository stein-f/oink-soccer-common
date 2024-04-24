package soccer

import "time"

type Injury struct {
	Description      string    `json:"description"`
	Expires          time.Time `json:"expires"`
	PlayerID         string    `json:"player_id"`
	SpeedReduction   float64   `json:"speed_reduction"`   // 0.8 = 20% reduction
	ControlReduction float64   `json:"control_reduction"` // 0.8 = 20% reduction
}
