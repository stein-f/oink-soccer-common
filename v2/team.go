package soccer

type Team struct {
	ID         string        `json:"id"`
	CustomName string        `json:"custom_name"`
	Formation  FormationType `json:"formation"`
	// Tactics is optional — zero value is "neutral", which preserves v1
	// behaviour for lineups that don't populate it.
	Tactics Tactics `json:"tactics,omitempty"`
}

type GameLineup struct {
	Team       Team             `json:"team"`
	Players    []SelectedPlayer `json:"players"`
	ItemBoosts []Boost          `json:"item_boosts"`
}
