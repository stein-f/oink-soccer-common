package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"

	soccer "github.com/stein-f/oink-soccer-common"
)

//go:embed home_team.json
var homeTeamConfig []byte

//go:embed away_team.json
var awayTeamConfig []byte

func main() {
	var homeWins, awayWins, draws, goals, homeChances, awayChances int
	gameCount := 10000

	scorerByPosition := make(map[soccer.PlayerPosition]int)

	homeLineup := loadConfig(homeTeamConfig)
	awayLineup := loadConfig(awayTeamConfig)

	for i := 0; i < gameCount; i++ {
		gameEvents, err := soccer.RunGame(homeLineup, awayLineup)
		if err != nil {
			panic(err)
		}

		gameStats := soccer.CreateGameStats(gameEvents)

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

		goals += gameStats.HomeTeamStats.Goals + gameStats.AwayTeamStats.Goals
		homeChances += gameStats.HomeTeamStats.Shots
		awayChances += gameStats.AwayTeamStats.Shots
	}

	goalsPerGame := float64(goals) / float64(gameCount)
	homeTeamChancePerGame := float64(homeChances) / float64(gameCount)
	awayTeamChancePerGame := float64(awayChances) / float64(gameCount)

	fmt.Printf("\nGame summary:\n")
	fmt.Printf("Games played: %d\n", gameCount)
	fmt.Printf("Home Team wins: %d\n", homeWins)
	fmt.Printf("Home Team chances/game: %f\n", homeTeamChancePerGame)
	fmt.Printf("Away Team wins: %d\n", awayWins)
	fmt.Printf("Away Team chances/game: %f\n", awayTeamChancePerGame)
	fmt.Printf("Draws: %d\n", draws)
	fmt.Printf("Goals/game: %f\n", goalsPerGame)

	attackerGoals := scorerByPosition[soccer.PlayerPositionAttack]
	totalGoals := scorerByPosition[soccer.PlayerPositionAttack] + scorerByPosition[soccer.PlayerPositionMidfield] + scorerByPosition[soccer.PlayerPositionDefense] + scorerByPosition[soccer.PlayerPositionGoalkeeper]
	attackerGoalsPercentage := float64(attackerGoals) / float64(totalGoals) * 100
	fmt.Printf("Attacker goals: %d (%f%%)\n", attackerGoals, attackerGoalsPercentage)

	midfielderGoals := scorerByPosition[soccer.PlayerPositionMidfield]
	midfielderGoalsPercentage := float64(midfielderGoals) / float64(totalGoals) * 100
	fmt.Printf("Midfielder goals: %d (%f%%)\n", midfielderGoals, midfielderGoalsPercentage)

	defenderGoals := scorerByPosition[soccer.PlayerPositionDefense]
	defenderGoalsPercentage := float64(defenderGoals) / float64(totalGoals) * 100
	fmt.Printf("Defender goals: %d (%f%%)\n", defenderGoals, defenderGoalsPercentage)

	goalkeeperGoals := scorerByPosition[soccer.PlayerPositionGoalkeeper]
	goalkeeperGoalsPercentage := float64(goalkeeperGoals) / float64(totalGoals) * 100
	fmt.Printf("Goalkeeper goals: %d (%f%%)\n", goalkeeperGoals, goalkeeperGoalsPercentage)
}

func loadConfig(config []byte) soccer.GameLineup {
	var lineup soccer.GameLineup
	if err := json.Unmarshal(config, &lineup); err != nil {
		log.Fatal(err)
	}
	return lineup
}
