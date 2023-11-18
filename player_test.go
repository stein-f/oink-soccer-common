package soccer_test

import (
	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stretchr/testify/assert"
	"testing"
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
			wantPlayerControlScore: 83,
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
