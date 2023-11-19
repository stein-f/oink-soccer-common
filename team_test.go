package soccer_test

import (
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestTeam_GetOverallTeamControlScore(t *testing.T) {
	tests := map[string]struct {
		gotPlayers                  []soccer.SelectedPlayer
		wantOverallTeamControlScore float64
	}{
		"low control midfielders has larger overall impact on team control": {
			gotPlayers: []soccer.SelectedPlayer{
				{
					SelectedPosition: soccer.PlayerPositionGoalkeeper,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionGoalkeeper,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionDefense,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionDefense,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						ControlRating: 50,
						SpeedRating:   50,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionAttack,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAttack,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
			},
			wantOverallTeamControlScore: 70,
		},
		"low control attackers has smaller overall impact on team control": {
			gotPlayers: []soccer.SelectedPlayer{
				{
					SelectedPosition: soccer.PlayerPositionGoalkeeper,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionGoalkeeper,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionDefense,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionDefense,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						ControlRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionAttack,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAttack,
						ControlRating: 50,
						SpeedRating:   50,
					},
				},
			},
			wantOverallTeamControlScore: 75,
		},
		"with max score": {
			gotPlayers: []soccer.SelectedPlayer{
				{
					SelectedPosition: soccer.PlayerPositionGoalkeeper,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionGoalkeeper,
						ControlRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionDefense,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionDefense,
						ControlRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						ControlRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						ControlRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionAttack,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAttack,
						ControlRating: 100,
						SpeedRating:   100,
					},
				},
			},
			wantOverallTeamControlScore: 100,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			overallTeamControlScore := soccer.CalculateTeamControlScore(test.gotPlayers)

			assert.Equal(t, test.wantOverallTeamControlScore, math.Floor(overallTeamControlScore))
		})
	}
}

func TestCalculateTeamControlScore_OutOfPositionPenalty(t *testing.T) {
	constRating := 80
	score := soccer.CalculateTeamControlScore([]soccer.SelectedPlayer{
		createPlayer(constRating, constRating, soccer.PlayerPositionGoalkeeper, soccer.PlayerPositionGoalkeeper),
		createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
		createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
		createPlayer(constRating, constRating, soccer.PlayerPositionMidfield, soccer.PlayerPositionMidfield),
		createPlayer(constRating, constRating, soccer.PlayerPositionAttack, soccer.PlayerPositionAttack),
	})
	assert.Equal(t, float64(constRating), score)

	scoreWithOutOfPositionMidfielder := soccer.CalculateTeamControlScore([]soccer.SelectedPlayer{
		createPlayer(constRating, constRating, soccer.PlayerPositionGoalkeeper, soccer.PlayerPositionGoalkeeper),
		createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
		createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
		createPlayer(constRating, constRating, soccer.PlayerPositionMidfield, soccer.PlayerPositionAttack),
		createPlayer(constRating, constRating, soccer.PlayerPositionAttack, soccer.PlayerPositionAttack),
	})
	assert.Equal(t, float64(27), math.Floor(scoreWithOutOfPositionMidfielder))
}

func createPlayer(control, speed int, position soccer.PlayerPosition, selectedPosition soccer.PlayerPosition) soccer.SelectedPlayer {
	return soccer.SelectedPlayer{
		SelectedPosition: selectedPosition,
		Attributes: soccer.PlayerAttributes{
			Position:      position,
			ControlRating: control,
			SpeedRating:   speed,
		},
	}
}

func TestTeam_GetOverallTeamDefenseScore(t *testing.T) {
	tests := map[string]struct {
		gotPlayers                  []soccer.SelectedPlayer
		wantOverallTeamDefenseScore float64
	}{
		"high scoring defenders has larger overall impact on team defense": {
			gotPlayers: []soccer.SelectedPlayer{
				{
					SelectedPosition: soccer.PlayerPositionGoalkeeper,
					Attributes: soccer.PlayerAttributes{
						Position:         soccer.PlayerPositionGoalkeeper,
						GoalkeeperRating: 80,
						SpeedRating:      80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionDefense,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionDefense,
						DefenseRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						DefenseRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						DefenseRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionAttack,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAttack,
						DefenseRating: 50,
						SpeedRating:   50,
					},
				},
			},
			wantOverallTeamDefenseScore: 86,
		},
		"low scoring defenders has smaller overall impact on team defense": {
			gotPlayers: []soccer.SelectedPlayer{
				{
					SelectedPosition: soccer.PlayerPositionGoalkeeper,
					Attributes: soccer.PlayerAttributes{
						Position:         soccer.PlayerPositionGoalkeeper,
						GoalkeeperRating: 80,
						SpeedRating:      80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionDefense,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionDefense,
						DefenseRating: 50,
						SpeedRating:   50,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						DefenseRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						DefenseRating: 80,
						SpeedRating:   80,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionAttack,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAttack,
						DefenseRating: 100,
						SpeedRating:   100,
					},
				},
			},
			wantOverallTeamDefenseScore: 69,
		},
		"with max score": {
			gotPlayers: []soccer.SelectedPlayer{
				{
					SelectedPosition: soccer.PlayerPositionGoalkeeper,
					Attributes: soccer.PlayerAttributes{
						Position:         soccer.PlayerPositionGoalkeeper,
						GoalkeeperRating: 100,
						SpeedRating:      100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionDefense,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionDefense,
						DefenseRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						DefenseRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionMidfield,
						DefenseRating: 100,
						SpeedRating:   100,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionAttack,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAttack,
						DefenseRating: 100,
						SpeedRating:   100,
					},
				},
			},
			wantOverallTeamDefenseScore: 100,
		},
		"handles free agents": {
			gotPlayers: []soccer.SelectedPlayer{
				{
					SelectedPosition: soccer.PlayerPositionGoalkeeper,
					Attributes: soccer.PlayerAttributes{
						Position:         soccer.PlayerPositionAny,
						GoalkeeperRating: 40,
						SpeedRating:      40,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionDefense,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAny,
						DefenseRating: 40,
						SpeedRating:   40,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAny,
						DefenseRating: 40,
						SpeedRating:   40,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionMidfield,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAny,
						DefenseRating: 40,
						SpeedRating:   40,
					},
				},
				{
					SelectedPosition: soccer.PlayerPositionAttack,
					Attributes: soccer.PlayerAttributes{
						Position:      soccer.PlayerPositionAny,
						DefenseRating: 40,
						SpeedRating:   40,
					},
				},
			},
			wantOverallTeamDefenseScore: 29,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			overallTeamDefenseScore := soccer.CalculateTeamDefenseScore(test.gotPlayers)

			assert.Equal(t, test.wantOverallTeamDefenseScore, math.Floor(overallTeamDefenseScore))
		})
	}
}
