package soccer

import (
	"math/rand"
	"testing"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stepLineup(formation FormationType, prefix string) GameLineup {
	var players []SelectedPlayer
	for slot := uint64(1); slot <= 5; slot++ {
		pos := formationConfigFor(formation, map[FormationType]map[uint64]PlayerPosition{
			FormationTypePyramid: slotsPyramid, FormationTypeDiamond: slotsDiamond,
			FormationTypeY: slotsY, FormationTypeBox: slotsBox,
		}[formation]).Slots[slot]
		players = append(players, SelectedPlayer{
			ID: prefix + string(rune('0'+slot)), SelectedPosition: pos,
			Attributes: PlayerAttributes{
				GoalkeeperRating: 70, DefenseRating: 70, ControlRating: 70, AttackRating: 70, SpeedRating: 70,
				PrimaryPosition: pos, Positions: []PlayerPosition{pos},
			},
		})
	}
	return GameLineup{Team: Team{ID: prefix, Formation: formation}, Players: players}
}

func stepSides(t *testing.T, home, away FormationType) (*stepSide, *stepSide) {
	t.Helper()
	r := rand.New(rand.NewSource(1))
	h, _ := newStepSide(r, TeamTypeHome, stepLineup(home, "h"), &TeamTotals{})
	a, _ := newStepSide(r, TeamTypeAway, stepLineup(away, "a"), &TeamTotals{})
	return h, a
}

func TestSteps_LanesFollowLineupOrder(t *testing.T) {
	box, diamond := stepSides(t, FormationTypeBox, FormationTypeDiamond)

	// Two in a line take a side each, in line-up order; one holds the centre.
	require.Len(t, box.lines[PlayerPositionDefense], 2)
	assert.Equal(t, -1.0, box.lines[PlayerPositionDefense][0].lane)
	assert.Equal(t, 1.0, box.lines[PlayerPositionDefense][1].lane)
	assert.Equal(t, "h2", box.lines[PlayerPositionDefense][0].ID)
	assert.Empty(t, box.lines[PlayerPositionMidfield], "the Box fields no midfield")
	require.Len(t, diamond.lines[PlayerPositionDefense], 1)
	assert.Equal(t, 0.0, diamond.lines[PlayerPositionDefense][0].lane)
	assert.NotNil(t, box.keeper)
	assert.Len(t, box.outfield, 4)
}

func TestSteps_AWingerFacesTheDefenderOnHisSide(t *testing.T) {
	box, other := stepSides(t, FormationTypeBox, FormationTypeBox)
	left, right := box.lines[PlayerPositionAttack][0], box.lines[PlayerPositionAttack][1]

	// The attacking left wing is the defending side's right.
	opp, cover := facing(left, box, other, PlayerPositionDefense, tackleScore)
	assert.Equal(t, other.lines[PlayerPositionDefense][1].ID, opp.ID)
	assert.Equal(t, 1.0, cover, "two against two: nobody spare, nobody stretched")
	opp, _ = facing(right, box, other, PlayerPositionDefense, tackleScore)
	assert.Equal(t, other.lines[PlayerPositionDefense][0].ID, opp.ID)
}

// TestSteps_NumbersDecideCover is the whole of what a formation does in the
// step model: every pairing of shapes, for the two places an attack is met.
func TestSteps_NumbersDecideCover(t *testing.T) {
	midfield := map[FormationType]int{FormationTypePyramid: 1, FormationTypeDiamond: 2, FormationTypeY: 1, FormationTypeBox: 0}
	defence := map[FormationType]int{FormationTypePyramid: 2, FormationTypeDiamond: 1, FormationTypeY: 1, FormationTypeBox: 2}
	attack := map[FormationType]int{FormationTypePyramid: 1, FormationTypeDiamond: 1, FormationTypeY: 2, FormationTypeBox: 2}
	want := func(defenders, attackers int) float64 {
		switch {
		case defenders == 0:
			return tuning.StepsLineCover
		case defenders > attackers:
			return tuning.StepsSupport
		case defenders < attackers:
			return tuning.StepsStretched
		}
		return 1
	}
	formations := []FormationType{FormationTypePyramid, FormationTypeDiamond, FormationTypeY, FormationTypeBox}
	for _, af := range formations {
		for _, df := range formations {
			att, def := stepSides(t, af, df)

			// The forwards against the defence.
			for _, fwd := range att.lines[PlayerPositionAttack] {
				opp, cover := facing(fwd, att, def, PlayerPositionDefense, tackleScore)
				require.NotNil(t, opp, "%s v %s", af, df)
				assert.Equal(t, PlayerPositionDefense, opp.line)
				assert.Equal(t, want(defence[df], attack[af]), cover, "%s forwards v %s defence", af, df)
			}

			// The passer against the midfield. A side with no midfield
			// passes from the back; one with none to defend with sends a
			// forward back, out of his place.
			passers := att.lines[PlayerPositionMidfield]
			if len(passers) == 0 {
				passers = att.lines[PlayerPositionDefense]
			}
			for _, passer := range passers {
				opp, cover := facing(passer, att, def, PlayerPositionMidfield, interceptScore)
				require.NotNil(t, opp, "%s v %s", af, df)
				assert.Equal(t, want(midfield[df], len(passers)), cover, "%s passers v %s midfield", af, df)
				if midfield[df] == 0 {
					assert.NotEqual(t, PlayerPositionMidfield, opp.line)
				}
			}
		}
	}
}

func TestSteps_ABallWinnerIsLessStretched(t *testing.T) {
	y, diamond := stepSides(t, FormationTypeY, FormationTypeDiamond)
	fwd := y.lines[PlayerPositionAttack][0]
	_, cover := facing(fwd, y, diamond, PlayerPositionDefense, tackleScore)
	assert.Equal(t, tuning.StepsStretched, cover, "one defender against two forwards")

	diamond.lines[PlayerPositionDefense][0].Role = PlayerRoleBallWinner
	_, cover = facing(fwd, y, diamond, PlayerPositionDefense, tackleScore)
	assert.InDelta(t, (1+tuning.StepsStretched)/2, cover, 1e-9, "he makes up half the ground")
}

func TestSteps_ARatingEdgeIsWorthSomethingButNotEverything(t *testing.T) {
	// The plan's target: 20 points better wins a contest by 15-25 points
	// more than a coin toss, and nobody is ever a certainty.
	edge := oddsToP(stepOdds(90, 70, 1)) - 0.5
	assert.InDelta(t, 0.20, edge, 0.05)
	assert.InDelta(t, 0.5, oddsToP(stepOdds(80, 80, 1)), 1e-9)
	assert.Less(t, oddsToP(stepOdds(95, 70, 1)), 0.85)
	assert.Greater(t, oddsToP(stepOdds(60, 90, 1)), 0.15)
}

func TestSteps_TeamBoostsAreSoftened(t *testing.T) {
	// A +5% item is in every contest of an attack and of defending one, so
	// it reaches the odds softened.
	lineup := stepLineup(FormationTypeDiamond, "h")
	lineup.ItemBoosts = []Boost{{BoostType: BoostTypeTeam, MinBoost: 1.05, MaxBoost: 1.05}}
	side, _ := newStepSide(rand.New(rand.NewSource(1)), TeamTypeAway, lineup, &TeamTotals{})
	mod := side.mod(side.outfield[0])
	assert.Greater(t, mod, 1.0)
	assert.Less(t, mod, 1.05)
}

func TestSteps_TheBetterSideSeesMoreOfTheBall(t *testing.T) {
	h, a := stepSides(t, FormationTypeDiamond, FormationTypeBox)
	assert.InDelta(t, 1.0, h.quality()/a.quality(), 1e-9, "equal players, whatever the shapes: an equal share")

	for _, p := range h.outfield {
		p.Attributes.OverallRating = 83
	}
	h.keeper.Attributes.OverallRating = 83
	for _, p := range a.outfield {
		p.Attributes.OverallRating = 80
	}
	a.keeper.Attributes.OverallRating = 80
	share := h.quality() / (h.quality() + a.quality())
	assert.InDelta(t, 0.61, share, 0.03, "three points better on average: about 61% of the attacks")

	// A player out of position counts for less.
	before := h.quality()
	h.outfield[0].state *= tuning.StepsOutOfPosition
	assert.Less(t, h.quality(), before)
}

func TestSteps_ASecondDefenderCovers(t *testing.T) {
	// A two-man defence has someone to cover when one of them is beaten; a
	// lone defender, once beaten, is beaten.
	box, diamond := stepSides(t, FormationTypeBox, FormationTypeDiamond)
	left, right := box.lines[PlayerPositionDefense][0], box.lines[PlayerPositionDefense][1]
	assert.Equal(t, right, coverFor(box, left))
	assert.Equal(t, left, coverFor(box, right))
	assert.Nil(t, coverFor(diamond, diamond.lines[PlayerPositionDefense][0]))
	assert.Nil(t, coverFor(box, nil))
	// A forward who dropped back to cover an empty midfield has nobody
	// behind him in that line to cover in turn.
	assert.Nil(t, coverFor(box, box.lines[PlayerPositionAttack][0]))
}

// routeWeight is how much of a side's attack is one kind, by one player
// (and, for a cross, aimed at one player). "" matches anyone.
func routeWeight(routes []stepRoute, kind, actor, target string) float64 {
	var w float64
	for _, r := range routes {
		if r.kind != kind || (actor != "" && r.actor.ID != actor) {
			continue
		}
		if target != "" && (r.target == nil || r.target.ID != target) {
			continue
		}
		w += r.weight
	}
	return w
}

func stepSidesOf(t *testing.T, home, away GameLineup) (*stepSide, *stepSide) {
	t.Helper()
	r := rand.New(rand.NewSource(1))
	h, _ := newStepSide(r, TeamTypeHome, home, &TeamTotals{})
	a, _ := newStepSide(r, TeamTypeAway, away, &TeamTotals{})
	return h, a
}

// forwardsOf returns the indexes of a line-up's forwards.
func forwardsOf(l GameLineup) []int {
	var out []int
	for i, p := range l.Players {
		if p.SelectedPosition == PlayerPositionAttack {
			out = append(out, i)
		}
	}
	return out
}

func TestSteps_AnAttackGoesToThePlayerItSuits(t *testing.T) {
	// Two forwards: a quick, tricky winger and a big man who is neither.
	home := stepLineup(FormationTypeY, "h")
	fw := forwardsOf(home)
	require.Len(t, fw, 2)
	winger, bigMan := &home.Players[fw[0]], &home.Players[fw[1]]
	winger.Attributes.Dribbling, winger.Attributes.Pace = 92, 90
	winger.Attributes.Heading, winger.Attributes.Strength = 50, 50
	bigMan.Attributes.Dribbling, bigMan.Attributes.Pace = 52, 52
	bigMan.Attributes.Heading, bigMan.Attributes.Strength = 92, 92
	h, a := stepSidesOf(t, home, stepLineup(FormationTypePyramid, "a"))
	routes := (&stepMatch{}).routes(h, a)

	// The winger does the running at people, with and without help.
	assert.Greater(t, routeWeight(routes, "take_on", winger.ID, ""), 6*routeWeight(routes, "take_on", bigMan.ID, ""))
	assert.Greater(t, routeWeight(routes, "run", winger.ID, ""), 3*routeWeight(routes, "run", bigMan.ID, ""))
	assert.Greater(t, routeWeight(routes, "solo", winger.ID, ""), 6*routeWeight(routes, "solo", bigMan.ID, ""))
	// The crosses are aimed at the big man.
	assert.Greater(t, routeWeight(routes, "cross", "", bigMan.ID), 4*routeWeight(routes, "cross", "", winger.ID))

	// And the side as a whole does more of what its players are good at:
	// swap the winger for a second big man and it crosses more, dribbles less.
	plain := stepLineup(FormationTypeY, "h")
	for _, i := range forwardsOf(plain) {
		plain.Players[i].Attributes = bigMan.Attributes
	}
	ph, pa := stepSidesOf(t, plain, stepLineup(FormationTypePyramid, "a"))
	bigMen := (&stepMatch{}).routes(ph, pa)
	share := func(rs []stepRoute, kind string) float64 {
		var total float64
		for _, r := range rs {
			total += r.weight
		}
		return routeWeight(rs, kind, "", "") / total
	}
	assert.Greater(t, share(routes, "take_on"), share(bigMen, "take_on")+0.10)
	assert.Greater(t, share(bigMen, "cross"), share(routes, "cross")+0.10)
}

func TestSteps_ASlowPlayerOnlyDribblesAtASlowerOne(t *testing.T) {
	home := stepLineup(FormationTypeDiamond, "h")
	fw := forwardsOf(home)
	require.Len(t, fw, 1)
	home.Players[fw[0]].Attributes.Dribbling, home.Players[fw[0]].Attributes.Pace = 58, 45
	id := home.Players[fw[0]].ID
	share := func(tackling int) float64 {
		away := stepLineup(FormationTypeDiamond, "a")
		for i := range away.Players {
			if away.Players[i].SelectedPosition == PlayerPositionDefense {
				a := &away.Players[i].Attributes
				// Only what a take-on is up against: he is no easier to run
				// in behind, to cross past or to beat to a long ball.
				a.Tackling = tackling
			}
		}
		h, a := stepSidesOf(t, home, away)
		routes := (&stepMatch{}).routes(h, a)
		var total float64
		for _, r := range routes {
			total += r.weight
		}
		return routeWeight(routes, "take_on", id, "") / total
	}
	good, poor := share(82), share(35)
	assert.Less(t, good, 0.10, "against a good defender he hardly tries it")
	assert.Greater(t, poor, 2.5*good, "against one he can beat, he does")
}

func TestSteps_OnlyAWidePlayerCutsInside(t *testing.T) {
	cutIns := func(formation FormationType) (wide, central int) {
		h, a := stepSides(t, formation, FormationTypePyramid)
		m := &stepMatch{r: rand.New(rand.NewSource(9))}
		for _, actor := range h.lines[PlayerPositionAttack] {
			for i := 0; i < 400; i++ {
				m.dribble(h, a, actor, nil, 1, false, 30)
			}
		}
		for _, e := range m.events {
			if e.Steps[0].Kind != StepKindTakeOn || e.Steps[0].Detail != "cut_in" {
				continue
			}
			for _, p := range h.outfield {
				if p.ID == e.Steps[0].ActorID {
					if p.lane != 0 {
						wide++
					} else {
						central++
					}
				}
			}
		}
		return wide, central
	}
	// The Y's two forwards take a side each; the Diamond's one is central.
	wide, central := cutIns(FormationTypeY)
	assert.Greater(t, wide, 100)
	assert.Zero(t, central)
	wide, central = cutIns(FormationTypeDiamond)
	assert.Zero(t, wide)
	assert.Zero(t, central)
}

func TestSteps_AWorldieNeedsPower(t *testing.T) {
	strike := func(power, technique int) float64 {
		return worldieChance(PlayerAttributes{ShotPower: power, Technique: technique})
	}
	assert.Zero(t, strike(45, 45))
	assert.Greater(t, strike(75, 75), 0.0)
	assert.Greater(t, strike(92, 88), 2*strike(75, 75))
	assert.LessOrEqual(t, strike(99, 99), 1.0)
}

func TestSteps_ReboundsFallToForwardsExceptFromACorner(t *testing.T) {
	scorers := func(ct ChanceType) map[PlayerPosition]int {
		h, a := stepSides(t, FormationTypeY, FormationTypePyramid)
		m := &stepMatch{r: rand.New(rand.NewSource(3))}
		for i := 0; i < 2000; i++ {
			// At the last follow-up allowed, so nothing comes of this one.
			m.rebound(h, a, ct, "post", 30, tuning.StepsMaxFollowUps-1)
		}
		byLine := map[PlayerPosition]int{}
		for _, e := range m.events {
			require.Equal(t, StepKindRebound, e.Steps[0].Kind)
			require.Equal(t, "post", e.Steps[0].Detail)
			require.Equal(t, ChanceTypeOpenPlay, e.ChanceType)
			if e.IsGoal() {
				require.Equal(t, "rebound", e.GetGoalEvent().Detail)
				require.Empty(t, e.GetGoalEvent().AssistID)
			}
			for _, p := range h.outfield {
				if p.ID == e.Steps[0].ActorID {
					byLine[p.line]++
				}
			}
		}
		return byLine
	}
	open := scorers(ChanceTypeOpenPlay)
	assert.Greater(t, open[PlayerPositionAttack], 1500, "a poacher's ball: the forwards")
	assert.Less(t, open[PlayerPositionDefense], 100)
	corner := scorers(ChanceTypeCorner)
	assert.Greater(t, corner[PlayerPositionDefense], 350, "from a corner the defender is up too")
}

func TestSteps_TheKeeperComesUpOnlyLateAndAGoalDown(t *testing.T) {
	keeperHeaders := func(minute, homeGoals, awayGoals int) int {
		h, a := stepSides(t, FormationTypeY, FormationTypePyramid)
		m := &stepMatch{r: rand.New(rand.NewSource(5)), homeGoals: homeGoals, awayGoals: awayGoals}
		n := 0
		for i := 0; i < 600; i++ {
			m.homeGoals, m.awayGoals = homeGoals, awayGoals
			before := len(m.events)
			m.corner(h, a, minute, tuning.StepsMaxFollowUps)
			for _, e := range m.events[before:] {
				require.Equal(t, ChanceTypeCorner, e.ChanceType)
				if e.Steps[1].ActorID == h.keeper.ID {
					n++
					if e.IsGoal() {
						require.Equal(t, "keeper", e.GetGoalEvent().Detail)
					}
				}
			}
		}
		return n
	}
	assert.Zero(t, keeperHeaders(88, 1, 1), "level: he stays at home")
	assert.Zero(t, keeperHeaders(88, 2, 1), "ahead: he stays at home")
	assert.Zero(t, keeperHeaders(60, 0, 1), "behind with time left: he stays at home")
	assert.Zero(t, keeperHeaders(88, 0, 3), "well beaten: no point")
	assert.Greater(t, keeperHeaders(88, 0, 1), 20, "a goal down at the death: up he comes")
}

// typicalStepLineup is a side of ordinary top-division players, each with
// the attributes his position really has (testdata.TypicalTeam's, which this
// package can't import).
func typicalStepLineup(formation FormationType, prefix string) GameLineup {
	typical := map[PlayerPosition]PlayerAttributes{
		PlayerPositionGoalkeeper: {
			GoalkeeperRating: 79, DefenseRating: 15, SpeedRating: 81, ControlRating: 15, AttackRating: 15, AggressionRating: 28, OverallRating: 81,
			WorkRate: 36, Finishing: 12, Heading: 41, Technique: 13, Composure: 59, Tackling: 15,
			Dribbling: 35, Passing: 48, Crossing: 14, Pace: 45, Strength: 68, Interceptions: 18, Positioning: 12, Marking: 15, ShotPower: 35,
			Reflexes: 82, Handling: 79, OneOnOne: 69,
		},
		PlayerPositionDefense: {
			GoalkeeperRating: 10, DefenseRating: 81, SpeedRating: 74, ControlRating: 74, AttackRating: 58, AggressionRating: 81, OverallRating: 81,
			WorkRate: 79, Finishing: 55, Heading: 81, Technique: 59, Composure: 79, Tackling: 81,
			Dribbling: 71, Passing: 75, Crossing: 69, Pace: 74, Strength: 80, Interceptions: 81, Positioning: 68, Marking: 81, ShotPower: 65,
			Reflexes: 10, Handling: 10, OneOnOne: 10,
		},
		PlayerPositionMidfield: {
			GoalkeeperRating: 10, DefenseRating: 57, SpeedRating: 77, ControlRating: 79, AttackRating: 77, AggressionRating: 70, OverallRating: 82,
			WorkRate: 80, Finishing: 77, Heading: 67, Technique: 75, Composure: 81, Tackling: 57,
			Dribbling: 81, Passing: 80, Crossing: 79, Pace: 77, Strength: 70, Interceptions: 60, Positioning: 80, Marking: 58, ShotPower: 79,
			Reflexes: 10, Handling: 10, OneOnOne: 10,
		},
		PlayerPositionAttack: {
			GoalkeeperRating: 9, DefenseRating: 41, SpeedRating: 77, ControlRating: 71, AttackRating: 83, AggressionRating: 72, OverallRating: 82,
			WorkRate: 75, Finishing: 85, Heading: 83, Technique: 69, Composure: 80, Tackling: 35,
			Dribbling: 76, Passing: 72, Crossing: 69, Pace: 76, Strength: 83, Interceptions: 38, Positioning: 85, Marking: 39, ShotPower: 80,
			Reflexes: 10, Handling: 9, OneOnOne: 10,
		},
	}
	l := stepLineup(formation, prefix)
	for i := range l.Players {
		pos := l.Players[i].SelectedPosition
		a := typical[pos]
		a.PrimaryPosition, a.Positions = pos, []PlayerPosition{pos}
		l.Players[i].Attributes = a
	}
	return l
}

var stepShapes = []FormationType{FormationTypePyramid, FormationTypeDiamond, FormationTypeY, FormationTypeBox}

func routeShare(routes []stepRoute, kind string) float64 {
	var total float64
	for _, r := range routes {
		total += r.weight
	}
	return routeWeight(routes, kind, "", "") / total
}

// The main ways to goal pay a typical side about the same. When one pays
// much more, the attributes behind it are the only ones worth having, and
// the ones that defend the others are worth nothing: that is how runs in
// behind once made pace everything and tackling pointless.
func TestSteps_NoRouteSimplyPaysMore(t *testing.T) {
	const attempts = 6000
	goals, tries := map[string]float64{}, map[string]float64{}
	for _, as := range stepShapes {
		for _, ds := range stepShapes {
			att, def := stepSidesOf(t, typicalStepLineup(as, "h"), typicalStepLineup(ds, "a"))
			m := &stepMatch{r: rand.New(rand.NewSource(7))}
			byKind := map[string][]stepRoute{}
			for _, r := range m.routes(att, def) {
				byKind[r.kind] = append(byKind[r.kind], r)
			}
			for kind, rs := range byKind {
				var total float64
				for _, r := range rs {
					total += r.weight
				}
				for i := 0; i < attempts; i++ {
					roll, route := m.r.Float64()*total, rs[len(rs)-1]
					for _, r := range rs {
						if roll < r.weight {
							route = r
							break
						}
						roll -= r.weight
					}
					m.events, m.homeGoals, m.awayGoals = m.events[:0], 0, 0
					m.attack(att, def, route, 30)
					tries[kind]++
					for _, e := range m.events {
						if e.IsGoal() {
							goals[kind]++
						}
					}
				}
			}
		}
	}
	main := []string{"take_on", "run", "cross", "long_ball"}
	var mean float64
	for _, kind := range main {
		mean += goals[kind] / tries[kind] / float64(len(main))
	}
	for _, kind := range main {
		pays := goals[kind] / tries[kind]
		t.Logf("%-10s %.3f goals an attempt", kind, pays)
		assert.InDelta(t, mean, pays, 0.25*mean, "%s pays %.3f goals an attempt against a mean of %.3f", kind, pays, mean)
	}
	// Shooting from distance and going alone from deep are the long shots.
	assert.Less(t, goals["long_shot"]/tries["long_shot"], mean)
	assert.Less(t, goals["solo"]/tries["solo"], mean)
}

func TestSteps_ASideWithoutAMidfieldGoesLongMost(t *testing.T) {
	long := func(shape FormationType, press PressLevel) float64 {
		away := typicalStepLineup(FormationTypePyramid, "a")
		away.Team.Tactics.Press = press
		h, a := stepSidesOf(t, typicalStepLineup(shape, "h"), away)
		return routeShare((&stepMatch{}).routes(h, a), "long_ball")
	}
	box, diamond := long(FormationTypeBox, PressLevelNone), long(FormationTypeDiamond, PressLevelNone)
	assert.Greater(t, box, 1.5*diamond, "the Box has nobody to play through")
	assert.Greater(t, box, 0.08)
	assert.Less(t, box, 0.25, "it is one way to goal among several, not the only one")
	// Going long is the way over a press, and not worth it against a side
	// sitting off.
	assert.Greater(t, long(FormationTypeBox, PressLevelHigh), 1.15*box)
	assert.Less(t, long(FormationTypeBox, PressLevelLow), box)
}

func TestSteps_ALongBallIsForTheBigManAndStrengthWinsIt(t *testing.T) {
	home := typicalStepLineup(FormationTypeBox, "h")
	fw := forwardsOf(home)
	require.Len(t, fw, 2)
	bigMan, partner := &home.Players[fw[0]], &home.Players[fw[1]]
	bigMan.Role = PlayerRoleTargetMan
	bigMan.Attributes.Strength, bigMan.Attributes.Heading = 92, 90
	partner.Attributes.Strength, partner.Attributes.Heading = 62, 62
	h, a := stepSidesOf(t, home, typicalStepLineup(FormationTypePyramid, "a"))
	routes := (&stepMatch{}).routes(h, a)
	assert.Greater(t, routeWeight(routes, "long_ball", bigMan.ID, ""), 8*routeWeight(routes, "long_ball", partner.ID, ""))
	// And the side goes long more for having him.
	ph, pa := stepSidesOf(t, typicalStepLineup(FormationTypeBox, "h"), typicalStepLineup(FormationTypePyramid, "a"))
	assert.Greater(t, routeShare(routes, "long_ball"), 1.3*routeShare((&stepMatch{}).routes(ph, pa), "long_ball"))

	// What he does with them is down to his strength against the defender's.
	won := func(strength int) float64 {
		home := typicalStepLineup(FormationTypeBox, "h")
		home.Players[fw[0]].Attributes.Strength = strength
		h, a := stepSidesOf(t, home, typicalStepLineup(FormationTypePyramid, "a"))
		target := h.lines[PlayerPositionAttack][0]
		m := &stepMatch{r: rand.New(rand.NewSource(3))}
		for i := 0; i < 4000; i++ {
			// Level throughout: a side well ahead eases off.
			m.homeGoals, m.awayGoals = 0, 0
			m.longBall(h, a, target, 30)
		}
		require.Equal(t, 4000, h.totals.Passes)
		return float64(h.totals.PassesCompleted) / float64(h.totals.Passes)
	}
	weak, typical, strong := won(60), won(83), won(98)
	assert.Less(t, typical, 0.45, "the defender is facing the ball: he is the favourite")
	assert.Greater(t, typical, 0.22)
	assert.Greater(t, strong, typical+0.08)
	assert.Less(t, weak, typical-0.08)
}

func TestSteps_AFlickOnIsCoveredByTheOtherDefenderOrAMidfielder(t *testing.T) {
	covers := func(shape FormationType) map[PlayerPosition]int {
		h, a := stepSidesOf(t, typicalStepLineup(FormationTypeBox, "h"), typicalStepLineup(shape, "a"))
		line := map[string]PlayerPosition{}
		for _, p := range a.outfield {
			line[p.ID] = p.line
		}
		target := h.lines[PlayerPositionAttack][0]
		m := &stepMatch{r: rand.New(rand.NewSource(11))}
		out := map[PlayerPosition]int{}
		for i := 0; i < 2000; i++ {
			m.events, m.homeGoals, m.awayGoals = m.events[:0], 0, 0
			m.longBall(h, a, target, 30)
			for _, e := range m.events {
				// (A corner or a rebound can follow the shot it led to.)
				if e.Steps[0].Detail != "high" {
					continue
				}
				require.Equal(t, target.ID, e.Steps[1].ActorID)
				if e.Steps[1].Detail != "flick_on" {
					continue
				}
				run := e.Steps[2]
				require.Equal(t, StepKindRun, run.Kind)
				require.NotEqual(t, target.ID, run.ActorID, "someone else runs on to it")
				require.NotEqual(t, e.Steps[1].OpponentID, run.OpponentID, "the man beaten in the air doesn't cover it too")
				out[line[run.OpponentID]]++
			}
		}
		return out
	}
	twoBack := covers(FormationTypePyramid)
	assert.Greater(t, twoBack[PlayerPositionDefense], 100)
	assert.Zero(t, twoBack[PlayerPositionMidfield], "two defenders: the other one covers")
	oneBack := covers(FormationTypeDiamond)
	assert.Greater(t, oneBack[PlayerPositionMidfield], 100, "one defender: a midfielder drops in")
	assert.Zero(t, oneBack[PlayerPositionDefense])
}

func TestSteps_APoorTacklerGivesAwayMoreFouls(t *testing.T) {
	fouls := func(tackling int) float64 {
		away := typicalStepLineup(FormationTypePyramid, "a")
		for i := range away.Players {
			away.Players[i].Attributes.Tackling = tackling
		}
		h, a := stepSidesOf(t, typicalStepLineup(FormationTypeY, "h"), away)
		m := &stepMatch{r: rand.New(rand.NewSource(9))}
		actor, opp := h.lines[PlayerPositionAttack][0], a.lines[PlayerPositionDefense][0]
		const n = 20000
		for i := 0; i < n; i++ {
			m.events, m.homeGoals, m.awayGoals = m.events[:0], 0, 0
			m.tackled(h, a, actor, opp, nil, 30)
		}
		return float64(a.totals.Fouls) / n
	}
	good, typical, poor := fouls(92), fouls(72), fouls(52)
	assert.Less(t, good, 0.8*typical, "a good tackler takes the ball")
	assert.Greater(t, poor, 1.4*typical, "a poor one takes the man too")
}

func TestSteps_TheCoveringDefenderHasToTackle(t *testing.T) {
	// Behind the first defender on a run, the second isn't racing: what
	// stops the runner is his tackling.
	caught := func(tackling int) float64 {
		away := typicalStepLineup(FormationTypePyramid, "a")
		for i := range away.Players {
			if away.Players[i].SelectedPosition == PlayerPositionDefense {
				away.Players[i].Attributes.Tackling = tackling
			}
		}
		h, a := stepSidesOf(t, typicalStepLineup(FormationTypeY, "h"), away)
		m := &stepMatch{r: rand.New(rand.NewSource(13))}
		var route stepRoute
		for _, r := range m.routes(h, a) {
			if r.kind == "run" {
				route = r
				break
			}
		}
		require.NotNil(t, route.actor)
		second, stopped := 0, 0
		for i := 0; i < 30000; i++ {
			m.events, m.homeGoals, m.awayGoals = m.events[:0], 0, 0
			m.attack(h, a, route, 30)
			for _, e := range m.events {
				runs := 0
				for _, s := range e.Steps {
					if s.Kind == StepKindRun {
						runs++
					}
				}
				if runs < 2 {
					continue
				}
				second++
				if e.Type == GameEventTypeStop {
					stopped++
				}
			}
		}
		require.Greater(t, second, 500)
		return float64(stopped) / float64(second)
	}
	assert.Greater(t, caught(95), caught(60)+0.05)
}

// tacticOdds is what a pair of tactics does to a typical Y attacking a typical
// Pyramid: how often its pass forward arrives, and how its attacks are shared
// between the ways to goal.
func tacticOdds(t *testing.T, attack, defend Tactics) (pass float64, share map[string]float64) {
	t.Helper()
	home, away := typicalStepLineup(FormationTypeY, "h"), typicalStepLineup(FormationTypePyramid, "a")
	home.Team.Tactics, away.Team.Tactics = attack, defend
	h, a := stepSidesOf(t, home, away)
	passer := h.lines[PlayerPositionMidfield][0]
	odds, opp := passOdds(passer, h, a, false, 1)
	require.NotNil(t, opp)
	routes := (&stepMatch{}).routes(h, a)
	share = map[string]float64{}
	for _, kind := range []string{"take_on", "run", "cross", "long_shot", "long_ball", "solo"} {
		share[kind] = routeShare(routes, kind)
	}
	return oddsToP(odds), share
}

func TestSteps_APressTradesTimeForSpace(t *testing.T) {
	pass, share := tacticOdds(t, Tactics{}, Tactics{})
	highPass, high := tacticOdds(t, Tactics{}, Tactics{Press: PressLevelHigh})
	lowPass, low := tacticOdds(t, Tactics{}, Tactics{Press: PressLevelLow})

	// Pressed, the man on the ball has no time: fewer passes arrive, and the
	// side stops trying what needs time for what needs room.
	assert.Less(t, highPass, pass-0.02)
	assert.Greater(t, high["take_on"]+high["run"], share["take_on"]+share["run"]+0.02)
	assert.Less(t, high["cross"]+high["long_shot"], share["cross"]+share["long_shot"]-0.02)
	// Sat off, it is the other way round.
	assert.Greater(t, lowPass, pass+0.005)
	assert.Less(t, low["take_on"]+low["run"], share["take_on"]+share["run"]-0.01)
	assert.Greater(t, low["cross"]+low["long_shot"], share["cross"]+share["long_shot"]+0.01)
}

func TestSteps_APressIsOnlyAsGoodAsTheLegsInIt(t *testing.T) {
	arrives := func(workRate int, press PressLevel) float64 {
		home, away := typicalStepLineup(FormationTypeY, "h"), typicalStepLineup(FormationTypePyramid, "a")
		away.Team.Tactics.Press = press
		for i := range away.Players {
			away.Players[i].Attributes.WorkRate = workRate
		}
		h, a := stepSidesOf(t, home, away)
		odds, _ := passOdds(h.lines[PlayerPositionMidfield][0], h, a, false, 1)
		return oddsToP(odds)
	}
	// Work rate is nothing to a side that doesn't press.
	assert.Equal(t, arrives(60, PressLevelNone), arrives(92, PressLevelNone))
	// Pressing, runners get to the ball and plodders don't.
	assert.Less(t, arrives(92, PressLevelHigh), arrives(60, PressLevelHigh)-0.01)
	assert.Less(t, arrives(60, PressLevelHigh), arrives(60, PressLevelNone))

	// And it costs the side its own attack late on, the less the more it
	// can run. Nothing before the hour, nothing without a press.
	fade := func(workRate int, press PressLevel, minute int) float64 {
		home := typicalStepLineup(FormationTypeY, "h")
		home.Team.Tactics.Press = press
		for i := range home.Players {
			home.Players[i].Attributes.WorkRate = workRate
		}
		h, _ := stepSidesOf(t, home, typicalStepLineup(FormationTypePyramid, "a"))
		return h.fade(minute)
	}
	assert.Equal(t, 1.0, fade(60, PressLevelHigh, 45))
	assert.Equal(t, 1.0, fade(60, PressLevelNone, 85))
	assert.Equal(t, 1.0, fade(60, PressLevelLow, 85))
	assert.Less(t, fade(76, PressLevelHigh, 65), 1.0)
	assert.Less(t, fade(76, PressLevelHigh, 85), fade(76, PressLevelHigh, 65))
	assert.Less(t, fade(60, PressLevelHigh, 85), fade(76, PressLevelHigh, 85)-0.05)
	assert.Greater(t, fade(92, PressLevelHigh, 85), fade(76, PressLevelHigh, 85)+0.04)
}

func TestSteps_TempoTradesRoomForCare(t *testing.T) {
	pass, share := tacticOdds(t, Tactics{}, Tactics{})
	fastPass, fast := tacticOdds(t, Tactics{Tempo: TempoLevelFast}, Tactics{})
	slowPass, slow := tacticOdds(t, Tactics{Tempo: TempoLevelSlow}, Tactics{})

	// Fast: at them before they are set, and loose with the ball.
	assert.Less(t, fastPass, pass-0.005)
	assert.Greater(t, fast["take_on"]+fast["run"], share["take_on"]+share["run"]+0.02)
	assert.Less(t, fast["cross"]+fast["long_shot"], share["cross"]+share["long_shot"]-0.02)
	// Slow: the ball is kept and the cross is the one waited for, against a
	// defence that has had time to get set.
	assert.Greater(t, slowPass, pass+0.005)
	assert.Less(t, slow["take_on"]+slow["run"], share["take_on"]+share["run"]-0.02)
	assert.Greater(t, slow["cross"]+slow["long_shot"], share["cross"]+share["long_shot"]+0.02)
}

func TestSteps_WhoIsBehindABeatenDefender(t *testing.T) {
	lone := func(shape FormationType, tune func(l *GameLineup)) (*stepSide, *stepPlayer) {
		away := typicalStepLineup(shape, "a")
		if tune != nil {
			tune(&away)
		}
		_, a := stepSidesOf(t, typicalStepLineup(FormationTypeY, "h"), away)
		return a, a.lines[PlayerPositionDefense][0]
	}
	// Two defenders: the other one, arriving late.
	pyramid, first := lone(FormationTypePyramid, nil)
	cover, scale := behind(pyramid, first)
	require.NotNil(t, cover)
	assert.Equal(t, PlayerPositionDefense, cover.line)
	assert.NotEqual(t, first.ID, cover.ID)
	assert.Equal(t, tuning.StepsCoverScale, scale)

	// One defender and two midfielders: the midfielder who can tackle sits.
	diamond, only := lone(FormationTypeDiamond, func(l *GameLineup) {
		n := 0
		for i := range l.Players {
			if l.Players[i].SelectedPosition == PlayerPositionMidfield {
				if n == 1 {
					l.Players[i].Attributes.Tackling = 85
				}
				n++
			}
		}
	})
	sitter, scale := behind(diamond, only)
	require.NotNil(t, sitter)
	assert.Equal(t, PlayerPositionMidfield, sitter.line)
	assert.Equal(t, 85, sitter.Attributes.Tackling)
	assert.Equal(t, tuning.StepsScreenScale, scale)

	// One defender and one midfielder: nobody. Once beaten, he is beaten.
	y, alone := lone(FormationTypeY, nil)
	nobody, _ := behind(y, alone)
	assert.Nil(t, nobody)
}
