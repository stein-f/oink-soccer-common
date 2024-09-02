package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/stein-f/oink-soccer-common/testdata"
	"log"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
)

// Embed formation JSON files
//
//go:embed standard_box.json
var boxTeam []byte

//go:embed standard_diamond.json
var diamondTeam []byte

//go:embed standard_pyramid.json
var pyramidTeam []byte

//go:embed standard_y.json
var yTeam []byte

var formations = map[string][]byte{
	"The Box":     boxTeam,
	"The Diamond": diamondTeam,
	"The Pyramid": pyramidTeam,
	"The Y":       yTeam,
}

func TestShowControlScores(t *testing.T) {
	for formationA, configA := range formations {
		teamLineup := loadConfig(configA)
		score := soccer.CalculateTeamControlScore(testdata.TimeNowRandSource(), teamLineup)
		fmt.Printf("%s: %f\n", formationA, score)
	}
}

func TestShowDefenseScores(t *testing.T) {
	for formationA, configA := range formations {
		teamLineup := loadConfig(configA)
		score := soccer.CalculateTeamDefenseScore(testdata.TimeNowRandSource(), teamLineup)
		fmt.Printf("%s: %f\n", formationA, score)
	}
}

func TestShowAttackScores(t *testing.T) {
	for formationA, configA := range formations {
		teamLineup := loadConfig(configA)
		for _, player := range teamLineup.Players {
			attackModifier := getAttackFormationBoost(teamLineup)
			attackScore := player.GetControlScore() * attackModifier
			fmt.Printf("%s: %s: %f\n", formationA, player.SelectedPosition, attackScore)
		}
	}
}

func TestShowWinRatios(t *testing.T) {
	// Store results (wins, losses, draws, shots)
	results := make(map[string][4]int) // [wins, losses, draws, total shots]

	gameCount := 50_000

	// Run matches between each pair of formations
	for formationA, configA := range formations {
		for formationB, configB := range formations {
			if formationA >= formationB {
				continue
			}

			winsA, winsB, draws, shotsA, shotsB := 0, 0, 0, 0, 0
			lineupA := loadConfig(configA)
			lineupB := loadConfig(configB)

			for i := 0; i < gameCount; i++ {
				gameEvents, _, err := soccer.RunGame(lineupA, lineupB)
				if err != nil {
					panic(err)
				}

				gameStats := soccer.CreateGameStats(gameEvents)

				if gameStats.HomeTeamStats.Goals > gameStats.AwayTeamStats.Goals {
					winsA++
				} else if gameStats.HomeTeamStats.Goals < gameStats.AwayTeamStats.Goals {
					winsB++
				} else {
					draws++
				}

				// Accumulate shots
				shotsA += gameStats.HomeTeamStats.Shots
				shotsB += gameStats.AwayTeamStats.Shots
			}

			// Update results for formationA
			resultA := results[formationA]
			resultA[0] += winsA
			resultA[1] += winsB
			resultA[2] += draws
			resultA[3] += shotsA
			results[formationA] = resultA

			// Update results for formationB
			resultB := results[formationB]
			resultB[0] += winsB
			resultB[1] += winsA
			resultB[2] += draws
			resultB[3] += shotsB
			results[formationB] = resultB
		}
	}

	// Print a summary table of results
	fmt.Println("Formation\tWins\tLosses\tDraws\tShots\tWin%")
	fmt.Println("-------------------------------------------------")
	for formation, result := range results {
		wins, losses, draws, shots := result[0], result[1], result[2], result[3]
		totalGames := wins + losses + draws
		winPercentage := float64(wins) / float64(totalGames) * 100

		fmt.Printf("%-10s\t%d\t%d\t%d\t%d\t%.2f%%\n", formation, wins, losses, draws, shots, winPercentage)
	}
}

func loadConfig(config []byte) soccer.GameLineup {
	var lineup soccer.GameLineup
	if err := json.Unmarshal(config, &lineup); err != nil {
		log.Fatal(err)
	}
	return lineup
}

func getAttackFormationBoost(lineup soccer.GameLineup) float64 {
	formationConfig := getFormationConfig(lineup.Team.Formation)
	return formationConfig.AttackModifier
}

func getFormationConfig(formationType soccer.FormationType) soccer.FormationConfig {
	switch formationType {
	case soccer.FormationTypePyramid:
		return soccer.ThePyramidFormation
	case soccer.FormationTypeY:
		return soccer.TheYFormation
	case soccer.FormationTypeBox:
		return soccer.TheBoxFormation
	default:
		return soccer.TheDiamondFormation
	}
}
