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
				Position:         soccer.PlayerPositionGoalkeeper,
				GoalkeeperRating: 86,
				SpeedRating:      80,
				ControlRating:    21,
			},
			expected: 85,
		},
		"df": {
			gotPlayer: soccer.PlayerAttributes{
				Position:      soccer.PlayerPositionDefense,
				DefenseRating: 80,
				SpeedRating:   73,
				ControlRating: 70,
				AttackRating:  12,
			},
			expected: 78,
		},
		"mf": {
			gotPlayer: soccer.PlayerAttributes{
				Position:      soccer.PlayerPositionMidfield,
				DefenseRating: 70,
				SpeedRating:   80,
				ControlRating: 83,
				AttackRating:  75,
			},
			expected: 82,
		},
		"att": {
			gotPlayer: soccer.PlayerAttributes{
				Position:      soccer.PlayerPositionAttack,
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
