package main

import (
	"fmt"
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
)

func main() {
	var homeWins, awayWins, draws, goals int
	games := 500

	scorerByPosition := make(map[soccer.PlayerPosition]int)

	for i := 0; i < games; i++ {
		homeLineup := soccer.GameLineup{
			Team: soccer.Team{
				ID:         "1",
				CustomName: "Coventry City",
				Formation:  soccer.FormationTypeDiamond,
			},
			Players: testdata.StrongTeamPlayers,
		}
		awayLineup := soccer.GameLineup{
			Team: soccer.Team{
				ID:         "2",
				CustomName: "Aston Villa",
				Formation:  soccer.FormationTypeDiamond,
			},
			Players: testdata.WeakTeamPlayers,
		}

		gameEvents, err := soccer.RunGame(homeLineup, awayLineup)
		if err != nil {
			panic(err)
		}

		gameStats := soccer.CreateGameStats(gameEvents)
		fmt.Printf("Coventry City %d - Aston Villa %d\n", gameStats.HomeTeamStats.Goals, gameStats.AwayTeamStats.Goals)

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

		fmt.Println(fmt.Sprintf("Coventry City scored %d goals from %d chances", gameStats.HomeTeamStats.Goals, gameStats.HomeTeamStats.Shots))
		fmt.Println(fmt.Sprintf("Aston Villa scored %d goals from %d chances", gameStats.AwayTeamStats.Goals, gameStats.AwayTeamStats.Shots))

		goals += gameStats.HomeTeamStats.Goals + gameStats.AwayTeamStats.Goals
	}

	goalsPerGame := float64(goals) / float64(games)

	fmt.Printf("\nCoventry City wins: %d\n", homeWins)
	fmt.Printf("Aston Villa wins: %d\n", awayWins)
	fmt.Printf("Draws: %d\n", draws)
	fmt.Printf("Goals/game: %f\n", goalsPerGame)
	fmt.Printf("Scorer by position: %v\n", scorerByPosition)
}
