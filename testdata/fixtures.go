package testdata

import (
	"math/rand"
	"time"

	soccer "github.com/stein-f/oink-soccer-common"
)

var StrongTeamPlayers = []soccer.SelectedPlayer{
	{
		ID:   "1",
		Name: "1",
		Attributes: soccer.PlayerAttributes{
			GoalkeeperRating: 88,
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
			ControlRating:    81,
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
}

func TimeNowRandSource() *rand.Rand {
	source := rand.NewSource(time.Now().UnixNano())
	return rand.New(source)
}

func FixedRandSource(seed int64) *rand.Rand {
	source := rand.NewSource(seed)
	return rand.New(source)
}

var WeakTeamPlayers = []soccer.SelectedPlayer{
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
			DefenseRating:    75,
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
			AttackRating:     74,
			Position:         soccer.PlayerPositionAttack,
		},
		SelectedPosition: soccer.PlayerPositionAttack,
	},
}
