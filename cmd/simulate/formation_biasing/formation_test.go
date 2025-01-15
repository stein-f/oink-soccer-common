package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/gocarina/gocsv"
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/cmd/allocation"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"os"
	"testing"
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
	allPlayers, err := GetAllPlayers()
	assert.NoError(t, err)

	fmt.Println("Running simulation, this may take a while...")

	// Store results (wins, losses, draws, shots)
	results := make(map[soccer.FormationType][4]int) // [wins, losses, draws, total shots]

	gameCount := 10000

	// Run matches between each pair of formations
	formations := soccer.FormationTypeValues()
	for _, formationA := range formations {
		for _, formationB := range formations {
			if formationA == formationB {
				continue
			}

			winsA, winsB, draws, shotsA, shotsB := 0, 0, 0, 0, 0
			for i := 0; i < gameCount; i++ {
				lineupA := getLineup(allPlayers, formationA)
				lineupB := getLineup(allPlayers, formationB)

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

func getLineup(allPlayers []allocation.FifaPlayer, formation soccer.FormationType) soccer.GameLineup {
	players := []soccer.SelectedPlayer{}
	if formation == soccer.FormationTypeBox {
		addPlayer(allPlayers, &players, soccer.PlayerPositionGoalkeeper)
		addPlayer(allPlayers, &players, soccer.PlayerPositionDefense)
		addPlayer(allPlayers, &players, soccer.PlayerPositionDefense)
		addPlayer(allPlayers, &players, soccer.PlayerPositionAttack)
		addPlayer(allPlayers, &players, soccer.PlayerPositionAttack)
	}
	if formation == soccer.FormationTypeDiamond {
		addPlayer(allPlayers, &players, soccer.PlayerPositionGoalkeeper)
		addPlayer(allPlayers, &players, soccer.PlayerPositionDefense)
		addPlayer(allPlayers, &players, soccer.PlayerPositionMidfield)
		addPlayer(allPlayers, &players, soccer.PlayerPositionMidfield)
		addPlayer(allPlayers, &players, soccer.PlayerPositionAttack)
	}
	if formation == soccer.FormationTypePyramid {
		addPlayer(allPlayers, &players, soccer.PlayerPositionGoalkeeper)
		addPlayer(allPlayers, &players, soccer.PlayerPositionDefense)
		addPlayer(allPlayers, &players, soccer.PlayerPositionDefense)
		addPlayer(allPlayers, &players, soccer.PlayerPositionMidfield)
		addPlayer(allPlayers, &players, soccer.PlayerPositionAttack)
	}
	if formation == soccer.FormationTypeY {
		addPlayer(allPlayers, &players, soccer.PlayerPositionGoalkeeper)
		addPlayer(allPlayers, &players, soccer.PlayerPositionDefense)
		addPlayer(allPlayers, &players, soccer.PlayerPositionMidfield)
		addPlayer(allPlayers, &players, soccer.PlayerPositionAttack)
		addPlayer(allPlayers, &players, soccer.PlayerPositionAttack)
	}
	return soccer.GameLineup{
		Team:    soccer.Team{Formation: formation},
		Players: players,
	}
}

func getDecentPlayer(allPlayers []allocation.FifaPlayer, position soccer.PlayerPosition) allocation.FifaPlayer {
	rand.Shuffle(len(allPlayers), func(i, j int) {
		allPlayers[i], allPlayers[j] = allPlayers[j], allPlayers[i]
	})

	for _, player := range allPlayers {
		if player.PlayerAttributes.Position == position && player.PlayerAttributes.GetOverallRating() > 80 {
			return player
		}
	}

	panic("no decent player found")
}

func addPlayer(allPlayers []allocation.FifaPlayer, selectedPlayers *[]soccer.SelectedPlayer, position soccer.PlayerPosition) {
	player := getDecentPlayer(allPlayers, position)
	*selectedPlayers = append(*selectedPlayers, soccer.SelectedPlayer{
		ID:               player.PlayerID,
		Attributes:       player.PlayerAttributes,
		SelectedPosition: position,
	})
}

func GetAllPlayers() ([]allocation.FifaPlayer, error) {
	file, err := os.ReadFile(fmt.Sprintf("%s/../../../cmd/allocation/fifa_players_22.csv", cwd()))
	if err != nil {
		return nil, err
	}
	var records []allocation.Record
	err = gocsv.UnmarshalBytes(file, &records)
	if err != nil {
		return nil, err
	}
	var players []allocation.FifaPlayer
	for _, record := range records {
		players = append(players, record.ToDomain(testdata.TimeNowRandSource()))
	}

	// get 10 random players above 80 for each position
	var fixedPlayers []allocation.FifaPlayer
	for _, position := range soccer.AllPositions {
		if position == soccer.PlayerPositionAny {
			continue
		}
		var playersPerPosition []allocation.FifaPlayer
		for {
			player := getDecentPlayer(players, position)
			playersPerPosition = append(playersPerPosition, player)
			if len(playersPerPosition) == 100 {
				fixedPlayers = append(fixedPlayers, playersPerPosition...)
				break
			}
		}
	}

	return fixedPlayers, nil
}

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get working directory: %v", err))
	}
	return dir
}
