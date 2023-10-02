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
//
// TODO apply penalty for players out of position
func CalculateTeamControlScore(players []SelectedPlayer) int {
	// group players by position
	var playersByPosition = make(map[PlayerPosition][]SelectedPlayer)
	for _, player := range players {
		playersByPosition[player.Attributes.Position] = append(playersByPosition[player.SelectedPosition], player)
	}

	// calculate the average control score for each position
	var averageControlScoresByPosition = make(map[PlayerPosition]int)
	for position, players := range playersByPosition {
		averageControlScoresByPosition[position] = getAverageControlScore(players)
	}

	overallTeamControlScore := 0
	gkScore := averageControlScoresByPosition[PlayerPositionGoalkeeper] * 5 / 100
	defScore := averageControlScoresByPosition[PlayerPositionDefense] * 15 / 100
	midfieldScore := averageControlScoresByPosition[PlayerPositionMidfield] * 65 / 100
	attackScore := averageControlScoresByPosition[PlayerPositionAttack] * 15 / 100

	return overallTeamControlScore + gkScore + defScore + midfieldScore + attackScore
}

func getAverageControlScore(players []SelectedPlayer) int {
	var totalControlScore int
	for _, player := range players {
		totalControlScore += player.Attributes.GetControlScore()
	}
	return totalControlScore / len(players)
}

// CalculateTeamDefenseScore calculates the overall team defense score for a team.
// It is the sum of the average defense score per position, weighted by position as follows:
//
//	goalkeeper: 35%
//	defense: 40%
//	midfield: 20%
//	attack: 5%
//
// TODO apply penalty for players out of position
func CalculateTeamDefenseScore(players []SelectedPlayer) int {
	// group players by position
	var playersByPosition = make(map[PlayerPosition][]SelectedPlayer)
	for _, player := range players {
		playersByPosition[player.Attributes.Position] = append(playersByPosition[player.SelectedPosition], player)
	}

	// calculate the average control score for each position
	var averageControlScoresByPosition = make(map[PlayerPosition]int)
	for position, players := range playersByPosition {
		averageControlScoresByPosition[position] = getAverageDefenseScore(players)
	}

	gkScore := averageControlScoresByPosition[PlayerPositionGoalkeeper] * 35 / 100
	defScore := averageControlScoresByPosition[PlayerPositionDefense] * 40 / 100
	midfieldScore := averageControlScoresByPosition[PlayerPositionMidfield] * 20 / 100
	attackScore := averageControlScoresByPosition[PlayerPositionAttack] * 5 / 100

	return gkScore + defScore + midfieldScore + attackScore
}

func getAverageDefenseScore(players []SelectedPlayer) int {
	var totalDefenseScore int
	for _, player := range players {
		totalDefenseScore += player.Attributes.GetDefenseScore()
	}
	return totalDefenseScore / len(players)
}
