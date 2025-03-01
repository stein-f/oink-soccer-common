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

	gkRatio := 0.05
	defRatio := 0.15
	midRatio := 0.65
	attackRatio := 0.15

	// adjust values for the box formation which has no midfielders
	if lineup.Team.Formation == FormationTypeBox {
		gkRatio = 0.05
		defRatio = 0.35
		midRatio = 0
		attackRatio = 0.6
	}

	gkScore := averageControlScoresByPosition[PlayerPositionGoalkeeper] * gkRatio
	defScore := averageControlScoresByPosition[PlayerPositionDefense] * defRatio
	midfieldScore := averageControlScoresByPosition[PlayerPositionMidfield] * midRatio
	attackScore := averageControlScoresByPosition[PlayerPositionAttack] * attackRatio

	controlScore := (gkScore + defScore + midfieldScore + attackScore) * getFormationControlBoost(lineup)

	itemBoost := GetTeamBoost(source, lineup)

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
	averageDefenseScoresByPosition := make(map[PlayerPosition]float64)
	boost := getPositionItemBoost(source, lineup.ItemBoosts, PlayerPositionDefense)
	for position, players := range playersByPosition {
		averageDefenseScoresByPosition[position] = getAverageDefenseScore(boost, players)
	}

	gkRatio := 0.35
	defRatio := 0.40
	midRatio := 0.20
	attackRatio := 0.05

	// adjust values for the box formation which has no midfielders
	if lineup.Team.Formation == FormationTypeBox {
		gkRatio = 0.35
		defRatio = 0.5
		midRatio = 0
		attackRatio = 0.15
	}

	gkScore := averageDefenseScoresByPosition[PlayerPositionGoalkeeper] * gkRatio
	defScore := averageDefenseScoresByPosition[PlayerPositionDefense] * defRatio
	midfieldScore := averageDefenseScoresByPosition[PlayerPositionMidfield] * midRatio
	attackScore := averageDefenseScoresByPosition[PlayerPositionAttack] * attackRatio

	defenseScore := (gkScore + defScore + midfieldScore + attackScore) * getFormationDefenseBoost(lineup)

	itemBoost := GetTeamBoost(source, lineup)

	resultBoosted := applyBoost(itemBoost, defenseScore)
	if resultBoosted > 100 {
		return 100
	}
	return resultBoosted
}

func getAverageDefenseScore(boost float64, players []SelectedPlayer) float64 {
	var totalDefenseScore float64
	for _, player := range players {
		totalDefenseScore += boost * player.GetDefenseScore()
	}
	return totalDefenseScore / float64(len(players))
}
