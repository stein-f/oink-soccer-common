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

// GetControlScore returns the control score for a player
// It is calculated using the control and speed rating where control is weighted 3x more than speed
// controlScore = (controlRating * 3 + speedRating) / 4
func (p PlayerAttributes) GetControlScore() float64 {
	return getRating(p.ControlRating, p.SpeedRating)
}

// GetAttackScore returns the attack score for a player
// It is calculated using the attack and speed rating where attack is weighted 3x more than speed
// attackScore = (attackRating * 3 + speedRating) / 4
func (p PlayerAttributes) GetAttackScore() float64 {
	return getRating(p.AttackRating, p.SpeedRating)
}

// GetDefenseScore returns the defense score for a player
// It is calculated using the defense and speed rating where defense is weighted 3x more than speed
// defenseScore = (defenseRating * 3 + speedRating) / 4
func (p PlayerAttributes) GetDefenseScore() float64 {
	rating := p.DefenseRating
	if p.Position == PlayerPositionGoalkeeper {
		rating = p.GoalkeeperRating
	}
	return getRating(rating, p.SpeedRating)
}

func getRating(rating, speedRating int) float64 {
	return math.Round(float64(rating*3+speedRating) / 4)
}
