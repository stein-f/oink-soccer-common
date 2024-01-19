package soccer

import "math/rand"

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
func CalculateTeamControlScore(source *rand.Rand, lineup GameLineup) float64 {
	// group players by position
	var playersByPosition = make(map[PlayerPosition][]SelectedPlayer)
	for _, player := range lineup.Players {
		playersByPosition[player.SelectedPosition] = append(playersByPosition[player.SelectedPosition], player)
	}

	// calculate the average control score for each position
	var averageControlScoresByPosition = make(map[PlayerPosition]float64)
	boost := getPositionItemBoost(source, lineup.ItemBoosts, PlayerPositionMidfield)
	for position, players := range playersByPosition {
		averageControlScoresByPosition[position] = getAverageControlScore(boost, players)
	}

	gkScore := averageControlScoresByPosition[PlayerPositionGoalkeeper] * 5 / 100
	defScore := averageControlScoresByPosition[PlayerPositionDefense] * 15 / 100
	midfieldScore := averageControlScoresByPosition[PlayerPositionMidfield] * 65 / 100
	attackScore := averageControlScoresByPosition[PlayerPositionAttack] * 15 / 100

	controlScore := (gkScore + defScore + midfieldScore + attackScore) * getFormationControlBoost(lineup)

	itemBoost := getTeamItemBoost(source, lineup)

	return applyBoost(itemBoost, controlScore)
}

func getAverageControlScore(boost float64, players []SelectedPlayer) float64 {
	var totalControlScore float64
	for _, player := range players {
		totalControlScore += boost * player.GetControlScore()
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
func CalculateTeamDefenseScore(source *rand.Rand, lineup GameLineup) float64 {
	// group players by position
	var playersByPosition = make(map[PlayerPosition][]SelectedPlayer)
	for _, player := range lineup.Players {
		playersByPosition[player.SelectedPosition] = append(playersByPosition[player.SelectedPosition], player)
	}

	// calculate the average control score for each position
	averageControlScoresByPosition := make(map[PlayerPosition]float64)
	boost := getPositionItemBoost(source, lineup.ItemBoosts, PlayerPositionDefense)
	for position, players := range playersByPosition {
		averageControlScoresByPosition[position] = getAverageDefenseScore(boost, players)
	}

	gkScore := averageControlScoresByPosition[PlayerPositionGoalkeeper] * 35 / 100
	defScore := averageControlScoresByPosition[PlayerPositionDefense] * 40 / 100
	midfieldScore := averageControlScoresByPosition[PlayerPositionMidfield] * 20 / 100
	attackScore := averageControlScoresByPosition[PlayerPositionAttack] * 5 / 100

	defenseScore := gkScore + defScore + midfieldScore + attackScore

	itemBoost := getTeamItemBoost(source, lineup)

	return applyBoost(itemBoost, defenseScore)
}

func getAverageDefenseScore(boost float64, players []SelectedPlayer) float64 {
	var totalDefenseScore float64
	for _, player := range players {
		totalDefenseScore += boost * player.GetDefenseScore()
	}
	return totalDefenseScore / float64(len(players))
}
