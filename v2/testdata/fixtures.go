// Package testdata contains shared test lineups for the v2 engine. The
// player ratings mirror v1's testdata fixtures so behaviour can be compared
// against the v1-baseline snapshots in v2/testdata/golden/v1-baseline/.
package testdata

import (
	"strconv"

	soccer "github.com/stein-f/oink-soccer-common/v2"
)

// Lineups are built from the formation's actual slot layout — the same way
// lost-pigs builds production lineups — so a Box fixture really fields
// GK/DEF/DEF/ATK/ATK. Each slot is filled with the archetype stat line for
// its position. Player IDs are "1".."5" by slot (StrongTeam) and "6".."10"
// (WeakTeam) to keep golden snapshots readable.

// stat lines: gk, def, ctrl, atk, speed
type statLine struct{ gk, def, ctrl, atk, speed int }

var strongStats = map[soccer.PlayerPosition]statLine{
	soccer.PlayerPositionGoalkeeper: {88, 33, 21, 37, 74},
	soccer.PlayerPositionDefense:    {14, 90, 81, 37, 80},
	soccer.PlayerPositionMidfield:   {14, 55, 85, 91, 80},
	soccer.PlayerPositionAttack:     {14, 22, 85, 93, 80},
}

var weakStats = map[soccer.PlayerPosition]statLine{
	soccer.PlayerPositionGoalkeeper: {65, 12, 33, 2, 55},
	soccer.PlayerPositionDefense:    {14, 75, 81, 11, 56},
	soccer.PlayerPositionMidfield:   {14, 65, 76, 72, 68},
	soccer.PlayerPositionAttack:     {14, 22, 67, 74, 68},
}

// StrongTeam returns a 5-player squad with high overall ratings, shaped to
// the given formation's slots.
func StrongTeam(formation soccer.FormationType) soccer.GameLineup {
	return teamForFormation("strong", formation, strongStats, 0)
}

// WeakTeam returns a deliberately mediocre squad — used as a foil for
// "stronger team should win more often" smoke tests.
func WeakTeam(formation soccer.FormationType) soccer.GameLineup {
	return teamForFormation("weak", formation, weakStats, 5)
}

func teamForFormation(teamID string, formation soccer.FormationType, stats map[soccer.PlayerPosition]statLine, idOffset int) soccer.GameLineup {
	config := formationConfig(formation)
	players := make([]soccer.SelectedPlayer, 0, 5)
	for slot := uint64(1); slot <= 5; slot++ {
		pos := config.Slots[slot]
		players = append(players, player(strconv.Itoa(int(slot)+idOffset), pos, stats[pos]))
	}
	return soccer.GameLineup{
		Team:    soccer.Team{ID: teamID, Formation: formation},
		Players: players,
	}
}

func formationConfig(formation soccer.FormationType) soccer.FormationConfig {
	switch formation {
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

func player(id string, pos soccer.PlayerPosition, s statLine) soccer.SelectedPlayer {
	return soccer.SelectedPlayer{
		ID: id, Name: id,
		Attributes: soccer.PlayerAttributes{
			GoalkeeperRating: s.gk, DefenseRating: s.def, ControlRating: s.ctrl, AttackRating: s.atk, SpeedRating: s.speed,
			PrimaryPosition: pos, Positions: []soccer.PlayerPosition{pos},
		},
		SelectedPosition: pos,
	}
}
