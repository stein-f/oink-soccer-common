package soccer_test

import (
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTeam_GetOverallTeamControlScore(t *testing.T) {
	tests := map[string]struct {
		gotPlayers                  []soccer.SelectedPlayer
		wantOverallTeamControlScore int
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

			assert.Equal(t, test.wantOverallTeamControlScore, overallTeamControlScore)
		})
	}
}

func TestTeam_GetOverallTeamDefenseScore(t *testing.T) {
	tests := map[string]struct {
		gotPlayers                  []soccer.SelectedPlayer
		wantOverallTeamDefenseScore int
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
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			overallTeamDefenseScore := soccer.CalculateTeamDefenseScore(test.gotPlayers)

			assert.Equal(t, test.wantOverallTeamDefenseScore, overallTeamDefenseScore)
		})
	}
}
