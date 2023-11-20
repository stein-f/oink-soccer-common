package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	soccer "github.com/stein-f/oink-soccer-common"
	"log"
)

//go:embed home_team.json
var homeTeamConfig []byte

//go:embed away_team.json
var awayTeamConfig []byte

func main() {
	var homeWins, awayWins, draws, goals int
	gameCount := 500

	scorerByPosition := make(map[soccer.PlayerPosition]int)

	homeLineup := loadConfig(homeTeamConfig)
	awayLineup := loadConfig(awayTeamConfig)

	for i := 0; i < gameCount; i++ {
		gameEvents, err := soccer.RunGame(homeLineup, awayLineup)
		if err != nil {
			panic(err)
		}

		gameStats := soccer.CreateGameStats(gameEvents)
		fmt.Printf("StrongTeam %d - WeakTeam %d\n", gameStats.HomeTeamStats.Goals, gameStats.AwayTeamStats.Goals)

		if gameStats.HomeTeamStats.Goals > gameStats.AwayTeamStats.Goals {
			homeWins++
		} else if gameStats.HomeTeamStats.Goals < gameStats.AwayTeamStats.Goals {
			awayWins++
		} else {
			draws++
		}

		for _, event := range gameEvents {
			if event.Type == soccer.GameEventTypeGoal {
				scorerID := event.Event.(soccer.GoalEvent).PlayerID
				homeScorer, homeFound := homeLineup.FindPlayer(scorerID)
				awayScorer, awayFound := awayLineup.FindPlayer(scorerID)
				if !homeFound && !awayFound {
					panic(fmt.Sprintf("scorer %s not found", scorerID))
				}
				if homeFound {
					scorerByPosition[homeScorer.SelectedPosition]++
					continue
				}
				scorerByPosition[awayScorer.SelectedPosition]++
			}
		}

		fmt.Println(fmt.Sprintf("StrongTeam scored %d goals from %d chances", gameStats.HomeTeamStats.Goals, gameStats.HomeTeamStats.Shots))
		fmt.Println(fmt.Sprintf("WeakTeam scored %d goals from %d chances", gameStats.AwayTeamStats.Goals, gameStats.AwayTeamStats.Shots))

		goals += gameStats.HomeTeamStats.Goals + gameStats.AwayTeamStats.Goals
	}

	goalsPerGame := float64(goals) / float64(gameCount)

	fmt.Printf("\nGame summary:\n")
	fmt.Printf("Games played: %d\n", gameCount)
	fmt.Printf("StrongTeam wins: %d\n", homeWins)
	fmt.Printf("WeakTeam wins: %d\n", awayWins)
	fmt.Printf("Draws: %d\n", draws)
	fmt.Printf("Goals/game: %f\n", goalsPerGame)
	fmt.Printf("Scorer by position: %v\n", scorerByPosition)
}

func loadConfig(config []byte) soccer.GameLineup {
	var lineup soccer.GameLineup
	if err := json.Unmarshal(config, &lineup); err != nil {
		log.Fatal(err)
	}
	return lineup
}
