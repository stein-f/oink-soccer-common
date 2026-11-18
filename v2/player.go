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

	// Player-on-player attributes, written from season 19 (FC 27 dataset) for
	// the step engine, which decides a match by one player meeting another.
	// All optional: when zero, the Effective* accessors fall back to the core
	// rating that used to stand in for them, so players from earlier seasons
	// and hand-built lineups keep working. The classic model reads none of
	// these.
	//
	// Mapping to dataset columns (allocation/fifacsv.StepSource):
	//   Dribbling     → avg(dribbling, skill_dribbling, movement_agility, movement_balance)
	//   Passing       → avg(passing, attacking_short_passing, mentality_vision, skill_long_passing)
	//   Crossing      → avg(attacking_crossing, skill_curve)
	//   Pace          → avg(movement_acceleration, movement_sprint_speed)
	//   Strength      → avg(power_strength, power_jumping)
	//   Interceptions → mentality_interceptions
	//   Positioning   → mentality_positioning (finding space going forward)
	//   Marking       → defending_marking_awareness (holding position without the ball)
	//   ShotPower     → avg(power_shot_power, power_long_shots)
	//   Reflexes      → avg(goalkeeping_reflexes, goalkeeping_diving)
	//   Handling      → goalkeeping_handling
	//   OneOnOne      → goalkeeping_positioning twice over goalkeeping_speed
	Dribbling     int `json:"dribbling,omitempty"`
	Passing       int `json:"passing,omitempty"`
	Crossing      int `json:"crossing,omitempty"`
	Pace          int `json:"pace,omitempty"`
	Strength      int `json:"strength,omitempty"`
	Interceptions int `json:"interceptions,omitempty"`
	Positioning   int `json:"positioning,omitempty"`
	Marking       int `json:"marking,omitempty"`
	ShotPower     int `json:"shot_power,omitempty"`
	Reflexes      int `json:"gk_reflexes,omitempty"`
	Handling      int `json:"gk_handling,omitempty"`
	OneOnOne      int `json:"gk_one_on_one,omitempty"`
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

// EffectiveDribbling returns Dribbling if set, otherwise ControlRating.
// Taking a defender on.
func (p PlayerAttributes) EffectiveDribbling() int {
	if p.Dribbling > 0 {
		return p.Dribbling
	}
	return p.ControlRating
}

// EffectivePassing returns Passing if set, otherwise ControlRating.
func (p PlayerAttributes) EffectivePassing() int {
	if p.Passing > 0 {
		return p.Passing
	}
	return p.ControlRating
}

// EffectiveCrossing returns Crossing if set, otherwise EffectiveTechnique
// (which itself falls back to ControlRating).
func (p PlayerAttributes) EffectiveCrossing() int {
	if p.Crossing > 0 {
		return p.Crossing
	}
	return p.EffectiveTechnique()
}

// EffectivePace returns Pace if set, otherwise SpeedRating. Pace is raw
// running speed for a race; SpeedRating stays the classic model's single
// physical rating.
func (p PlayerAttributes) EffectivePace() int {
	if p.Pace > 0 {
		return p.Pace
	}
	return p.SpeedRating
}

// EffectiveStrength returns Strength if set. Otherwise it is read from the
// rating for the work the player does: DefenseRating for a defender,
// AttackRating for an attacker, and the mean of the two for anyone else.
func (p PlayerAttributes) EffectiveStrength() int {
	if p.Strength > 0 {
		return p.Strength
	}
	switch p.PrimaryPosition {
	case PlayerPositionDefense, PlayerPositionGoalkeeper:
		return p.DefenseRating
	case PlayerPositionAttack:
		return p.AttackRating
	}
	return (p.DefenseRating + p.AttackRating) / 2
}

// EffectiveInterceptions returns Interceptions if set, otherwise
// DefenseRating. Cutting out a pass.
func (p PlayerAttributes) EffectiveInterceptions() int {
	if p.Interceptions > 0 {
		return p.Interceptions
	}
	return p.DefenseRating
}

// EffectivePositioning returns Positioning if set, otherwise AttackRating.
// Finding space going forward: the run in behind, the spot in the box.
func (p PlayerAttributes) EffectivePositioning() int {
	if p.Positioning > 0 {
		return p.Positioning
	}
	return p.AttackRating
}

// EffectiveMarking returns Marking if set, otherwise DefenseRating. Holding
// position without the ball: tracking a runner, getting in front of a shot.
func (p PlayerAttributes) EffectiveMarking() int {
	if p.Marking > 0 {
		return p.Marking
	}
	return p.DefenseRating
}

// EffectiveShotPower returns ShotPower if set, otherwise EffectiveTechnique.
func (p PlayerAttributes) EffectiveShotPower() int {
	if p.ShotPower > 0 {
		return p.ShotPower
	}
	return p.EffectiveTechnique()
}

// EffectiveReflexes returns Reflexes if set, otherwise GoalkeeperRating.
// Shot-stopping.
func (p PlayerAttributes) EffectiveReflexes() int {
	if p.Reflexes > 0 {
		return p.Reflexes
	}
	return p.GoalkeeperRating
}

// EffectiveHandling returns Handling if set, otherwise GoalkeeperRating.
// Holding a shot rather than parrying it, and claiming crosses.
func (p PlayerAttributes) EffectiveHandling() int {
	if p.Handling > 0 {
		return p.Handling
	}
	return p.GoalkeeperRating
}

// EffectiveOneOnOne returns OneOnOne if set, otherwise GoalkeeperRating.
// Coming off the line to a forward who is through.
func (p PlayerAttributes) EffectiveOneOnOne() int {
	if p.OneOnOne > 0 {
		return p.OneOnOne
	}
	return p.GoalkeeperRating
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
