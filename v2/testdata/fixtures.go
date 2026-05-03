// Package testdata contains shared test lineups for the v2 engine. The
// player ratings mirror v1's testdata fixtures so behaviour can be compared
// against the v1-baseline snapshots in v2/testdata/golden/v1-baseline/.
package testdata

import (
	soccer "github.com/stein-f/oink-soccer-common/v2"
)

// StrongTeam returns a 5-player squad with high overall ratings. Same
// numbers as v1's testdata.StrongTeam.
func StrongTeam(formation soccer.FormationType) soccer.GameLineup {
	return soccer.GameLineup{
		Team: soccer.Team{ID: "strong", Formation: formation},
		Players: []soccer.SelectedPlayer{
			player("1", soccer.PlayerPositionGoalkeeper, 88, 33, 21, 37, 74),
			player("2", soccer.PlayerPositionDefense, 14, 90, 81, 37, 80),
			player("3", soccer.PlayerPositionMidfield, 14, 55, 85, 91, 80),
			player("4", soccer.PlayerPositionMidfield, 11, 75, 81, 71, 81),
			player("5", soccer.PlayerPositionAttack, 14, 22, 85, 93, 80),
		},
	}
}

// WeakTeam returns a deliberately mediocre squad — used as a foil for
// "stronger team should win more often" smoke tests.
func WeakTeam(formation soccer.FormationType) soccer.GameLineup {
	return soccer.GameLineup{
		Team: soccer.Team{ID: "weak", Formation: formation},
		Players: []soccer.SelectedPlayer{
			player("6", soccer.PlayerPositionGoalkeeper, 65, 12, 33, 2, 55),
			player("7", soccer.PlayerPositionDefense, 14, 75, 81, 11, 56),
			player("8", soccer.PlayerPositionMidfield, 14, 65, 76, 72, 68),
			player("9", soccer.PlayerPositionMidfield, 11, 67, 71, 55, 71),
			player("10", soccer.PlayerPositionAttack, 14, 22, 67, 74, 68),
		},
	}
}

func player(id string, pos soccer.PlayerPosition, gk, def, ctrl, atk, speed int) soccer.SelectedPlayer {
	return soccer.SelectedPlayer{
		ID: id, Name: id,
		Attributes: soccer.PlayerAttributes{
			GoalkeeperRating: gk, DefenseRating: def, ControlRating: ctrl, AttackRating: atk, SpeedRating: speed,
			PrimaryPosition: pos, Positions: []soccer.PlayerPosition{pos},
		},
		SelectedPosition: pos,
	}
}
