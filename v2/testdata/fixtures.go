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

// TypicalTeam and ModestTeam are squads of typical season-19 players: every
// slot holds the median FC 27 player for its position, at overall 80-86
// (TypicalTeam) or 66-72 (ModestTeam), with the player-on-player attributes
// set. Unlike StrongTeam, whose defenders and forwards pass like midfielders,
// they have the shape real squads have, so they are the ones to measure
// formation balance and step-model realism on.
func TypicalTeam(formation soccer.FormationType) soccer.GameLineup {
	return typicalTeam("typical", formation, typicalPlayers, 20)
}

// ModestTeam is TypicalTeam a level down. See TypicalTeam.
func ModestTeam(formation soccer.FormationType) soccer.GameLineup {
	return typicalTeam("modest", formation, modestPlayers, 30)
}

func typicalTeam(teamID string, formation soccer.FormationType, players map[soccer.PlayerPosition]soccer.PlayerAttributes, idOffset int) soccer.GameLineup {
	config := formationConfig(formation)
	selected := make([]soccer.SelectedPlayer, 0, 5)
	for slot := uint64(1); slot <= 5; slot++ {
		pos := config.Slots[slot]
		id := strconv.Itoa(int(slot) + idOffset)
		attrs := players[pos]
		attrs.PrimaryPosition, attrs.Positions = pos, []soccer.PlayerPosition{pos}
		selected = append(selected, soccer.SelectedPlayer{ID: id, Name: id, Attributes: attrs, SelectedPosition: pos})
	}
	return soccer.GameLineup{Team: soccer.Team{ID: teamID, Formation: formation}, Players: selected}
}

var typicalPlayers = map[soccer.PlayerPosition]soccer.PlayerAttributes{
	soccer.PlayerPositionGoalkeeper: {
		GoalkeeperRating: 79, DefenseRating: 15, SpeedRating: 81, ControlRating: 15, AttackRating: 15, AggressionRating: 28, OverallRating: 81,
		WorkRate: 36, Finishing: 12, Heading: 41, Technique: 13, Composure: 59, Tackling: 15,
		Dribbling: 35, Passing: 48, Crossing: 14, Pace: 45, Strength: 68, Interceptions: 18, Positioning: 12, Marking: 15, ShotPower: 35,
		Reflexes: 82, Handling: 79, OneOnOne: 69,
	},
	soccer.PlayerPositionDefense: {
		GoalkeeperRating: 10, DefenseRating: 81, SpeedRating: 74, ControlRating: 74, AttackRating: 58, AggressionRating: 81, OverallRating: 81,
		WorkRate: 79, Finishing: 55, Heading: 81, Technique: 59, Composure: 79, Tackling: 81,
		Dribbling: 71, Passing: 75, Crossing: 69, Pace: 74, Strength: 80, Interceptions: 81, Positioning: 68, Marking: 81, ShotPower: 65,
		Reflexes: 10, Handling: 10, OneOnOne: 10,
	},
	soccer.PlayerPositionMidfield: {
		GoalkeeperRating: 10, DefenseRating: 57, SpeedRating: 77, ControlRating: 79, AttackRating: 77, AggressionRating: 70, OverallRating: 82,
		WorkRate: 80, Finishing: 77, Heading: 67, Technique: 75, Composure: 81, Tackling: 57,
		Dribbling: 81, Passing: 80, Crossing: 79, Pace: 77, Strength: 70, Interceptions: 60, Positioning: 80, Marking: 58, ShotPower: 79,
		Reflexes: 10, Handling: 10, OneOnOne: 10,
	},
	soccer.PlayerPositionAttack: {
		GoalkeeperRating: 9, DefenseRating: 41, SpeedRating: 77, ControlRating: 71, AttackRating: 83, AggressionRating: 72, OverallRating: 82,
		WorkRate: 75, Finishing: 85, Heading: 83, Technique: 69, Composure: 80, Tackling: 35,
		Dribbling: 76, Passing: 72, Crossing: 69, Pace: 76, Strength: 83, Interceptions: 38, Positioning: 85, Marking: 39, ShotPower: 80,
		Reflexes: 10, Handling: 9, OneOnOne: 10,
	},
}

var modestPlayers = map[soccer.PlayerPosition]soccer.PlayerAttributes{
	soccer.PlayerPositionGoalkeeper: {
		GoalkeeperRating: 67, DefenseRating: 15, SpeedRating: 69, ControlRating: 15, AttackRating: 15, AggressionRating: 25, OverallRating: 69,
		WorkRate: 29, Finishing: 10, Heading: 36, Technique: 12, Composure: 41, Tackling: 13,
		Dribbling: 30, Passing: 34, Crossing: 13, Pace: 37, Strength: 61, Interceptions: 14, Positioning: 9, Marking: 12, ShotPower: 30,
		Reflexes: 69, Handling: 67, OneOnOne: 57,
	},
	soccer.PlayerPositionDefense: {
		GoalkeeperRating: 10, DefenseRating: 66, SpeedRating: 67, ControlRating: 59, AttackRating: 45, AggressionRating: 68, OverallRating: 68,
		WorkRate: 72, Finishing: 41, Heading: 69, Technique: 44, Composure: 62, Tackling: 66,
		Dribbling: 63, Passing: 60, Crossing: 52, Pace: 67, Strength: 73, Interceptions: 66, Positioning: 53, Marking: 66, ShotPower: 49,
		Reflexes: 10, Handling: 10, OneOnOne: 10,
	},
	soccer.PlayerPositionMidfield: {
		GoalkeeperRating: 10, DefenseRating: 51, SpeedRating: 74, ControlRating: 65, AttackRating: 63, AggressionRating: 58, OverallRating: 68,
		WorkRate: 70, Finishing: 64, Heading: 58, Technique: 60, Composure: 65, Tackling: 51,
		Dribbling: 72, Passing: 66, Crossing: 62, Pace: 74, Strength: 63, Interceptions: 52, Positioning: 66, Marking: 49, ShotPower: 64,
		Reflexes: 10, Handling: 10, OneOnOne: 10,
	},
	soccer.PlayerPositionAttack: {
		GoalkeeperRating: 10, DefenseRating: 31, SpeedRating: 72, ControlRating: 57, AttackRating: 68, AggressionRating: 60, OverallRating: 68,
		WorkRate: 68, Finishing: 70, Heading: 73, Technique: 55, Composure: 65, Tackling: 26,
		Dribbling: 66, Passing: 57, Crossing: 53, Pace: 72, Strength: 77, Interceptions: 25, Positioning: 70, Marking: 27, ShotPower: 67,
		Reflexes: 10, Handling: 10, OneOnOne: 10,
	},
}
