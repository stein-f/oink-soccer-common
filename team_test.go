package soccer_test

import (
	"math"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
	"github.com/stretchr/testify/assert"
)

func TestTeam_GetOverallTeamControlScore(t *testing.T) {
	tests := map[string]struct {
		gotTeam                     soccer.GameLineup
		wantOverallTeamControlScore float64
	}{
		"low control midfielders has larger overall impact on team control": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamControlScore: 68,
		},
		"low control attackers has smaller overall impact on team control": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamControlScore: 73,
		},
		"with box formation": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypeBox,
				},
				Players: []soccer.SelectedPlayer{
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
						SelectedPosition: soccer.PlayerPositionDefense,
						Attributes: soccer.PlayerAttributes{
							Position:      soccer.PlayerPositionDefense,
							ControlRating: 80,
							SpeedRating:   80,
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
					{
						SelectedPosition: soccer.PlayerPositionAttack,
						Attributes: soccer.PlayerAttributes{
							Position:      soccer.PlayerPositionAttack,
							ControlRating: 80,
							SpeedRating:   80,
						},
					},
				},
			},
			wantOverallTeamControlScore: 80,
		},
		"with diamond control boost": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypeDiamond,
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamControlScore: 76,
		},
		"with max score": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamControlScore: 97,
		},
		"with 10% position boost": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				ItemBoosts: []soccer.Boost{
					{
						BoostType:     soccer.BoostTypePosition,
						BoostPosition: soccer.PlayerPositionMidfield,
						MinBoost:      1.1,
						MaxBoost:      1.1,
					},
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamControlScore: 74,
		},
		"with no position boost when position doesn't match": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				ItemBoosts: []soccer.Boost{
					{
						BoostType:     soccer.BoostTypePosition,
						BoostPosition: soccer.PlayerPositionAttack,
						MinBoost:      1.1,
						MaxBoost:      1.1,
					},
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamControlScore: 68,
		},
		"handles free agents": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
					{
						SelectedPosition: soccer.PlayerPositionGoalkeeper,
						Attributes: soccer.PlayerAttributes{
							Position:      soccer.PlayerPositionAny,
							ControlRating: 55,
							SpeedRating:   55,
						},
					},
					{
						SelectedPosition: soccer.PlayerPositionDefense,
						Attributes: soccer.PlayerAttributes{
							Position:      soccer.PlayerPositionAny,
							ControlRating: 55,
							SpeedRating:   55,
						},
					},
					{
						SelectedPosition: soccer.PlayerPositionMidfield,
						Attributes: soccer.PlayerAttributes{
							Position:      soccer.PlayerPositionAny,
							ControlRating: 55,
							SpeedRating:   55,
						},
					},
					{
						SelectedPosition: soccer.PlayerPositionMidfield,
						Attributes: soccer.PlayerAttributes{
							Position:      soccer.PlayerPositionAny,
							ControlRating: 55,
							SpeedRating:   55,
						},
					},
					{
						SelectedPosition: soccer.PlayerPositionAttack,
						Attributes: soccer.PlayerAttributes{
							Position:      soccer.PlayerPositionAny,
							ControlRating: 55,
							SpeedRating:   55,
						},
					},
				},
			},
			wantOverallTeamControlScore: 53,
		},
		"with 10% team boost": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				ItemBoosts: []soccer.Boost{
					{
						BoostType: soccer.BoostTypeTeam,
						MinBoost:  1.1,
						MaxBoost:  1.1,
					},
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamControlScore: 74,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			overallTeamControlScore := soccer.CalculateTeamControlScore(testdata.TimeNowRandSource(), test.gotTeam)

			assert.Equal(t, test.wantOverallTeamControlScore, math.Floor(overallTeamControlScore))
		})
	}
}

func TestCalculateTeamControlScore_OutOfPositionPenalty(t *testing.T) {
	constRating := 80
	score := soccer.CalculateTeamControlScore(testdata.TimeNowRandSource(), soccer.GameLineup{
		Team: soccer.Team{Formation: soccer.FormationTypePyramid},
		Players: []soccer.SelectedPlayer{
			createPlayer(constRating, constRating, soccer.PlayerPositionGoalkeeper, soccer.PlayerPositionGoalkeeper),
			createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
			createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
			createPlayer(constRating, constRating, soccer.PlayerPositionMidfield, soccer.PlayerPositionMidfield),
			createPlayer(constRating, constRating, soccer.PlayerPositionAttack, soccer.PlayerPositionAttack),
		}})
	assert.Equal(t, roundToOneDecimal(77.6), roundToOneDecimal(score))

	scoreWithOutOfPositionMidfielder := soccer.CalculateTeamControlScore(testdata.TimeNowRandSource(), soccer.GameLineup{
		Team: soccer.Team{Formation: soccer.FormationTypePyramid},
		Players: []soccer.SelectedPlayer{
			createPlayer(constRating, constRating, soccer.PlayerPositionGoalkeeper, soccer.PlayerPositionGoalkeeper),
			createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
			createPlayer(constRating, constRating, soccer.PlayerPositionDefense, soccer.PlayerPositionDefense),
			createPlayer(constRating, constRating, soccer.PlayerPositionAttack, soccer.PlayerPositionMidfield),
			createPlayer(constRating, constRating, soccer.PlayerPositionAttack, soccer.PlayerPositionAttack),
		}})
	assert.Equal(t, float64(70), math.Floor(scoreWithOutOfPositionMidfielder))
}

func roundToOneDecimal(num float64) float64 {
	return math.Round(num*10) / 10
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
		gotTeam                     soccer.GameLineup
		wantOverallTeamDefenseScore float64
	}{
		"high scoring defenders has larger overall impact on team defense": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
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
				}},
			wantOverallTeamDefenseScore: 94,
		},
		"low scoring defenders has smaller overall impact on team defense": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamDefenseScore: 75,
		},
		"with max score": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamDefenseScore: 100,
		},
		"handles free agents": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamDefenseScore: 31,
		},
		"with 10% boost": {
			gotTeam: soccer.GameLineup{
				Team: soccer.Team{
					Formation: soccer.FormationTypePyramid,
				},
				ItemBoosts: []soccer.Boost{
					{
						BoostType: soccer.BoostTypeTeam,
						MinBoost:  1.1,
						MaxBoost:  1.1,
					},
				},
				Players: []soccer.SelectedPlayer{
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
			},
			wantOverallTeamDefenseScore: 34,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			overallTeamDefenseScore := soccer.CalculateTeamDefenseScore(testdata.TimeNowRandSource(), test.gotTeam)

			assert.Equal(t, test.wantOverallTeamDefenseScore, math.Floor(overallTeamDefenseScore))
		})
	}
}
