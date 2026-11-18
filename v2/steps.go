package soccer

import (
	"math"
	"math/rand"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
)

// The step model (ModelSteps, season 19+).
//
// An attack is a short chain of steps, each one a contest between two named
// players:
//
//  1. a pass forward, against the opponent trying to cut it out;
//  2. what the side does with it: a forward takes his man on, runs in behind
//     the last defender, a cross is put in for a header, or someone shoots
//     from distance (a long ball from the back skips the pass: it goes over
//     the midfield for a forward to win against the defender on him);
//  3. the shot, against a defender trying to block it;
//  4. the shot on target, against the keeper.
//
// Who meets whom is decided by where they stand, not by lottery. Each line
// (defence, midfield, attack) is spread across the pitch in line-up order,
// and a contest is met by the nearest player of the defending line that
// faces it. A line that outnumbers the attackers in front of it has a spare
// man; one that is outnumbered is stretched; an empty line is covered from
// another, out of place. That is the whole of what a formation does here:
// no shape gets its own multipliers, and every shape has the same four
// outfielders to be an extra man somewhere and a man short somewhere else.
//
// Every shot is emitted as a goal or a miss, as in the other models, with
// the steps that led to it. A few attacks that end before a shot are emitted
// as stops; all of them are counted in the totals.

// stepPlayer is a player with his place on the pitch.
type stepPlayer struct {
	SelectedPlayer
	line PlayerPosition
	// lane runs from -1 (the left, as his side attacks) to +1 (the right).
	lane float64
	// state multiplies his odds: out of position, carrying an injury,
	// wearing the armband.
	state float64
}

// stepSide is everything about one team the steps read, worked out once.
type stepSide struct {
	team      TeamType
	lineup    GameLineup
	tactics   Tactics
	outfield  []*stepPlayer
	lines     map[PlayerPosition][]*stepPlayer
	keeper    *stepPlayer
	teamMod   float64 // captain, items and home advantage: every player's odds
	posBoosts positionMultipliers
	// legs is how much running the side has in it, against a typical side's
	// 1: what a high press asks for (see the tactics below).
	legs   float64
	totals *TeamTotals
}

func newStepSide(r *rand.Rand, team TeamType, lineup GameLineup, totals *TeamTotals) (*stepSide, []BoostRoll) {
	s := &stepSide{
		team:    team,
		lineup:  lineup,
		tactics: lineup.Team.Tactics,
		lines:   map[PlayerPosition][]*stepPlayer{},
		totals:  totals,
	}
	itemBoost, rolls := teamBoost(r, lineup)
	var posRolls []BoostRoll
	s.posBoosts, posRolls = rollPositionBoosts(r, lineup)
	s.teamMod = captainBoost(lineup) * itemBoost
	if team == TeamTypeHome {
		s.teamMod *= tuning.StepsHomeDuel
	}

	for _, p := range lineup.Players {
		sp := &stepPlayer{SelectedPlayer: p, line: p.SelectedPosition, state: injuryScale(p.Injury)}
		if p.IsOutOfPosition() {
			sp.state *= tuning.StepsOutOfPosition
		}
		if p.Role == PlayerRoleCaptain {
			sp.state *= tuning.CaptainSelfBoost(captainQuality(p.Attributes))
		}
		switch p.SelectedPosition {
		case PlayerPositionGoalkeeper:
			if s.keeper == nil {
				s.keeper = sp
				continue
			}
			sp.line = PlayerPositionDefense // a second keeper plays as a defender
		case PlayerPositionDefense, PlayerPositionMidfield, PlayerPositionAttack:
		default:
			sp.line = PlayerPositionMidfield
		}
		s.outfield = append(s.outfield, sp)
		s.lines[sp.line] = append(s.lines[sp.line], sp)
	}
	var workRate float64
	for _, p := range s.outfield {
		workRate += float64(p.Attributes.EffectiveWorkRate())
	}
	s.legs = 1
	if len(s.outfield) > 0 {
		s.legs = legs(workRate / float64(len(s.outfield)))
	}
	// Spread each line across the pitch in line-up order: one player holds
	// the centre, two take a side each, three a side each and the centre.
	for _, line := range s.lines {
		for i, p := range line {
			if len(line) > 1 {
				p.lane = -1 + 2*float64(i)/float64(len(line)-1)
			}
		}
	}
	return s, append(rolls, posRolls...)
}

// injuryFactor is what a side's own choices do to its players' chance of an
// injury: pressing high raises it, sitting off lowers it. (The opponent's
// aggression is counted separately.) Formations play no part. The classic
// and contest models multiply in the opponent's formation, which made a Box
// a little more bruising to play against; here no shape has a multiplier of
// any kind.
func (s *stepSide) injuryFactor() float64 {
	return pressInjuryFactor(s.tactics.Press)
}

// mod is everything that multiplies this player's odds in any contest.
func (s *stepSide) mod(p *stepPlayer) float64 {
	return math.Pow(s.teamMod*s.posBoosts.forPosition(p.SelectedPosition), tuning.StepsBoostExponent) * p.state
}

func simulateStepsMatch(r *rand.Rand, home, away GameLineup) MatchOutcome {
	tempoFactor := (tempoChanceFactor(home.Team.Tactics.Tempo) + tempoChanceFactor(away.Team.Tactics.Tempo)) / 2.0
	rng := tuning.StepsPossessions
	possessions := scaleChances(r, r.Intn(rng.Max-rng.Min+1)+rng.Min, tempoFactor)
	minutes := scheduleMinutes(r, possessions)

	totals := &MatchTotals{}
	h, homeRolls := newStepSide(r, TeamTypeHome, home, &totals.Home)
	a, awayRolls := newStepSide(r, TeamTypeAway, away, &totals.Away)

	m := &stepMatch{r: r}
	// The better side sees more of the ball (see StepsPossessionExponent),
	// the opponent's press and line take some of it away, and the home side
	// has a little more.
	homeBall := h.quality() * pressBall(a) * lineHeightControlFactor(a.tactics.LineHeight) * tuning.StepsHomePossession
	awayBall := a.quality() * pressBall(h) * lineHeightControlFactor(h.tactics.LineHeight)
	for _, minute := range minutes {
		hw := homeBall * stepsScoreEffect(m.homeGoals-m.awayGoals)
		aw := awayBall * stepsScoreEffect(m.awayGoals-m.homeGoals)
		if pickAttackingTeam(r, hw, aw) == TeamTypeHome {
			m.possession(h, a, minute)
		} else {
			m.possession(a, h, minute)
		}
	}
	m.capStops()

	homeInjuries := rollInjuries(r, home, teamAverageAggression(away), h.injuryFactor(), home.Team.ID)
	awayInjuries := rollInjuries(r, away, teamAverageAggression(home), a.injuryFactor(), away.Team.ID)
	return MatchOutcome{
		Events:   m.events,
		Injuries: Injuries{HomeTeamInjuries: homeInjuries, AwayTeamInjuries: awayInjuries},
		Boosts:   MatchBoosts{Home: homeRolls, Away: awayRolls},
		Totals:   totals,
	}
}

// passers is who plays a side's ball forward: its midfield, or its defence
// when it has none.
func (s *stepSide) passers() []*stepPlayer {
	return firstNonEmpty(s.lines[PlayerPositionMidfield], s.lines[PlayerPositionDefense], s.outfield)
}

// passOdds is a passer's odds of getting the ball forward against def: the
// contest the first step of every attack plays. long is a ball from the
// defence straight to the forwards.
func passOdds(passer *stepPlayer, att, def *stepSide, long bool, attackMod float64) (odds float64, opp *stepPlayer) {
	intercept := interceptScore
	opp, cover := facing(passer, att, def, PlayerPositionMidfield, intercept)
	if opp == nil {
		return math.Inf(1), nil
	}
	// A pass needs time: the man closing the passer down takes it away if his
	// side presses, and a high line leaves less room to play into. A side
	// playing fast gives the ball away more.
	mult := attackMod * squeeze(def, opp) * care(att) * passLineOdds(def.tactics.LineHeight)
	if long {
		mult *= tuning.StepsLongBall
	}
	if cover == tuning.StepsSupport || cover == tuning.StepsStretched {
		cover = 1 + (cover-1)*tuning.StepsMidfieldNumbers
	}
	return stepOdds(passScore(passer.Attributes, long), tuning.StepsInterceptScale*intercept(opp.Attributes), mult/(def.mod(opp)*cover)), opp
}

// quality is a side's possession weight: the mean overall rating of the
// players it fields, each as fit and as well placed as he is, sharpened by
// StepsPossessionExponent, with the team-wide boosts.
func (s *stepSide) quality() float64 {
	players := s.outfield
	if s.keeper != nil {
		players = append([]*stepPlayer{s.keeper}, players...)
	}
	if len(players) == 0 {
		return 1
	}
	var sum float64
	for _, p := range players {
		sum += overallOf(p.Attributes, p.line) * p.state
	}
	return math.Pow(sum/float64(len(players))/100, tuning.StepsPossessionExponent) *
		math.Pow(s.teamMod, tuning.StepsBoostExponent)
}

// overallOf is a player's overall rating, or for a player built without one
// (tests, hand-made line-ups) the core rating for the line he plays in.
func overallOf(a PlayerAttributes, line PlayerPosition) float64 {
	if a.OverallRating > 0 {
		return float64(a.OverallRating)
	}
	switch {
	case isGoalkeeper(a):
		return float64(a.GoalkeeperRating)
	case line == PlayerPositionDefense:
		return float64(a.DefenseRating)
	case line == PlayerPositionAttack:
		return float64(a.AttackRating)
	}
	return float64(a.ControlRating)
}

// passWeight is how often a player is the one who plays the pass.
func passWeight(p *stepPlayer) float64 {
	s := passScore(p.Attributes, false)
	w := s * s
	if p.Role == PlayerRolePlaymaker {
		w *= tuning.StepsPlaymakerPassWeight
	}
	return w
}

// stepsScoreEffect is the possession multiplier for a side leading by `lead`
// goals (negative when behind).
func stepsScoreEffect(lead int) float64 {
	shift := tuning.StepsScoreEffect * float64(lead)
	if shift > tuning.StepsScoreEffectCap {
		shift = tuning.StepsScoreEffectCap
	} else if shift < -tuning.StepsScoreEffectCap {
		shift = -tuning.StepsScoreEffectCap
	}
	return 1 - shift
}

type stepMatch struct {
	r                    *rand.Rand
	events               []GameEvent
	homeGoals, awayGoals int
	// What each side could try against the other. It depends only on who is
	// on the pitch, so it is worked out once.
	routeCache map[*stepSide][]stepRoute
}

func (m *stepMatch) lead(t TeamType) int {
	if t == TeamTypeHome {
		return m.homeGoals - m.awayGoals
	}
	return m.awayGoals - m.homeGoals
}

// stepOdds is the attacker's odds in one contest: his score against the
// defender's, sharpened by tuning.StepsDuelExponent, times the modifiers.
func stepOdds(a, b, mult float64) float64 {
	if a < 1 {
		a = 1
	}
	if b < 1 {
		b = 1
	}
	return math.Pow(a/b, tuning.StepsDuelExponent) * mult
}

func oddsToP(odds float64) float64 { return odds / (1 + odds) }

func (m *stepMatch) wins(odds float64) bool { return m.r.Float64() < oddsToP(odds) }

// --- What each contest compares ----------------------------------------------
//
// Plain blends of the attributes on a 0-100 scale. No skill curve: the
// sharpening is done once, in stepOdds.

func mix(sum, weights int) float64 { return float64(sum) / float64(weights) }

func passScore(a PlayerAttributes, long bool) float64 {
	if long {
		return mix(a.EffectivePassing()*2+a.EffectiveTechnique(), 3)
	}
	return float64(a.EffectivePassing())
}

func interceptScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveInterceptions()*2+a.EffectiveMarking(), 3)
}

// --- Tactics -----------------------------------------------------------------
//
// The press trades time for space. A side pressing high gives the man on the
// ball no time (his pass, his cross, his shot from distance) and leaves room
// behind for anyone who gets through (a take-on, a run in behind, a
// flick-on). A side sitting off does the reverse. Neither is better: it
// depends on what the opponent is good at, and on whether the pressing side
// can run (legs), because a press is only as good as the running in it and
// costs the side its own attack late in the match.
//
// Tempo is the attacking side's half of the same trade. Playing fast catches
// the defence before it is set, which is room to run into, and costs care on
// the ball; playing slow keeps the ball and picks the moment, against a
// defence that is waiting.

// legs turns a work rate into how much running there is in it, against the
// typical player's 1.
func legs(workRate float64) float64 {
	l := math.Pow(workRate/tuning.StepsTypicalWorkRate, tuning.StepsLegsExponent)
	return math.Max(tuning.StepsLegsLeast, math.Min(tuning.StepsLegsMost, l))
}

// squeeze is what the defending side's press does to the odds of an attacker
// who needs time on the ball. presser is the man closing him down.
func squeeze(def *stepSide, presser *stepPlayer) float64 {
	switch def.tactics.Press {
	case PressLevelHigh:
		l := def.legs
		if presser != nil {
			l = legs(float64(presser.Attributes.EffectiveWorkRate()))
		}
		return 1 / (1 + tuning.StepsPressSqueeze*l)
	case PressLevelLow:
		return tuning.StepsSitOffTime
	}
	return 1
}

// space is what the defending side's press does to the odds of an attacker
// who needs room to run into.
func space(def *stepSide) float64 {
	switch def.tactics.Press {
	case PressLevelHigh:
		return tuning.StepsPressSpace
	case PressLevelLow:
		return tuning.StepsSitOffSpace
	}
	return 1
}

// pressBall is what a side's press does to how much of the ball its opponent
// sees.
func pressBall(def *stepSide) float64 {
	switch def.tactics.Press {
	case PressLevelHigh:
		return 1 - tuning.StepsPressBall*def.legs
	case PressLevelLow:
		return 1 + tuning.StepsSitOffBall
	}
	return 1
}

// fade is what its own press has taken out of a side's attack by this minute.
func (s *stepSide) fade(minute int) float64 {
	if s.tactics.Press != PressLevelHigh {
		return 1
	}
	switch {
	case minute >= 75:
		return 1 - tuning.StepsPressFadeLate/s.legs
	case minute >= 60:
		return 1 - tuning.StepsPressFade/s.legs
	}
	return 1
}

// onTheBreak is what a side's tempo does to its odds where it needs room: a
// take-on, a run in behind.
func onTheBreak(att *stepSide) float64 {
	switch att.tactics.Tempo {
	case TempoLevelFast:
		return tuning.StepsFastBreak
	case TempoLevelSlow:
		return tuning.StepsSlowBreak
	}
	return 1
}

// care is what a side's tempo does to its odds where it needs care on the
// ball: a pass, the cross a header is won from, a shot from distance.
func care(att *stepSide) float64 {
	switch att.tactics.Tempo {
	case TempoLevelFast:
		return tuning.StepsFastCare
	case TempoLevelSlow:
		return tuning.StepsSlowCare
	}
	return 1
}

// Pace is in four of these blends, on both sides of the ball, and it tracks a
// player's overall rating less than any other attribute. Its weights are kept
// modest so that a quick, ordinary player isn't worth more than a good one
// (the attribute report in cmd/calibrate is the check).

func takeOnScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveDribbling()*3+a.EffectivePace(), 4)
}

// soloScore is a run from deep with the ball: it has to be carried a long
// way at speed, and held on to.
func soloScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveDribbling()*2+a.EffectivePace()*2+a.EffectiveStrength(), 5)
}

// poachScore is being where a loose ball drops, and putting it away.
func poachScore(a PlayerAttributes) float64 {
	return mix(a.EffectivePositioning()*2+a.EffectiveFinishing(), 3)
}

// worldieChance is how likely a goal from distance is to be one people talk
// about: struck with real power, by someone with the technique to keep it
// down. It changes nothing about whether the shot goes in.
func worldieChance(a PlayerAttributes) float64 {
	s := (mix(a.EffectiveShotPower()*2+a.EffectiveTechnique(), 3) - tuning.StepsWorldieFrom) / (100 - tuning.StepsWorldieFrom)
	if s <= 0 {
		return 0
	}
	return math.Min(1, s*s*tuning.StepsWorldieShare)
}

func tackleScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveTackling()*3+a.EffectivePace()+a.EffectiveStrength(), 5)
}

func runScore(a PlayerAttributes) float64 {
	return mix(a.EffectivePace()+a.EffectivePositioning(), 2)
}

func chaseScore(a PlayerAttributes) float64 {
	return mix(a.EffectivePace()+a.EffectiveMarking(), 2)
}

func crossScore(a PlayerAttributes) float64 { return float64(a.EffectiveCrossing()) }

func markScore(a PlayerAttributes) float64 { return float64(a.EffectiveMarking()) }

func aerialScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveHeading()+a.EffectiveStrength(), 2)
}

// holdUpScore is winning a long ball with a man on your back, or being that
// man: strength first, then getting a head to it.
func holdUpScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveStrength()*2+a.EffectiveHeading(), 3)
}

// blockScore is getting something in the way of a shot: standing in the right
// place, and the tackle made as it is struck.
func blockScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveMarking()+a.EffectiveTackling(), 2)
}

// shotScore is the shooter's side of the shot, by the kind of chance.
func shotScore(a PlayerAttributes, ct ChanceType) float64 {
	switch ct {
	case ChanceTypeCross, ChanceTypeCorner: // a header
		// Winning it in the air is its own contest (aerialScore); this is
		// putting it on target.
		return mix(a.EffectiveHeading()+a.EffectiveFinishing()+a.AttackRating, 3)
	case ChanceTypeLongRange:
		return mix(a.EffectiveShotPower()+a.EffectiveTechnique()+a.AttackRating, 3)
	case ChanceTypeFreeKick:
		return mix(a.EffectiveTechnique()*2+a.EffectiveShotPower(), 3)
	case ChanceTypePenalty:
		return mix(a.EffectiveComposure()*2+a.EffectiveFinishing(), 3)
	case ChanceTypeGoalKeeperShot: // through on goal
		return mix(a.EffectiveFinishing()*2+a.EffectiveComposure(), 3)
	default:
		return mix(a.EffectiveFinishing()*2+a.AttackRating+a.EffectiveComposure(), 4)
	}
}

// keeperScore is the keeper's side of a shot on target.
func keeperScore(a PlayerAttributes, ct ChanceType) float64 {
	switch ct {
	case ChanceTypeGoalKeeperShot:
		return float64(a.EffectiveOneOnOne())
	case ChanceTypeLongRange, ChanceTypeFreeKick:
		return mix(a.EffectiveReflexes()+a.EffectiveHandling(), 2)
	default:
		return float64(a.EffectiveReflexes())
	}
}

// --- Who meets whom ----------------------------------------------------------

// facing finds the defender who meets attacker a from the given line, and how
// well placed he is to do it (a multiplier on his odds). It is nil when the
// defending side has nobody to send.
func facing(a *stepPlayer, att, def *stepSide, line PlayerPosition, score func(PlayerAttributes) float64) (*stepPlayer, float64) {
	cands := def.lines[line]
	cover := 1.0
	if len(cands) == 0 {
		// Nobody in that line: it is covered from whichever other line is
		// better at the job, out of its place. A side with no midfield has
		// a defender step up to cut out a pass rather than send a forward
		// back to do it badly.
		bestMean := -1.0
		for _, l := range []PlayerPosition{PlayerPositionDefense, PlayerPositionMidfield, PlayerPositionAttack} {
			others := def.lines[l]
			if l == line || len(others) == 0 {
				continue
			}
			var sum float64
			for _, c := range others {
				sum += score(c.Attributes)
			}
			if mean := sum / float64(len(others)); mean > bestMean {
				bestMean, cands = mean, others
			}
		}
		if len(cands) == 0 {
			return nil, 1
		}
		cover = tuning.StepsLineCover
	} else if n := len(att.lines[a.line]); len(cands) > n {
		cover = tuning.StepsSupport
	} else if len(cands) < n {
		cover = tuning.StepsStretched
	}

	// The nearest man. His lanes are counted from his own side's view, so
	// the attacker's left wing faces his right.
	var best *stepPlayer
	bestDist := math.Inf(1)
	for _, c := range cands {
		dist := math.Abs(a.lane + c.lane)
		if dist < bestDist || (dist == bestDist && score(c.Attributes) > score(best.Attributes)) {
			best, bestDist = c, dist
		}
	}
	// A Ball Winner makes up some of the ground a stretched line is short.
	if cover == tuning.StepsStretched && best.Role == PlayerRoleBallWinner {
		cover += (1 - cover) * tuning.StepsBallWinnerCover
	}
	return best, cover
}

// coverFor is the defender who covers when a team-mate in his line is beaten:
// the other man of a two-man line (the nearest other, in a longer one). nil
// when the beaten man was the only one, or wasn't playing in his own line.
func coverFor(def *stepSide, beaten *stepPlayer) *stepPlayer {
	if beaten == nil {
		return nil
	}
	var cover *stepPlayer
	for _, p := range def.lines[beaten.line] {
		if p == beaten {
			continue
		}
		if cover == nil || math.Abs(p.lane-beaten.lane) < math.Abs(cover.lane-beaten.lane) {
			cover = p
		}
	}
	if beaten.line != PlayerPositionDefense {
		return nil
	}
	return cover
}

// behind is who an attacker still has to beat once he is past the defender
// who met him, and how much that man's tackle counts: the other defender of
// a two-man line, arriving late (StepsCoverScale), or, for a side with one
// defender and two or more midfielders, the midfielder who sits in front of
// him (StepsScreenScale). The one who sits is its best tackler. nil when
// there is nobody: a lone defender with one midfielder ahead of him, once
// beaten, is beaten.
func behind(def *stepSide, beaten *stepPlayer) (*stepPlayer, float64) {
	if cover := coverFor(def, beaten); cover != nil {
		return cover, tuning.StepsCoverScale
	}
	midfield := def.lines[PlayerPositionMidfield]
	if beaten == nil || beaten.line != PlayerPositionDefense || len(midfield) < 2 {
		return nil, 0
	}
	sitter := midfield[0]
	for _, p := range midfield[1:] {
		if tackleScore(p.Attributes) > tackleScore(sitter.Attributes) {
			sitter = p
		}
	}
	return sitter, tuning.StepsScreenScale
}

// pick chooses one player in proportion to weight, in slice order.
func (m *stepMatch) pick(players []*stepPlayer, weight func(*stepPlayer) float64) *stepPlayer {
	var total float64
	weights := make([]float64, len(players))
	for i, p := range players {
		if w := weight(p); w > 0 {
			weights[i] = w
			total += w
		}
	}
	if total <= 0 {
		return nil
	}
	roll := m.r.Float64() * total
	for i, w := range weights {
		if roll < w {
			return players[i]
		}
		roll -= w
	}
	return players[len(players)-1]
}

func without(players []*stepPlayer, skip ...*stepPlayer) []*stepPlayer {
	out := make([]*stepPlayer, 0, len(players))
next:
	for _, p := range players {
		for _, s := range skip {
			if p == s {
				continue next
			}
		}
		out = append(out, p)
	}
	return out
}

// firstNonEmpty returns the first group that has anyone in it.
func firstNonEmpty(groups ...[]*stepPlayer) []*stepPlayer {
	for _, g := range groups {
		if len(g) > 0 {
			return g
		}
	}
	return nil
}

func squared(score func(PlayerAttributes) float64) func(*stepPlayer) float64 {
	return func(p *stepPlayer) float64 {
		s := score(p.Attributes)
		return s * s
	}
}

// --- The attack --------------------------------------------------------------

// stepRoute is one thing a side could try once it has the ball: who would do
// it, and how often.
type stepRoute struct {
	kind   string      // take_on | solo | run | cross | long_ball | long_shot
	actor  *stepPlayer // who does it (a long ball: who it is hit to)
	target *stepPlayer // cross only: who it is aimed at
	weight float64
}

// stepOption is one player's claim on a kind of attack.
type stepOption struct {
	actor, target *stepPlayer
	// suits is how good he is at it, a 0-100 score. works is how likely it
	// is to come off against the men he would meet.
	suits, works float64
	// bias is what his place and role add: a wide player crosses more, a
	// Target Man is the one aimed at.
	bias float64
}

// pull is how strongly an option draws the ball, among the players who could
// do the same thing.
func (o stepOption) pull() float64 {
	return o.bias * math.Pow(o.suits/tuning.StepsTypicalScore, tuning.StepsAptitude) * math.Pow(o.works, tuning.StepsChoiceByOdds)
}

// spread shares one kind of attack out between the players who could do it.
// Who does it follows pull: what suits the player, and whether it would work
// against the man in front of him. How often the side does it at all follows
// the same two things, so a side of quick, tricky forwards runs at people and
// a side of big ones crosses, and nobody slow dribbles unless the defender is
// worse.
func spread(out []stepRoute, kind string, share float64, options []stepOption) []stepRoute {
	var total float64
	for _, o := range options {
		total += o.pull()
	}
	if total <= 0 {
		return out
	}
	for _, o := range options {
		w := tuning.StepsRouteWeights[kind] * share * o.pull() / total *
			math.Pow(o.works, tuning.StepsOddsMix) * math.Pow(o.suits/tuning.StepsTypicalScore, tuning.StepsAptitudeMix)
		if w > 0 {
			out = append(out, stepRoute{kind: kind, actor: o.actor, target: o.target, weight: w})
		}
	}
	return out
}

// attackMod is a player's odds going forward: his own, less the easing off of
// a side that is comfortably ahead.
func (m *stepMatch) attackMod(att *stepSide, p *stepPlayer) float64 {
	return att.mod(p) * easeOff(m.lead(att.team))
}

// defendMod is a defender's odds in his own third.
func defendMod(def *stepSide, p *stepPlayer, cover float64) float64 {
	return def.mod(p) * cover
}

// passLineOdds is what the defending line's height does to the pass forward.
func passLineOdds(line LineHeight) float64 {
	switch line {
	case LineHeightHigh:
		return tuning.StepsPassVsHighLine
	case LineHeightDeep:
		return tuning.StepsPassVsDeepLine
	default:
		return 1
	}
}

// routes weighs up everything the side could try: each kind of attack, by
// each player who could make it.
func (m *stepMatch) routes(att, def *stepSide) []stepRoute {
	if cached, ok := m.routeCache[att]; ok {
		return cached
	}
	// The forwards do most of the attacking, with a midfielder joining in
	// now and then. A side with neither sends whoever it has.
	attack, midfield := att.lines[PlayerPositionAttack], att.lines[PlayerPositionMidfield]
	if len(attack)+len(midfield) == 0 {
		attack = att.outfield
	}
	// The forwards make most of a side's take-ons and runs between them and
	// its midfielders the rest (StepsMidfieldAttackShare), whatever the
	// numbers in each line: a lone striker carries as much of his side's
	// attack as two strikers carry of theirs.
	attackShare, midfieldShare := 1.0, 1.0
	if len(attack) > 0 && len(midfield) > 0 {
		attackShare, midfieldShare = 1-tuning.StepsMidfieldAttackShare, tuning.StepsMidfieldAttackShare
	}
	// timed is for a contest where the attacker needs time on the ball, and
	// the man he meets is the one closing him down.
	against := func(p *stepPlayer, line PlayerPosition, mine float64, theirs func(PlayerAttributes) float64, scale, mult float64, timed bool) float64 {
		opp, cover := facing(p, att, def, line, theirs)
		if opp == nil {
			return 1
		}
		if line == PlayerPositionMidfield && (cover == tuning.StepsSupport || cover == tuning.StepsStretched) {
			cover = 1 + (cover-1)*tuning.StepsMidfieldNumbers
		}
		if timed {
			mult *= squeeze(def, opp)
		}
		return oddsToP(stepOdds(mine, scale*theirs(opp.Attributes), mult/cover))
	}
	// What the two sides' tactics do to an attack that needs room.
	room := space(def) * onTheBreak(att)
	options := func(players []*stepPlayer, option func(*stepPlayer) stepOption) []stepOption {
		out := make([]stepOption, 0, len(players))
		for _, p := range players {
			out = append(out, option(p))
		}
		return out
	}

	takeOn := func(p *stepPlayer) stepOption {
		s := takeOnScore(p.Attributes)
		return stepOption{actor: p, suits: s, bias: 1,
			works: against(p, PlayerPositionDefense, s, tackleScore, tuning.StepsTackleScale, room, false)}
	}
	runWeight := 1.0
	switch def.tactics.LineHeight {
	case LineHeightHigh:
		runWeight = tuning.StepsRunVsHighLine
	case LineHeightDeep:
		runWeight = tuning.StepsRunVsDeepLine
	}
	run := func(p *stepPlayer) stepOption {
		s := runScore(p.Attributes)
		return stepOption{actor: p, suits: s, bias: 1,
			works: against(p, PlayerPositionDefense, s, chaseScore, tuning.StepsChaseScale, runLineOdds(def.tactics.LineHeight)*room, false)}
	}
	// A run from deep has a midfielder to get past before the defence.
	solo := func(p *stepPlayer) stepOption {
		s := soloScore(p.Attributes)
		return stepOption{actor: p, suits: s, bias: 1,
			works: against(p, PlayerPositionMidfield, s, tackleScore, tuning.StepsTackleScale, onTheBreak(att), true) *
				against(p, PlayerPositionDefense, s, tackleScore, tuning.StepsTackleScale, room, false)}
	}

	var out []stepRoute
	out = spread(out, "take_on", attackShare, options(attack, takeOn))
	out = spread(out, "take_on", midfieldShare, options(midfield, takeOn))
	out = spread(out, "run", attackShare*runWeight, options(attack, run))
	out = spread(out, "run", midfieldShare*runWeight, options(midfield, run))
	advanced := append(append([]*stepPlayer{}, attack...), midfield...)
	out = spread(out, "solo", 1, options(advanced, solo))

	// A cross needs someone to put it in and someone else to attack it.
	var crosses []stepOption
	for _, crosser := range advanced {
		for _, target := range advanced {
			if target == crosser {
				continue
			}
			bias := 1.0
			if crosser.lane != 0 {
				bias *= 1.5 // wide players do most of the crossing
			}
			if target.Role == PlayerRoleTargetMan {
				bias *= tuning.StepsTargetManWeight
			}
			if target.line == PlayerPositionAttack {
				bias *= 2 // forwards are in the box; a midfielder has to arrive
			}
			cross, header := crossScore(crosser.Attributes), aerialScore(target.Attributes)
			crosses = append(crosses, stepOption{actor: crosser, target: target, bias: bias,
				suits: math.Sqrt(cross * header),
				works: against(crosser, PlayerPositionDefense, cross, markScore, tuning.StepsCloseDownScale, 1, true) *
					against(target, PlayerPositionDefense, header, aerialScore, tuning.StepsMarkerScale, care(att), false)})
		}
	}
	out = spread(out, "cross", 1, crosses)

	// A long ball from the back, for a forward to win with a defender on him.
	// It misses out the midfield: a side with none goes long most, and so
	// does one being pressed high, because it is the press it goes over.
	longShare := 1.0
	if len(midfield) > 0 {
		longShare = tuning.StepsLongBallWithMidfield
	}
	switch def.tactics.Press {
	case PressLevelHigh:
		longShare *= tuning.StepsLongBallVsHighPress
	case PressLevelLow:
		longShare *= tuning.StepsLongBallVsLowPress
	}
	out = spread(out, "long_ball", longShare, options(firstNonEmpty(attack, midfield), func(p *stepPlayer) stepOption {
		s := holdUpScore(p.Attributes)
		bias := 1.0
		if p.Role == PlayerRoleTargetMan {
			bias = tuning.StepsTargetManWeight
		}
		return stepOption{actor: p, suits: s, bias: bias,
			works: against(p, PlayerPositionDefense, s, holdUpScore, tuning.StepsHighBallScale, 1, false)}
	}))

	shooters := firstNonEmpty(midfield, attack, att.outfield)
	cfg := tuning.StepsShots[string(ChanceTypeLongRange)]
	out = spread(out, "long_shot", 1, options(shooters, func(p *stepPlayer) stepOption {
		s := shotScore(p.Attributes, ChanceTypeLongRange)
		return stepOption{actor: p, suits: s, bias: 1, works: oddsToP(stepOdds(s, cfg.OnTarget, squeeze(def, nil)*care(att)))}
	}))

	if m.routeCache == nil {
		m.routeCache = map[*stepSide][]stepRoute{}
	}
	m.routeCache[att] = out
	return out
}

func runLineOdds(line LineHeight) float64 {
	switch line {
	case LineHeightHigh:
		return tuning.StepsRunOddsVsHighLine
	case LineHeightDeep:
		return tuning.StepsRunOddsVsDeepLine
	default:
		return 1
	}
}

// passer picks who plays the ball forward to the players the route involves:
// a midfielder if there is one free to, otherwise a defender, hitting it long.
func (m *stepMatch) passer(att *stepSide, route stepRoute) *stepPlayer {
	for _, group := range [][]*stepPlayer{att.lines[PlayerPositionMidfield], att.lines[PlayerPositionDefense], att.outfield} {
		if p := m.pick(without(group, route.actor, route.target), passWeight); p != nil {
			return p
		}
	}
	return nil
}

// possession plays one attack by att against def.
func (m *stepMatch) possession(att, def *stepSide, minute int) {
	att.totals.Possessions++
	options := m.routes(att, def)
	if len(options) == 0 {
		return
	}
	roll, total := 0.0, 0.0
	for _, o := range options {
		total += o.weight
	}
	route := options[len(options)-1]
	roll = m.r.Float64() * total
	for _, o := range options {
		if roll < o.weight {
			route = o
			break
		}
		roll -= o.weight
	}
	m.attack(att, def, route, minute)
}

// attack plays one attack the way the side chose to make it.
func (m *stepMatch) attack(att, def *stepSide, route stepRoute, minute int) {
	if route.kind == "long_ball" {
		m.longBall(att, def, route.actor, minute)
		return
	}
	var chain []Step
	momentum := 1.0
	assist := ""

	// 1. The ball forward. A run from deep starts with the man himself.
	if passer := m.passer(att, route); passer != nil && route.kind != "solo" {
		long := passer.line == PlayerPositionDefense && route.actor.line == PlayerPositionAttack
		detail := "short"
		switch {
		case route.kind == "run":
			detail = "through"
		case long:
			detail = "long"
		}
		step := Step{Kind: StepKindPass, ActorID: passer.ID, Won: true, Detail: detail}
		if odds, opp := passOdds(passer, att, def, long, m.attackMod(att, passer)); opp != nil {
			if route.kind == "run" {
				// The ball in behind is the hard one to play.
				odds *= tuning.StepsThroughBall
			}
			step.OpponentID = opp.ID
			step.Won = m.wins(odds)
		}
		att.totals.Passes++
		if !step.Won {
			def.totals.Interceptions++
			return
		}
		att.totals.PassesCompleted++
		chain = append(chain, step)
		momentum = tuning.StepsMomentum
		assist = passer.ID
	}

	// 2. What the side does with it.
	actor := route.actor
	switch route.kind {
	case "take_on":
		m.dribble(att, def, actor, chain, momentum, false, minute)

	case "solo":
		// He picks it up deep and goes on his own: first past the midfielder
		// who comes to meet him, then at the defence.
		step := Step{Kind: StepKindTakeOn, ActorID: actor.ID, Won: true, Detail: "solo"}
		if opp, cover := facing(actor, att, def, PlayerPositionMidfield, tackleScore); opp != nil {
			if cover == tuning.StepsSupport || cover == tuning.StepsStretched {
				cover = 1 + (cover-1)*tuning.StepsMidfieldNumbers
			}
			step.OpponentID = opp.ID
			// In midfield he is the one being pressed.
			mult := m.attackMod(att, actor) * onTheBreak(att) * squeeze(def, opp) / defendMod(def, opp, cover)
			step.Won = m.wins(stepOdds(soloScore(actor.Attributes), tuning.StepsTackleScale*tackleScore(opp.Attributes), mult))
		}
		att.totals.TakeOns++
		if !step.Won {
			// Dispossessed in midfield: nothing to show for it.
			def.totals.Tackles++
			return
		}
		att.totals.TakeOnsWon++
		m.dribble(att, def, actor, []Step{step}, tuning.StepsMomentum, true, minute)

	case "run":
		step := Step{Kind: StepKindRun, ActorID: actor.ID, Won: true}
		opp, cover := facing(actor, att, def, PlayerPositionDefense, chaseScore)
		if opp != nil {
			step.OpponentID = opp.ID
			mult := m.attackMod(att, actor) * momentum * runLineOdds(def.tactics.LineHeight) * space(def) * onTheBreak(att) / (def.mod(opp) * cover)
			step.Won = m.wins(stepOdds(runScore(actor.Attributes), tuning.StepsChaseScale*chaseScore(opp.Attributes), mult))
		}
		chain = append(chain, step)
		// In behind the last man, unless someone gets across. He isn't
		// racing him any more: he has to make the tackle.
		if cover, scale := behind(def, opp); step.Won && cover != nil {
			mult := m.attackMod(att, actor) * runLineOdds(def.tactics.LineHeight) * space(def) * onTheBreak(att) / def.mod(cover)
			second := Step{Kind: StepKindRun, ActorID: actor.ID, OpponentID: cover.ID,
				Won: m.wins(stepOdds(runScore(actor.Attributes), scale*tackleScore(cover.Attributes), mult))}
			chain = append(chain, second)
			step, opp = second, cover
		}
		if step.Won {
			m.shot(att, def, actor, ChanceTypeGoalKeeperShot, chain, tuning.StepsMomentum, assist, minute, 0, "")
			return
		}
		def.totals.Tackles++
		m.stop(def, opp, actor, "caught", chain, minute)

	case "cross":
		step := Step{Kind: StepKindCross, ActorID: actor.ID, Won: true}
		if opp, cover := facing(actor, att, def, PlayerPositionDefense, markScore); opp != nil {
			step.OpponentID = opp.ID
			mult := m.attackMod(att, actor) * momentum * squeeze(def, opp) / defendMod(def, opp, cover)
			step.Won = m.wins(stepOdds(crossScore(actor.Attributes), tuning.StepsCloseDownScale*markScore(opp.Attributes), mult))
		}
		att.totals.Crosses++
		chain = append(chain, step)
		if !step.Won {
			m.cleared(att, def, minute, 0)
			return
		}
		att.totals.CrossesCompleted++
		marker, cover := facing(route.target, att, def, PlayerPositionDefense, aerialScore)
		// The delivery counts, as it does from a corner: a good cross is
		// harder to defend, and so is one a patient side waited for.
		cover /= tuning.CornerDeliveryFactor(actor.Attributes.EffectiveCrossing()) * care(att)
		m.aerial(att, def, route.target, marker, cover, ChanceTypeCross, chain, actor.ID, minute, 0)

	case "long_shot":
		m.shot(att, def, actor, ChanceTypeLongRange, chain, momentum, assist, minute, 0, "")
	}
}

// dribble is a player running at the defence with the ball: past the man who
// meets him, then whoever covers. A solo run from deep he finishes himself;
// otherwise, once through, he squares it if a team-mate is the better bet.
func (m *stepMatch) dribble(att, def *stepSide, actor *stepPlayer, chain []Step, momentum float64, solo bool, minute int) {
	score, detail, how := takeOnScore(actor.Attributes), "", ""
	if solo {
		score, detail, how = soloScore(actor.Attributes), "solo", "solo"
	}
	step := Step{Kind: StepKindTakeOn, ActorID: actor.ID, Won: true, Detail: detail}
	opp, cover := facing(actor, att, def, PlayerPositionDefense, tackleScore)
	if opp != nil {
		step.OpponentID = opp.ID
		mult := m.attackMod(att, actor) * momentum * space(def) * onTheBreak(att) / defendMod(def, opp, cover)
		step.Won = m.wins(stepOdds(score, tuning.StepsTackleScale*tackleScore(opp.Attributes), mult))
	}
	att.totals.TakeOns++
	chain = append(chain, step)
	// Past his man, he still has to get past the one covering.
	if cover, scale := behind(def, opp); step.Won && cover != nil {
		att.totals.TakeOnsWon++
		mult := m.attackMod(att, actor) * space(def) * onTheBreak(att) / defendMod(def, cover, 1)
		second := Step{Kind: StepKindTakeOn, ActorID: actor.ID, OpponentID: cover.ID, Detail: detail,
			Won: m.wins(stepOdds(score, scale*tackleScore(cover.Attributes), mult))}
		att.totals.TakeOns++
		chain = append(chain, second)
		step, opp = second, cover
	}
	if !step.Won {
		m.tackled(att, def, actor, opp, chain, minute)
		return
	}
	att.totals.TakeOnsWon++
	if !solo {
		// Through, he squares it or goes alone. How often depends on who is
		// waiting: a poor finisher looks for a better one, a good one
		// shoots. Going alone makes the chance his own: no assist.
		mates := without(append(append([]*stepPlayer{}, att.lines[PlayerPositionAttack]...), att.lines[PlayerPositionMidfield]...), actor)
		if mate := m.pick(mates, squared(func(a PlayerAttributes) float64 { return shotScore(a, ChanceTypeOpenPlay) })); mate != nil {
			ratio := shotScore(mate.Attributes, ChanceTypeOpenPlay) / math.Max(1, shotScore(actor.Attributes, ChanceTypeOpenPlay))
			share := tuning.StepsCutBack * math.Min(tuning.StepsCutBackMost, math.Max(tuning.StepsCutBackLeast, ratio*ratio))
			if m.r.Float64() < share {
				m.shot(att, def, mate, ChanceTypeOpenPlay, chain, tuning.StepsMomentum, actor.ID, minute, 0, "")
				return
			}
		}
	}
	// A wide player who goes alone has come inside to do it: the winger
	// cutting in to shoot, where squaring it was the run to the byline.
	if !solo && actor.lane != 0 {
		for i := range chain {
			if chain[i].Kind == StepKindTakeOn && chain[i].ActorID == actor.ID {
				chain[i].Detail = "cut_in"
			}
		}
	}
	m.shot(att, def, actor, ChanceTypeOpenPlay, chain, tuning.StepsMomentum, "", minute, 0, how)
}

// longBall is a ball hit from the back over the midfield, for a forward to
// win against the defender on him. Nobody cuts it out on the way: it is all
// decided where it drops. If he wins it he flicks it on for a team-mate
// running beyond him (his strike partner, or a midfielder if he is up there
// alone), who has only the covering defender to beat. Now and then he knocks
// it down instead for a midfielder to hit, and with nobody near him he
// brings it down and shoots.
func (m *stepMatch) longBall(att, def *stepSide, target *stepPlayer, minute int) {
	back := without(att.lines[PlayerPositionDefense], target)
	if len(back) == 0 && att.keeper != nil {
		back = []*stepPlayer{att.keeper}
	}
	passer := m.pick(firstNonEmpty(back, without(att.outfield, target)), func(p *stepPlayer) float64 {
		s := passScore(p.Attributes, true)
		return s * s
	})
	if passer == nil {
		return
	}
	chain := []Step{{Kind: StepKindPass, ActorID: passer.ID, Won: true, Detail: "high"}}
	att.totals.Passes++

	step := Step{Kind: StepKindAerial, ActorID: target.ID, Won: true}
	marker, cover := facing(target, att, def, PlayerPositionDefense, holdUpScore)
	if marker != nil {
		step.OpponentID = marker.ID
		// The ball counts: one hit well is one he can attack.
		delivery := tuning.CornerDeliveryFactor(int(passScore(passer.Attributes, true)))
		mult := m.attackMod(att, target) * delivery / defendMod(def, marker, cover)
		step.Won = m.wins(stepOdds(holdUpScore(target.Attributes), tuning.StepsHighBallScale*holdUpScore(marker.Attributes), mult))
	}
	if !step.Won {
		def.totals.AerialsWon++
		def.totals.Clearances++
		m.stop(def, marker, target, "header", append(chain, step), minute)
		return
	}
	att.totals.PassesCompleted++
	att.totals.AerialsWon++

	// Knocked down for a midfielder coming on to it.
	arriving := without(att.lines[PlayerPositionMidfield], target)
	if mate := m.pick(arriving, squared(func(a PlayerAttributes) float64 { return shotScore(a, ChanceTypeOpenPlay) })); mate != nil && m.r.Float64() < tuning.StepsKnockDown {
		step.Detail = "knock_down"
		m.shot(att, def, mate, ChanceTypeOpenPlay, append(chain, step), tuning.StepsMomentum, target.ID, minute, 0, "")
		return
	}
	// Flicked on: for the one who reads it and is quick enough.
	beyond := firstNonEmpty(without(att.lines[PlayerPositionAttack], target), arriving)
	if runner := m.pick(beyond, func(p *stepPlayer) float64 {
		return math.Pow(runScore(p.Attributes)/tuning.StepsTypicalScore, tuning.StepsAptitude)
	}); runner != nil {
		step.Detail = "flick_on"
		run := Step{Kind: StepKindRun, ActorID: runner.ID, Won: true}
		// The other defender covers. A side with only one has a midfielder
		// drop in: the ball was in the air long enough for him to get back.
		second, scale := coverFor(def, marker), tuning.StepsCoverScale
		if second == nil && len(def.lines[PlayerPositionMidfield]) > 0 {
			second, _ = facing(runner, att, def, PlayerPositionMidfield, tackleScore)
			scale = tuning.StepsTrackBackScale
		}
		if second != nil {
			// It went over the press: whatever room that left is his.
			mult := m.attackMod(att, runner) * runLineOdds(def.tactics.LineHeight) * space(def) / def.mod(second)
			run.OpponentID = second.ID
			run.Won = m.wins(stepOdds(runScore(runner.Attributes), scale*tackleScore(second.Attributes), mult))
		}
		chain = append(chain, step, run)
		if !run.Won {
			def.totals.Tackles++
			m.stop(def, second, runner, "caught", chain, minute)
			return
		}
		m.shot(att, def, runner, ChanceTypeGoalKeeperShot, chain, tuning.StepsMomentum, target.ID, minute, 0, "")
		return
	}
	step.Detail = "brought_down"
	m.shot(att, def, target, ChanceTypeOpenPlay, append(chain, step), tuning.StepsMomentum, passer.ID, minute, 0, "")
}

// tackled is a take-on the defender won: cleanly, which ends the attack, or
// with a foul, which may give a free kick in range or a penalty.
func (m *stepMatch) tackled(att, def *stepSide, actor, opp *stepPlayer, chain []Step, minute int) {
	// A good tackler takes the ball; a poor one takes the man as well.
	fouls := foulChance(opp.SelectedPlayer, def.tactics.Press) *
		math.Pow(tuning.StepsTypicalScore/math.Max(1, float64(opp.Attributes.EffectiveTackling())), tuning.StepsCleanTackle)
	if opp.Role == PlayerRoleBallWinner {
		fouls *= tuning.StepsBallWinnerFoul
	}
	fouls = math.Min(fouls, tuning.ContestFoulCap)
	if m.r.Float64() >= fouls {
		def.totals.Tackles++
		m.stop(def, opp, actor, "tackle", chain, minute)
		return
	}
	def.totals.Fouls++
	roll := m.r.Float64()
	switch {
	case roll < tuning.StepsPenaltyShareOfFouls:
		chain = append(chain, Step{Kind: StepKindFoul, ActorID: actor.ID, OpponentID: opp.ID, Won: true, Detail: "penalty"})
		if taker := m.taker(att, ChanceTypePenalty); taker != nil {
			m.shot(att, def, taker, ChanceTypePenalty, chain, 1, "", minute, 0, "")
		}
	case roll < tuning.StepsPenaltyShareOfFouls+tuning.StepsFreeKickShareOfFouls:
		chain = append(chain, Step{Kind: StepKindFoul, ActorID: actor.ID, OpponentID: opp.ID, Won: true, Detail: "free_kick"})
		if taker := m.taker(att, ChanceTypeFreeKick); taker != nil {
			m.shot(att, def, taker, ChanceTypeFreeKick, chain, 1, "", minute, 0, "")
		}
	}
}

// taker is the named set-piece taker if he is on the pitch, otherwise the
// best player for that kind of kick.
func (m *stepMatch) taker(att *stepSide, ct ChanceType) *stepPlayer {
	var best *stepPlayer
	for _, p := range att.outfield {
		if p.ID == att.tactics.SetPieceTaker {
			return p
		}
		if best == nil || shotScore(p.Attributes, ct) > shotScore(best.Attributes, ct) {
			best = p
		}
	}
	return best
}

// aerial is two players going for a ball in the box.
func (m *stepMatch) aerial(att, def *stepSide, target, marker *stepPlayer, cover float64, ct ChanceType, chain []Step, assist string, minute, depth int) {
	step := Step{Kind: StepKindAerial, ActorID: target.ID, Won: true}
	if marker != nil {
		step.OpponentID = marker.ID
		mult := m.attackMod(att, target) / defendMod(def, marker, cover)
		step.Won = m.wins(stepOdds(aerialScore(target.Attributes), tuning.StepsMarkerScale*aerialScore(marker.Attributes), mult))
	}
	chain = append(chain, step)
	if !step.Won {
		def.totals.AerialsWon++
		m.cleared(att, def, minute, depth)
		return
	}
	att.totals.AerialsWon++
	m.shot(att, def, target, ct, chain, tuning.StepsMomentum, assist, minute, depth, "")
}

// cleared is the defending side getting the ball away from its box, which
// sometimes means behind for a corner.
func (m *stepMatch) cleared(att, def *stepSide, minute, depth int) {
	def.totals.Clearances++
	m.maybeCorner(att, def, tuning.StepsCornerAfterClearance, minute, depth)
}

func (m *stepMatch) maybeCorner(att, def *stepSide, chance float64, minute, depth int) {
	if depth < tuning.StepsMaxFollowUps && m.r.Float64() < chance {
		m.corner(att, def, minute, depth+1)
	}
}

// corner is a delivery by the taker to a team-mate, who has to beat a
// defender in the air. Everyone is back for it, so the defending side sends
// one of its best in the air whatever line he plays in.
func (m *stepMatch) corner(att, def *stepSide, minute, depth int) {
	var taker *stepPlayer
	for _, p := range att.outfield {
		if p.ID == att.tactics.SetPieceTaker {
			taker = p
		}
	}
	if taker == nil {
		taker = m.pick(att.outfield, squared(crossScore))
	}
	box := without(att.outfield, taker)
	// A goal down with the match nearly over, the keeper comes up for it.
	if att.keeper != nil && minute >= tuning.StepsKeeperUpFromMinute && m.lead(att.team) == -1 {
		box = append(box, att.keeper)
	}
	target := m.pick(box, func(p *stepPlayer) float64 {
		s := aerialScore(p.Attributes)
		w := s * s
		if p.Role == PlayerRoleTargetMan {
			w *= tuning.StepsTargetManWeight
		}
		if p == att.keeper {
			// Nobody is told to mark him, and everybody looks for him.
			w = tuning.StepsKeeperUpWeight * tuning.StepsTypicalScore * tuning.StepsTypicalScore
		}
		return w
	})
	if taker == nil || target == nil {
		return
	}
	marker := m.pick(def.outfield, squared(aerialScore))
	// The delivery counts: a good one is harder to defend.
	delivery := tuning.CornerDeliveryFactor(taker.Attributes.EffectiveCrossing())
	chain := []Step{{Kind: StepKindCross, ActorID: taker.ID, Won: true, Detail: "corner"}}
	m.aerial(att, def, target, marker, 1/delivery, ChanceTypeCorner, chain, taker.ID, minute, depth)
}

// shot plays a chance to its end and emits it: blocked, off target, saved or
// a goal. chain is how the chance came about; momentum is what the last step
// earned the shooter.
//
// how is what a goal from this chance would be remembered as (solo,
// rebound), if anything.
func (m *stepMatch) shot(att, def *stepSide, shooter *stepPlayer, ct ChanceType, chain []Step, momentum float64, assist string, minute, depth int, how string) {
	r := m.r
	cfg := tuning.StepsShots[string(ct)]
	s := shotScore(shooter.Attributes, ct)
	if shooter == att.keeper {
		// A keeper up for a corner is a big man nobody has planned for.
		s = math.Max(s, tuning.StepsKeeperUpHeader)
	}
	mult := m.attackMod(att, shooter) * momentum * att.fade(minute)
	if ct == ChanceTypeLongRange {
		// A shot from distance needs a moment to set himself.
		mult *= squeeze(def, nil) * care(att)
	}
	att.totals.Shots++
	if assist == shooter.ID {
		assist = ""
	}

	miss := func(detail, stopper string) {
		m.events = append(m.events, GameEvent{
			Type: GameEventTypeMiss, Minute: minute, ChanceType: ct, Steps: chain,
			Event: MissEvent{PlayerID: shooter.ID, TeamType: att.team, Detail: detail, StopperID: stopper},
		})
	}

	// A defender in the way. Long shots are met by the midfield, the rest
	// by the defence.
	if cfg.Block > 0 {
		line := PlayerPositionDefense
		if ct == ChanceTypeLongRange {
			line = PlayerPositionMidfield
		}
		if blocker, cover := facing(shooter, att, def, line, blockScore); blocker != nil {
			beaten := oddsToP(stepOdds(s, blockScore(blocker.Attributes), mult/defendMod(def, blocker, cover)))
			if r.Float64() < cfg.Block*(1-beaten) {
				def.totals.Blocks++
				chain = append(chain, Step{Kind: StepKindShot, ActorID: shooter.ID, OpponentID: blocker.ID, Detail: "blocked"})
				miss("blocked", blocker.ID)
				m.maybeCorner(att, def, tuning.StepsCornerAfterBlock, minute, depth)
				return
			}
		}
	}

	if !m.wins(stepOdds(s, cfg.OnTarget, mult)) {
		detail := "wide"
		switch roll := r.Float64(); {
		case roll < tuning.StepsWoodworkShare/2:
			detail = "post"
		case roll < tuning.StepsWoodworkShare:
			detail = "bar"
		case roll < tuning.StepsWoodworkShare+tuning.StepsOverShare:
			detail = "over"
		}
		if detail == "post" || detail == "bar" {
			att.totals.Woodwork++
		}
		chain = append(chain, Step{Kind: StepKindShot, ActorID: shooter.ID, Detail: detail})
		miss(detail, "")
		// Off the woodwork it can come straight back out.
		if (detail == "post" || detail == "bar") && ct != ChanceTypePenalty &&
			depth < tuning.StepsMaxFollowUps && r.Float64() < tuning.StepsReboundAfterWoodwork {
			m.rebound(att, def, ct, detail, minute, depth)
		}
		return
	}
	att.totals.ShotsOnTarget++
	chain = append(chain, Step{Kind: StepKindShot, ActorID: shooter.ID, Won: true, Detail: "on_target"})

	keeper, keeperID, keeperMod := tuning.StepsFallbackKeeper, "", 1.0
	handling := tuning.StepsFallbackKeeper
	if def.keeper != nil {
		keeper = keeperScore(def.keeper.Attributes, ct)
		handling = float64(def.keeper.Attributes.EffectiveHandling())
		keeperID = def.keeper.ID
		keeperMod = def.mod(def.keeper)
	}
	if m.wins(stepOdds(s, cfg.Keeper*keeper, mult/keeperMod)) {
		chain = append(chain, Step{Kind: StepKindSave, ActorID: shooter.ID, OpponentID: keeperID, Won: true, Detail: "goal"})
		switch {
		case shooter == att.keeper:
			how = "keeper"
		case how == "" && ct == ChanceTypeLongRange && r.Float64() < worldieChance(shooter.Attributes):
			how = "worldie"
		}
		m.events = append(m.events, GameEvent{
			Type: GameEventTypeGoal, Minute: minute, ChanceType: ct, Steps: chain,
			Event: GoalEvent{PlayerID: shooter.ID, TeamType: att.team, AssistID: assist, Detail: how},
		})
		if att.team == TeamTypeHome {
			m.homeGoals++
		} else {
			m.awayGoals++
		}
		return
	}

	def.totals.Saves++
	// Held, or pushed away. A penalty saved is always pushed away.
	if ct != ChanceTypePenalty && m.wins(stepOdds(handling, tuning.StepsHeldScale*s, keeperMod)) {
		chain = append(chain, Step{Kind: StepKindSave, ActorID: shooter.ID, OpponentID: keeperID, Detail: "held"})
		miss("held", keeperID)
		return
	}
	chain = append(chain, Step{Kind: StepKindSave, ActorID: shooter.ID, OpponentID: keeperID, Detail: "parried"})
	miss("parried", keeperID)
	if depth >= tuning.StepsMaxFollowUps {
		return
	}
	switch roll := r.Float64(); {
	case roll < tuning.StepsCornerAfterParry:
		m.corner(att, def, minute, depth+1)
	case roll < tuning.StepsCornerAfterParry+tuning.StepsReboundAfterParry:
		m.rebound(att, def, ct, "save", minute, depth)
	}
}

// rebound is a loose ball in the box, off the keeper or the woodwork, and a
// shot for whoever gets to it first. Normally that is a forward who reads
// where it will drop. From a corner everybody is up, and it can be anyone.
func (m *stepMatch) rebound(att, def *stepSide, ct ChanceType, source string, minute, depth int) {
	follower := m.pick(att.outfield, func(p *stepPlayer) float64 {
		w := math.Pow(poachScore(p.Attributes)/tuning.StepsTypicalScore, tuning.StepsAptitude)
		if ct != ChanceTypeCorner {
			switch p.line {
			case PlayerPositionAttack:
				w *= tuning.StepsPoacherAttack
			case PlayerPositionDefense:
				w *= tuning.StepsPoacherDefense
			}
		}
		return w
	})
	if follower == nil {
		return
	}
	chain := []Step{{Kind: StepKindRebound, ActorID: follower.ID, Won: true, Detail: source}}
	m.shot(att, def, follower, ChanceTypeOpenPlay, chain, tuning.StepsMomentum, "", minute, depth+1, "rebound")
}

// stop records an attack that ended before a shot. Every such stop is a
// candidate event; capStops keeps a few of them.
func (m *stepMatch) stop(def *stepSide, stopper, against *stepPlayer, detail string, chain []Step, minute int) {
	m.events = append(m.events, GameEvent{
		Type: GameEventTypeStop, Minute: minute, Steps: chain,
		Event: StopEvent{PlayerID: stopper.ID, TeamType: def.team, AgainstID: against.ID, Detail: detail},
	})
}

// capStops keeps at most tuning.StepsMaxStops stop events, chosen at random,
// so a replay stays about as long as it was.
func (m *stepMatch) capStops() {
	var stops []int
	for i, e := range m.events {
		if e.Type == GameEventTypeStop {
			stops = append(stops, i)
		}
	}
	if len(stops) <= tuning.StepsMaxStops {
		return
	}
	drop := map[int]bool{}
	for _, j := range m.r.Perm(len(stops))[tuning.StepsMaxStops:] {
		drop[stops[j]] = true
	}
	kept := m.events[:0]
	for i, e := range m.events {
		if !drop[i] {
			kept = append(kept, e)
		}
	}
	m.events = kept
}
