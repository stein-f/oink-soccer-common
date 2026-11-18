package soccer_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var allFormations = []soccer.FormationType{
	soccer.FormationTypePyramid, soccer.FormationTypeDiamond, soccer.FormationTypeY, soccer.FormationTypeBox,
}

func runSteps(t *testing.T, seed int64, home, away soccer.GameLineup) soccer.MatchOutcome {
	t.Helper()
	out, err := soccer.RunGameWithModel(rand.New(rand.NewSource(seed)), home, away, soccer.ModelSteps)
	require.NoError(t, err)
	return out
}

func TestSteps_IsDeterministic(t *testing.T) {
	home := testdata.StrongTeam(soccer.FormationTypeY)
	away := testdata.WeakTeam(soccer.FormationTypePyramid)
	assert.Equal(t, runSteps(t, 42, home, away), runSteps(t, 42, home, away))
	assert.NotEqual(t, runSteps(t, 42, home, away).Events, runSteps(t, 43, home, away).Events)
}

func TestSteps_OtherModelsCarryNoStepDetail(t *testing.T) {
	// Seasons 1-18 must serialise exactly as they did: none of the step
	// model's fields may appear on a classic or contest match.
	home := testdata.StrongTeam(soccer.FormationTypeDiamond)
	away := testdata.WeakTeam(soccer.FormationTypeBox)
	for _, model := range []soccer.Model{soccer.ModelClassic, soccer.ModelContest} {
		out, err := soccer.RunGameWithModel(rand.New(rand.NewSource(7)), home, away, model)
		require.NoError(t, err)
		raw, err := json.Marshal(out)
		require.NoError(t, err)
		for _, key := range []string{`"steps"`, `"assist_id"`, `"detail"`, `"stopper_id"`, `"totals"`, `"Stop"`} {
			assert.NotContains(t, string(raw), key, "model %q", model)
		}
	}
}

// playerIDs is the set of player ids in a line-up.
func playerIDs(l soccer.GameLineup) map[string]bool {
	ids := map[string]bool{}
	for _, p := range l.Players {
		ids[p.ID] = true
	}
	return ids
}

// TestSteps_EventsAreCausal checks every event of many matches tells a story
// that could have happened: the steps are in an order that makes sense, the
// players belong to the right sides, and the summary fields agree with the
// steps.
func TestSteps_EventsAreCausal(t *testing.T) {
	// Take-ons a wide player finished himself, by the home side's shape.
	cutIns := map[soccer.FormationType]int{}
	defer func() {
		// The Y has two forwards, a side each. (Away sides cut in too, so
		// every shape sees some.)
		assert.Greater(t, cutIns[soccer.FormationTypeY], 0)
	}()
	for _, hf := range allFormations {
		for _, af := range allFormations {
			home, away := testdata.StrongTeam(hf), testdata.WeakTeam(af)
			homeIDs, awayIDs := playerIDs(home), playerIDs(away)
			for seed := int64(0); seed < 40; seed++ {
				out := runSteps(t, seed, home, away)
				require.NotNil(t, out.Totals)
				stops, lastMinute := 0, 0
				for i, e := range out.Events {
					if e.Steps[0].Kind == soccer.StepKindRebound {
						// A rebound follows the save or the woodwork it came off.
						require.Greater(t, i, 0)
						before := out.Events[i-1]
						require.Equal(t, e.Minute, before.Minute)
						require.Equal(t, soccer.GameEventTypeMiss, before.Type)
						came := map[string]string{"parried": "save", "post": "post", "bar": "bar"}[before.GetMissEvent().Detail]
						require.Equal(t, came, e.Steps[0].Detail)
						require.Equal(t, soccer.ChanceTypeOpenPlay, e.ChanceType)
					}
					require.GreaterOrEqual(t, e.Minute, lastMinute, "events in order")
					lastMinute = e.Minute
					require.NotEmpty(t, e.Steps, "%s event with no steps", e.Type)

					var team soccer.TeamType
					switch ev := e.Event.(type) {
					case soccer.GoalEvent:
						team = ev.TeamType
						last := e.Steps[len(e.Steps)-1]
						require.Equal(t, soccer.StepKindSave, last.Kind)
						require.True(t, last.Won)
						require.Equal(t, "goal", last.Detail)
						require.Equal(t, ev.PlayerID, last.ActorID)
						require.NotEqual(t, ev.PlayerID, ev.AssistID, "a player can't assist himself")
						if ev.AssistID != "" {
							set := false
							for _, s := range e.Steps {
								if s.ActorID == ev.AssistID && s.Won && (s.Kind == soccer.StepKindPass || s.Kind == soccer.StepKindCross || s.Kind == soccer.StepKindTakeOn) {
									set = true
								}
							}
							require.True(t, set, "the assister set the goal up in the steps")
						}
						switch ev.Detail {
						case "":
						case "worldie":
							require.Equal(t, soccer.ChanceTypeLongRange, e.ChanceType)
						case "solo":
							require.Equal(t, soccer.StepKindTakeOn, e.Steps[0].Kind)
							require.Equal(t, "solo", e.Steps[0].Detail)
							require.Empty(t, ev.AssistID, "a solo goal is nobody else's")
						case "rebound":
							require.Equal(t, soccer.StepKindRebound, e.Steps[0].Kind)
						case "keeper":
							require.Equal(t, soccer.ChanceTypeCorner, e.ChanceType)
						default:
							t.Fatalf("unknown goal detail %q", ev.Detail)
						}
					case soccer.MissEvent:
						team = ev.TeamType
						last := e.Steps[len(e.Steps)-1]
						require.False(t, last.Won)
						require.Equal(t, ev.PlayerID, last.ActorID)
						require.Equal(t, ev.Detail, last.Detail)
						switch ev.Detail {
						case "blocked":
							require.Equal(t, soccer.StepKindShot, last.Kind)
							require.NotEmpty(t, ev.StopperID)
							require.Equal(t, ev.StopperID, last.OpponentID)
						case "wide", "over", "post", "bar":
							require.Equal(t, soccer.StepKindShot, last.Kind)
							require.Empty(t, ev.StopperID)
						case "held", "parried":
							require.Equal(t, soccer.StepKindSave, last.Kind)
							require.Equal(t, ev.StopperID, last.OpponentID)
						default:
							t.Fatalf("unknown miss detail %q", ev.Detail)
						}
					case soccer.StopEvent:
						stops++
						require.Equal(t, soccer.GameEventTypeStop, e.Type)
						require.Empty(t, e.ChanceType)
						last := e.Steps[len(e.Steps)-1]
						require.False(t, last.Won)
						require.Equal(t, ev.AgainstID, last.ActorID)
						require.Equal(t, ev.PlayerID, last.OpponentID)
						require.Contains(t, []string{"tackle", "caught"}, ev.Detail)
						// The stop belongs to the defending side.
						team = soccer.TeamTypeHome
						if ev.TeamType == soccer.TeamTypeHome {
							team = soccer.TeamTypeAway
						}
					default:
						t.Fatalf("unexpected event %T", e.Event)
					}

					attackers, defenders := homeIDs, awayIDs
					if team == soccer.TeamTypeAway {
						attackers, defenders = awayIDs, homeIDs
					}
					sawShotOnTarget := false
					for i, s := range e.Steps {
						require.True(t, attackers[s.ActorID], "step %d actor %q is on the attacking side", i, s.ActorID)
						if s.OpponentID != "" {
							require.True(t, defenders[s.OpponentID], "step %d opponent %q is on the defending side", i, s.OpponentID)
						}
						switch s.Kind {
						case soccer.StepKindTakeOn:
							if s.Detail == "cut_in" {
								// He came inside to shoot, and did.
								require.True(t, s.Won)
								require.Equal(t, soccer.ChanceTypeOpenPlay, e.ChanceType)
								require.Equal(t, s.ActorID, e.Steps[len(e.Steps)-1].ActorID)
								cutIns[hf]++
							}
						case soccer.StepKindShot:
							sawShotOnTarget = s.Won
						case soccer.StepKindSave:
							require.True(t, sawShotOnTarget, "a save follows a shot on target")
							require.Equal(t, len(e.Steps)-1, i, "nothing follows the save")
						}
						if i < len(e.Steps)-1 && s.Kind != soccer.StepKindTakeOn {
							// Only a lost take-on carries on (into the foul it drew).
							require.True(t, s.Won, "step %d (%s) was lost but the attack went on", i, s.Kind)
						}
					}
					if e.ChanceType == soccer.ChanceTypeCorner {
						require.Equal(t, soccer.StepKindCross, e.Steps[0].Kind)
						require.Equal(t, "corner", e.Steps[0].Detail)
						require.Equal(t, soccer.StepKindAerial, e.Steps[1].Kind)
					}
				}
				require.LessOrEqual(t, stops, 3, "stops shown are capped")

				stats := soccer.CreateGameStats(out.Events)
				require.Equal(t, stats.HomeTeamStats.Shots, out.Totals.Home.Shots)
				require.Equal(t, stats.AwayTeamStats.Shots, out.Totals.Away.Shots)
				for _, side := range []soccer.TeamTotals{out.Totals.Home, out.Totals.Away} {
					require.LessOrEqual(t, side.PassesCompleted, side.Passes)
					require.LessOrEqual(t, side.TakeOnsWon, side.TakeOns)
					require.LessOrEqual(t, side.ShotsOnTarget, side.Shots)
				}
			}
		}
	}
}

func TestSteps_ScoringLooksLikeFootball(t *testing.T) {
	home := testdata.TypicalTeam(soccer.FormationTypeDiamond)
	away := testdata.TypicalTeam(soccer.FormationTypeDiamond)
	const matches = 2000
	var totalGoals, shots, draws, nilNil int
	for seed := int64(0); seed < matches; seed++ {
		out := runSteps(t, seed, home, away)
		stats := soccer.CreateGameStats(out.Events)
		h, a := stats.HomeTeamStats.Goals, stats.AwayTeamStats.Goals
		totalGoals += h + a
		shots += stats.HomeTeamStats.Shots + stats.AwayTeamStats.Shots
		if h == a {
			draws++
		}
		if h+a == 0 {
			nilNil++
		}
	}
	perMatch := float64(totalGoals) / matches
	assert.InDelta(t, 3.2, perMatch, 0.8, "goals per match")
	assert.InDelta(t, 12.5, float64(shots)/matches, 3.5, "shots per match")
	assert.InDelta(t, 0.24, float64(draws)/matches, 0.08, "draws")
	assert.Less(t, float64(nilNil)/matches, 0.10, "0-0s")
}

func stepsWinRates(t *testing.T, matches int, home, away soccer.GameLineup) (homeWins, awayWins float64) {
	t.Helper()
	var hw, aw int
	for seed := int64(0); seed < int64(matches); seed++ {
		h, a := goals(runSteps(t, seed, home, away).Events)
		switch {
		case h > a:
			hw++
		case a > h:
			aw++
		}
	}
	return float64(hw) / float64(matches), float64(aw) / float64(matches)
}

func TestSteps_StrongerTeamWinsMoreOften(t *testing.T) {
	strong, weak := testdata.TypicalTeam(soccer.FormationTypeDiamond), testdata.ModestTeam(soccer.FormationTypeDiamond)
	strongAtHome, _ := stepsWinRates(t, 1000, strong, weak)
	_, strongAway := stepsWinRates(t, 1000, weak, strong)
	assert.Greater(t, strongAtHome, 0.70)
	assert.Greater(t, strongAway, 0.60)
}

// TestSteps_TheStrongerSideMostlyWins is the point of the possession rule: a
// side a few points better on the card wins clearly more than it loses, and
// more so the bigger the gap. Without it the step model was more of a coin
// flip than the classic one.
func TestSteps_TheStrongerSideMostlyWins(t *testing.T) {
	better := func(by int) soccer.GameLineup {
		l := testdata.TypicalTeam(soccer.FormationTypeDiamond)
		for i := range l.Players {
			a := &l.Players[i].Attributes
			a.OverallRating += by
			a.Dribbling, a.Passing, a.Pace, a.Finishing = a.Dribbling+by, a.Passing+by, a.Pace+by, a.Finishing+by
			a.Marking, a.Interceptions, a.Tackling, a.Heading = a.Marking+by, a.Interceptions+by, a.Tackling+by, a.Heading+by
			a.Reflexes, a.Handling, a.OneOnOne, a.Positioning = a.Reflexes+by, a.Handling+by, a.OneOnOne+by, a.Positioning+by
		}
		return l
	}
	base := testdata.TypicalTeam(soccer.FormationTypeDiamond)
	record := func(by int) (wins, losses float64) {
		w1, l1 := stepsWinRates(t, 1500, better(by), base)
		l2, w2 := stepsWinRates(t, 1500, base, better(by))
		return (w1 + w2) / 2, (l1 + l2) / 2
	}
	w3, l3 := record(3)
	w6, l6 := record(6)
	assert.Greater(t, w3, 0.52, "three points better: wins most of its matches")
	assert.Greater(t, w3, l3+0.25)
	assert.Greater(t, w6, 0.68, "six points better: wins about seven in ten")
	assert.Less(t, l6, 0.12)
}

func TestSteps_HomeAdvantage(t *testing.T) {
	team := testdata.TypicalTeam(soccer.FormationTypeDiamond)
	home, away := stepsWinRates(t, 4000, team, team)
	assert.Greater(t, home, away, "the same side at home wins more than away")
	assert.Less(t, home-away, 0.20, "but not by much")
}

// TestSteps_AnAttributeMovesItsOwnContest: a forward who is better on the
// ball takes his man on more and beats him more. The win rate of the take-ons
// a side chooses to try moves less than the player's rating, because a side
// with a poor dribbler tries it only when it is on; so this counts what he
// achieves over a match, and checks nobody is ever a certainty.
func TestSteps_AnAttributeMovesItsOwnContest(t *testing.T) {
	away := testdata.TypicalTeam(soccer.FormationTypeDiamond)
	takeOns := func(dribbling int) (wonPerMatch, winRate float64) {
		home := testdata.TypicalTeam(soccer.FormationTypeDiamond)
		for i := range home.Players {
			if home.Players[i].SelectedPosition == soccer.PlayerPositionAttack {
				home.Players[i].Attributes.Dribbling = dribbling
			}
		}
		const matches = 1500
		var won, tried int
		for seed := int64(0); seed < matches; seed++ {
			out := runSteps(t, seed, home, away)
			won += out.Totals.Home.TakeOnsWon
			tried += out.Totals.Home.TakeOns
		}
		return float64(won) / matches, float64(won) / float64(tried)
	}
	poor, _ := takeOns(55)
	good, _ := takeOns(75)
	great, rate := takeOns(95)
	assert.Greater(t, good, poor*1.15)
	assert.Greater(t, great, good*1.15)
	assert.Less(t, rate, 0.85, "the best dribbler still loses the ball sometimes")
}

func TestSteps_ABetterKeeperConcedesLess(t *testing.T) {
	home := testdata.TypicalTeam(soccer.FormationTypeDiamond)
	conceded := func(reflexes int) int {
		away := testdata.TypicalTeam(soccer.FormationTypeDiamond)
		for i := range away.Players {
			if away.Players[i].SelectedPosition == soccer.PlayerPositionGoalkeeper {
				a := &away.Players[i].Attributes
				a.Reflexes, a.Handling, a.OneOnOne = reflexes, reflexes, reflexes
			}
		}
		total := 0
		for seed := int64(0); seed < 1500; seed++ {
			h, _ := goals(runSteps(t, seed, home, away).Events)
			total += h
		}
		return total
	}
	assert.Greater(t, conceded(60), conceded(90))
}

func benchmarkModel(b *testing.B, model soccer.Model) {
	home := testdata.TypicalTeam(soccer.FormationTypeDiamond)
	away := testdata.TypicalTeam(soccer.FormationTypeY)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := soccer.RunGameWithModel(rand.New(rand.NewSource(int64(i))), home, away, model); err != nil {
			b.Fatal(err)
		}
	}
}

// Prediction lines run a few hundred simulations per match, so a model's
// cost per match matters. Compare with: go test -bench RunGame -run xxx .
func BenchmarkRunGame_Classic(b *testing.B) { benchmarkModel(b, soccer.ModelClassic) }
func BenchmarkRunGame_Contest(b *testing.B) { benchmarkModel(b, soccer.ModelContest) }
func BenchmarkRunGame_Steps(b *testing.B)   { benchmarkModel(b, soccer.ModelSteps) }
