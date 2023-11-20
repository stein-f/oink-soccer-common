package main

import (
	"fmt"
	soccer "github.com/stein-f/oink-soccer-common"
)

var homeLineup = soccer.GameLineup{
	Team: soccer.Team{
		ID:         "1",
		CustomName: "StrongTeam",
		Formation:  soccer.FormationTypeDiamond,
	},
	Players: []soccer.SelectedPlayer{
		{
			ID:   "1",
			Name: "1",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 90,
				SpeedRating:      74,
				DefenseRating:    33,
				ControlRating:    21,
				AttackRating:     37,
				Position:         soccer.PlayerPositionGoalkeeper,
			},
			SelectedPosition: soccer.PlayerPositionGoalkeeper,
		},
		{
			ID:   "2",
			Name: "2",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 14,
				SpeedRating:      80,
				DefenseRating:    90,
				ControlRating:    81,
				AttackRating:     37,
				Position:         soccer.PlayerPositionDefense,
			},
			SelectedPosition: soccer.PlayerPositionDefense,
		},
		{
			ID:   "3",
			Name: "3",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 14,
				SpeedRating:      80,
				DefenseRating:    55,
				ControlRating:    85,
				AttackRating:     91,
				Position:         soccer.PlayerPositionMidfield,
			},
			SelectedPosition: soccer.PlayerPositionMidfield,
		},
		{
			ID:   "4",
			Name: "4",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 11,
				SpeedRating:      81,
				DefenseRating:    75,
				ControlRating:    88,
				AttackRating:     71,
				Position:         soccer.PlayerPositionMidfield,
			},
			SelectedPosition: soccer.PlayerPositionMidfield,
		},
		{
			ID:   "5",
			Name: "5",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 14,
				SpeedRating:      80,
				DefenseRating:    22,
				ControlRating:    85,
				AttackRating:     93,
				Position:         soccer.PlayerPositionAttack,
			},
			SelectedPosition: soccer.PlayerPositionAttack,
		},
	},
}

var awayLineup = soccer.GameLineup{
	Team: soccer.Team{
		ID:         "2",
		CustomName: "WeakTeam",
		Formation:  soccer.FormationTypeDiamond,
	},
	Players: []soccer.SelectedPlayer{
		{
			ID:   "6",
			Name: "6",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 65,
				SpeedRating:      55,
				DefenseRating:    12,
				ControlRating:    33,
				AttackRating:     2,
				Position:         soccer.PlayerPositionGoalkeeper,
			},
			SelectedPosition: soccer.PlayerPositionGoalkeeper,
		},
		{
			ID:   "7",
			Name: "7",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 14,
				SpeedRating:      56,
				DefenseRating:    68,
				ControlRating:    81,
				AttackRating:     11,
				Position:         soccer.PlayerPositionDefense,
			},
			SelectedPosition: soccer.PlayerPositionDefense,
		},
		{
			ID:   "8",
			Name: "8",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 14,
				SpeedRating:      68,
				DefenseRating:    65,
				ControlRating:    76,
				AttackRating:     72,
				Position:         soccer.PlayerPositionMidfield,
			},
			SelectedPosition: soccer.PlayerPositionMidfield,
		},
		{
			ID:   "9",
			Name: "9",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 11,
				SpeedRating:      71,
				DefenseRating:    67,
				ControlRating:    71,
				AttackRating:     55,
				Position:         soccer.PlayerPositionMidfield,
			},
			SelectedPosition: soccer.PlayerPositionMidfield,
		},
		{
			ID:   "10",
			Name: "10",
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: 14,
				SpeedRating:      68,
				DefenseRating:    22,
				ControlRating:    67,
				AttackRating:     70,
				Position:         soccer.PlayerPositionAttack,
			},
			SelectedPosition: soccer.PlayerPositionAttack,
		},
	},
}

func main() {
	var homeWins, awayWins, draws, goals int
	gameCount := 500

	scorerByPosition := make(map[soccer.PlayerPosition]int)

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
