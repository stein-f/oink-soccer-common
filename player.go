package soccer

import "math"

type PlayerAttributes struct {
	GoalkeeperRating int            `json:"goalkeeper_rating"`
	DefenseRating    int            `json:"defense_rating"`
	SpeedRating      int            `json:"speed_rating"`
	ControlRating    int            `json:"control_rating"`
	AttackRating     int            `json:"attack_rating"`
	OverallRating    int            `json:"overall_rating"`
	PlayerLevel      PlayerLevel    `json:"player_level"`
	Position         PlayerPosition `json:"position"`
	Tag              []string       `json:"tags"`
	BasedOnPlayer    string         `json:"based_on_player"`
	BasedOnPlayerURL string         `json:"based_on_player_url"`
}

func (p PlayerAttributes) IsInjuryProne() bool {
	for _, tag := range p.Tag {
		if tag == string(PlayerTagInjuryProne) {
			return true
		}
	}
	return false
}

// GetOverallRating returns the overall rating for a player based on their position
func (p PlayerAttributes) GetOverallRating() int {
	if p.Position == PlayerPositionGoalkeeper {
		return (p.GoalkeeperRating*5 + p.SpeedRating) / 6
	}
	if p.Position == PlayerPositionDefense {
		return (p.DefenseRating*5 + p.SpeedRating) / 6
	}
	if p.Position == PlayerPositionMidfield {
		return (p.ControlRating*4 + p.SpeedRating) / 5
	}
	if p.Position == PlayerPositionAttack {
		return (p.AttackRating*3 + p.SpeedRating) / 4
	}
	return p.OverallRating
}

// GetControlScore returns the control score for a gotPlayer
// It is calculated using the control and speed rating where control is weighted 3x more than speed
// controlScore = (controlRating * 4 + speedRating) / 5
func (p PlayerAttributes) GetControlScore() float64 {
	return math.Round(float64(p.ControlRating*4+p.SpeedRating) / 5)
}

// GetAttackScore returns the attack score for a gotPlayer
// It is calculated using the attack and speed rating where attack is weighted 3x more than speed
// attackScore = (attackRating * 3 + speedRating) / 4
func (p PlayerAttributes) GetAttackScore() float64 {
	return math.Round(float64(p.AttackRating*3+p.SpeedRating) / 4)
}

// GetDefenseScore returns the defense score for a gotPlayer
// It is calculated using the defense and speed rating where defense is weighted 3x more than speed
// defenseScore = (defenseRating * 5 + speedRating) / 6
func (p PlayerAttributes) GetDefenseScore() float64 {
	rating := p.DefenseRating
	if p.Position == PlayerPositionGoalkeeper {
		rating = p.GoalkeeperRating
	}
	return math.Round(float64(rating*5+p.SpeedRating) / 6)
}
