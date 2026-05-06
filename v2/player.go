package soccer

import "slices"

// PlayerAttributes is the canonical bundle of ratings for a player.
//
// SpeedRating is the single physical rating used by both attacking and
// defensive actions. Earlier v2 also exposed Pace and Recovery as separate
// fields for attack/defense; those were consolidated back into SpeedRating
// to keep the attribute model simple — the game leans arcade-style and the
// extra granularity created clutter without enabling new tactic mechanics.
//
// WorkRate remains separate: it drives midfield / control actions (the
// Press tactic shifts its weight). When zero, EffectiveWorkRate falls back
// to SpeedRating so legacy rosters keep behaving as before.
type PlayerAttributes struct {
	GoalkeeperRating int              `json:"goalkeeper_rating"`
	DefenseRating    int              `json:"defense_rating"`
	SpeedRating      int              `json:"speed_rating"`
	ControlRating    int              `json:"control_rating"`
	AttackRating     int              `json:"attack_rating"`
	AggressionRating int              `json:"aggression_rating"`
	OverallRating    int              `json:"overall_rating"`
	PlayerLevel      PlayerLevel      `json:"player_level"`
	PrimaryPosition  PlayerPosition   `json:"position"`
	Positions        []PlayerPosition `json:"positions"`
	Tag              []string         `json:"tags"`
	BasedOnPlayer    string           `json:"based_on_player"`
	BasedOnPlayerURL string           `json:"based_on_player_url"`

	// WorkRate is the only optional physical attribute kept separate from
	// SpeedRating. Drives midfield / control scoring under the Press tactic.
	// Falls back to SpeedRating when zero.
	WorkRate int `json:"work_rate,omitempty"`

	// Per-chance-type attributes — each one specialises the player on a kind
	// of chance. All optional; when zero, the Effective* accessors fall back
	// to a sensible composite so legacy rosters keep behaving identically.
	//
	// Mapping to FIFA columns (used by the allocation pipeline):
	//   Finishing → attacking_finishing
	//   Heading   → avg(attacking_heading_accuracy, power_jumping)
	//   Technique → avg(skill_curve, skill_fk_accuracy, power_long_shots)
	//   Composure → mentality_composure
	//   Tackling  → avg(defending_standing_tackle, defending_sliding_tackle, mentality_interceptions)
	Finishing int `json:"finishing,omitempty"`
	Heading   int `json:"heading,omitempty"`
	Technique int `json:"technique,omitempty"`
	Composure int `json:"composure,omitempty"`
	Tackling  int `json:"tackling,omitempty"`
}

// EffectiveWorkRate returns WorkRate if set, otherwise SpeedRating.
func (p PlayerAttributes) EffectiveWorkRate() int {
	if p.WorkRate > 0 {
		return p.WorkRate
	}
	return p.SpeedRating
}

// EffectiveFinishing returns Finishing if set, otherwise AttackRating.
// Open-play conversion uses this — a high-AttackRating player who lacks
// explicit finishing data falls back to their composite shooting score.
func (p PlayerAttributes) EffectiveFinishing() int {
	if p.Finishing > 0 {
		return p.Finishing
	}
	return p.AttackRating
}

// EffectiveHeading returns Heading if set, otherwise AttackRating. Drives
// corner / cross conversion; backfill keeps strikers without explicit
// heading data competitive in the air.
func (p PlayerAttributes) EffectiveHeading() int {
	if p.Heading > 0 {
		return p.Heading
	}
	return p.AttackRating
}

// EffectiveTechnique returns Technique if set, otherwise ControlRating.
// Technique drives long-range and free-kick conversion (curve + accuracy +
// long-shot power); ControlRating is the closest legacy proxy.
func (p PlayerAttributes) EffectiveTechnique() int {
	if p.Technique > 0 {
		return p.Technique
	}
	return p.ControlRating
}

// EffectiveComposure returns Composure if set, otherwise ControlRating.
// Composure drives penalty conversion — a clutch finisher.
func (p PlayerAttributes) EffectiveComposure() int {
	if p.Composure > 0 {
		return p.Composure
	}
	return p.ControlRating
}

// EffectiveTackling returns Tackling if set, otherwise DefenseRating.
// Tackling supplements DefenseRating for outfield defense — a player who
// can press, intercept, and dispossess. Goalkeepers don't use it.
func (p PlayerAttributes) EffectiveTackling() int {
	if p.Tackling > 0 {
		return p.Tackling
	}
	return p.DefenseRating
}

// IsInjuryProne reports whether a player carries the InjuryProne tag.
func (p PlayerAttributes) IsInjuryProne() bool {
	for _, t := range p.Tag {
		if t == string(PlayerTagInjuryProne) {
			return true
		}
	}
	return false
}

type SelectedPlayer struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Attributes       PlayerAttributes `json:"attributes"`
	SelectedPosition PlayerPosition   `json:"position"`
	Injury           *InjuryEvent     `json:"injury,omitempty"`
	// Role is optional. The zero value (PlayerRoleNone) means no special
	// contribution — the engine treats this player the same as any other.
	Role PlayerRole `json:"role,omitempty"`
}

// IsOutOfPosition reports whether SelectedPosition isn't one the player can
// actually play. Players with PlayerPositionAny in their listed positions
// (or as their primary) can play anywhere.
func (p SelectedPlayer) IsOutOfPosition() bool {
	if p.Attributes.PrimaryPosition == PlayerPositionAny ||
		slices.Contains(p.Attributes.Positions, PlayerPositionAny) {
		return false
	}
	return !slices.Contains(p.PlayablePositions(), p.SelectedPosition)
}

// PlayablePositions returns the union of Positions and PrimaryPosition.
// Some legacy data only sets PrimaryPosition; combining both means newer
// rosters with explicit Positions arrays still work.
func (p SelectedPlayer) PlayablePositions() []PlayerPosition {
	out := make([]PlayerPosition, 0, len(p.Attributes.Positions)+1)
	out = append(out, p.Attributes.Positions...)
	if p.Attributes.PrimaryPosition != "" {
		out = append(out, p.Attributes.PrimaryPosition)
	}
	return out
}
