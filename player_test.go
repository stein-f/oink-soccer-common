package soccer_test

import (
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
)

func TestPlayerAttributes_GetControlRating(t *testing.T) {
	tests := map[string]struct {
		gotPlayerSpeedRating   int
		gotPlayerControlRating int
		wantPlayerControlScore float64
	}{
		"equal speed and control rating": {
			gotPlayerSpeedRating:   80,
			gotPlayerControlRating: 80,
			wantPlayerControlScore: 80,
		},
		"higher speed rating has small overall impact on score": {
			gotPlayerSpeedRating:   90,
			gotPlayerControlRating: 80,
			wantPlayerControlScore: 82,
		},
		"higher control rating has large overall impact on score": {
			gotPlayerSpeedRating:   80,
			gotPlayerControlRating: 90,
			wantPlayerControlScore: 88,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			attributes := soccer.PlayerAttributes{
				SpeedRating:   test.gotPlayerSpeedRating,
				ControlRating: test.gotPlayerControlRating,
			}

			controlScore := attributes.GetControlScore()

			assert.Equal(t, test.wantPlayerControlScore, controlScore)
		})
	}
}

func TestPlayerAttributes_GetAttackRating(t *testing.T) {
	tests := map[string]struct {
		gotPlayerSpeedRating  int
		gotPlayerAttackRating int
		wantPlayerAttackScore float64
	}{
		"equal speed and attack rating": {
			gotPlayerSpeedRating:  80,
			gotPlayerAttackRating: 80,
			wantPlayerAttackScore: 80,
		},
		"higher speed rating has small overall impact on score": {
			gotPlayerSpeedRating:  90,
			gotPlayerAttackRating: 80,
			wantPlayerAttackScore: 83,
		},
		"higher attack rating has large overall impact on score": {
			gotPlayerSpeedRating:  80,
			gotPlayerAttackRating: 90,
			wantPlayerAttackScore: 88,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			attributes := soccer.PlayerAttributes{
				SpeedRating:  test.gotPlayerSpeedRating,
				AttackRating: test.gotPlayerAttackRating,
			}

			attackScore := attributes.GetAttackScore()

			assert.Equal(t, test.wantPlayerAttackScore, attackScore)
		})
	}
}

func TestGetOverallRating(t *testing.T) {
	tests := map[string]struct {
		gotPlayer soccer.PlayerAttributes
		expected  int
	}{
		"gk": {
			gotPlayer: soccer.PlayerAttributes{
				PrimaryPosition: soccer.PlayerPositionGoalkeeper,
				Positions: []soccer.PlayerPosition{
					soccer.PlayerPositionGoalkeeper,
				},
				GoalkeeperRating: 86,
				SpeedRating:      80,
				ControlRating:    21,
			},
			expected: 85,
		},
		"df": {
			gotPlayer: soccer.PlayerAttributes{
				PrimaryPosition: soccer.PlayerPositionDefense,
				Positions: []soccer.PlayerPosition{
					soccer.PlayerPositionDefense,
				},
				DefenseRating: 80,
				SpeedRating:   73,
				ControlRating: 70,
				AttackRating:  12,
			},
			expected: 78,
		},
		"mf": {
			gotPlayer: soccer.PlayerAttributes{
				PrimaryPosition: soccer.PlayerPositionMidfield,
				Positions: []soccer.PlayerPosition{
					soccer.PlayerPositionMidfield,
				},
				DefenseRating: 70,
				SpeedRating:   80,
				ControlRating: 83,
				AttackRating:  75,
			},
			expected: 82,
		},
		"att": {
			gotPlayer: soccer.PlayerAttributes{
				PrimaryPosition: soccer.PlayerPositionAttack,
				Positions: []soccer.PlayerPosition{
					soccer.PlayerPositionAttack,
				},
				DefenseRating: 44,
				SpeedRating:   78,
				ControlRating: 79,
				AttackRating:  92,
			},
			expected: 88,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.gotPlayer.GetOverallRating(); got != tt.expected {
				t.Errorf("GetOverallRating() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsOutOfPosition(t *testing.T) {
	tests := map[string]struct {
		gotPosition       soccer.PlayerPosition
		gotPositions      []soccer.PlayerPosition
		wantOutOfPosition bool
	}{
		"attack player in attack position": {
			gotPosition: soccer.PlayerPositionAttack,
			gotPositions: []soccer.PlayerPosition{
				soccer.PlayerPositionAttack,
			},
			wantOutOfPosition: false,
		},
		"attack player in defense position": {
			gotPosition: soccer.PlayerPositionDefense,
			gotPositions: []soccer.PlayerPosition{
				soccer.PlayerPositionAttack,
			},
			wantOutOfPosition: true,
		},
		"multi-position player in secondary position": {
			gotPosition: soccer.PlayerPositionMidfield,
			gotPositions: []soccer.PlayerPosition{
				soccer.PlayerPositionAttack,
				soccer.PlayerPositionMidfield,
			},
			wantOutOfPosition: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			player := soccer.SelectedPlayer{
				ID:   "1",
				Name: "Player",
				Attributes: soccer.PlayerAttributes{
					GoalkeeperRating: 14,
					SpeedRating:      80,
					DefenseRating:    22,
					ControlRating:    85,
					AttackRating:     93,
					PrimaryPosition:  soccer.PlayerPositionAttack,
					Positions:        tt.gotPositions,
				},
				SelectedPosition: tt.gotPosition,
			}

			gotOutOfPosition := player.IsOutOfPosition()

			assert.Equal(t, tt.wantOutOfPosition, gotOutOfPosition)
		})
	}
}
