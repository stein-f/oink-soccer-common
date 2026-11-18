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
				// in behind or to cross past.
				a.Tackling, a.Strength = tackling, tackling
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
