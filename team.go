package soccer

type Team struct {
	ID         string        `json:"id"`
	CustomName string        `json:"custom_name"`
	Formation  FormationType `json:"formation"`
}

// CalculateTeamControlScore calculates the overall team control score for a team.
// It is the sum of the average control score per position, weighted by position as follows:
//
//	goalkeeper: 5%
//	defense: 15%
//	midfield: 65%
//	attack: 15%
func CalculateTeamControlScore(players []SelectedPlayer) float64 {
	// group players by position
	var playersByPosition = make(map[PlayerPosition][]SelectedPlayer)
	for _, player := range players {
		playersByPosition[player.Attributes.Position] = append(playersByPosition[player.SelectedPosition], player)
	}

	// calculate the average control score for each position
	var averageControlScoresByPosition = make(map[PlayerPosition]float64)
	for position, players := range playersByPosition {
		averageControlScoresByPosition[position] = getAverageControlScore(players)
	}

	gkScore := averageControlScoresByPosition[PlayerPositionGoalkeeper] * 5 / 100
	defScore := averageControlScoresByPosition[PlayerPositionDefense] * 15 / 100
	midfieldScore := averageControlScoresByPosition[PlayerPositionMidfield] * 65 / 100
	attackScore := averageControlScoresByPosition[PlayerPositionAttack] * 15 / 100

	return gkScore + defScore + midfieldScore + attackScore
}

func getAverageControlScore(players []SelectedPlayer) float64 {
	var totalControlScore float64
	for _, player := range players {
		totalControlScore += player.GetControlScore()
	}
	return totalControlScore / float64(len(players))
}

// CalculateTeamDefenseScore calculates the overall team defense score for a team.
// It is the sum of the average defense score per position, weighted by position as follows:
//
//	goalkeeper: 35%
//	defense: 40%
//	midfield: 20%
//	attack: 5%
func CalculateTeamDefenseScore(players []SelectedPlayer) float64 {
	// group players by position
	var playersByPosition = make(map[PlayerPosition][]SelectedPlayer)
	for _, player := range players {
		playersByPosition[player.SelectedPosition] = append(playersByPosition[player.SelectedPosition], player)
	}

	// calculate the average control score for each position
	averageControlScoresByPosition := make(map[PlayerPosition]float64)
	for position, players := range playersByPosition {
		averageControlScoresByPosition[position] = getAverageDefenseScore(players)
	}

	gkScore := averageControlScoresByPosition[PlayerPositionGoalkeeper] * 35 / 100
	defScore := averageControlScoresByPosition[PlayerPositionDefense] * 40 / 100
	midfieldScore := averageControlScoresByPosition[PlayerPositionMidfield] * 20 / 100
	attackScore := averageControlScoresByPosition[PlayerPositionAttack] * 5 / 100

	return gkScore + defScore + midfieldScore + attackScore
}

func getAverageDefenseScore(players []SelectedPlayer) float64 {
	var totalDefenseScore float64
	for _, player := range players {
		totalDefenseScore += player.GetDefenseScore()
	}
	return totalDefenseScore / float64(len(players))
}
