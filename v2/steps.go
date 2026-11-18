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
//     from distance;
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
	team       TeamType
	lineup     GameLineup
	tactics    Tactics
	outfield   []*stepPlayer
	lines      map[PlayerPosition][]*stepPlayer
	keeper     *stepPlayer
	teamMod    float64 // captain, items and home advantage: every player's odds
	posBoosts  positionMultipliers
	injuryRisk float64
	totals     *TeamTotals
}

func newStepSide(r *rand.Rand, team TeamType, lineup GameLineup, totals *TeamTotals) (*stepSide, []BoostRoll) {
	s := &stepSide{
		team:       team,
		lineup:     lineup,
		tactics:    lineup.Team.Tactics,
		lines:      map[PlayerPosition][]*stepPlayer{},
		injuryRisk: formationProfileFor(lineup.Team.Formation).InjuryRisk,
		totals:     totals,
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
	// the opponent's press and line take some of it away as they do in the
	// classic model, and the home side has a little more.
	homeBall := h.quality() * pressControlFactor(a.tactics.Press) * lineHeightControlFactor(a.tactics.LineHeight) * tuning.StepsHomePossession
	awayBall := a.quality() * pressControlFactor(h.tactics.Press) * lineHeightControlFactor(h.tactics.LineHeight)
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

	homeInjuries := rollInjuries(r, home, teamAverageAggression(away), a.injuryRisk*pressInjuryFactor(h.tactics.Press), home.Team.ID)
	awayInjuries := rollInjuries(r, away, teamAverageAggression(home), h.injuryRisk*pressInjuryFactor(a.tactics.Press), away.Team.ID)
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
	if def.tactics.Press == PressLevelHigh {
		intercept = pressingInterceptScore
	}
	// The opponent's press squeezes the passer, as it squeezes a side's
	// control in the classic model, and so does a high line.
	mult := attackMod * pressControlFactor(def.tactics.Press) * passLineOdds(def.tactics.LineHeight)
	if long {
		mult *= tuning.StepsLongBall
	}
	opp, cover := facing(passer, att, def, PlayerPositionMidfield, intercept)
	if opp == nil {
		return math.Inf(1), nil
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

// pressingInterceptScore is interceptScore for a side pressing high: closing
// a passer down is running as much as reading the game, so work rate counts.
// It is what makes a stamina midfielder the one to press with.
func pressingInterceptScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveInterceptions()*2+a.EffectiveMarking()+a.EffectiveWorkRate()*2, 5)
}

// Pace is in four of these blends, on both sides of the ball, and it tracks a
// player's overall rating less than any other attribute. Its weights are kept
// modest so that a quick, ordinary player isn't worth more than a good one
// (the attribute report in cmd/calibrate is the check).

func takeOnScore(a PlayerAttributes) float64 {
	return mix(a.EffectiveDribbling()*3+a.EffectivePace(), 4)
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

// shotScore is the shooter's side of the shot, by the kind of chance.
func shotScore(a PlayerAttributes, ct ChanceType) float64 {
	switch ct {
	case ChanceTypeCross, ChanceTypeCorner: // a header
		return mix(a.EffectiveHeading()*2+a.AttackRating, 3)
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

// stepRoute is what a side tries once it has the ball forward.
type stepRoute struct {
	kind   string      // take_on | run | cross | long_shot
	actor  *stepPlayer // who does it
	target *stepPlayer // cross only: who it is aimed at
	weight float64
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

// routes weighs up what the side could try. Each option is weighted by its
// base share and by how likely it is to work for the players it would
// involve, so a side plays to its strengths and at an opponent's weakness
// without doing only that.
func (m *stepMatch) routes(att, def *stepSide) []stepRoute {
	// The forwards do most of the attacking, with a midfielder joining in
	// now and then. A side with no forwards sends its midfield.
	forwards := append(append([]*stepPlayer{}, att.lines[PlayerPositionAttack]...), att.lines[PlayerPositionMidfield]...)
	if len(forwards) == 0 {
		forwards = att.outfield
	}
	if len(forwards) == 0 {
		return nil
	}
	// The forwards make most of a side's take-ons and runs between them and
	// its midfielders the rest (StepsMidfieldAttackShare), whatever the
	// numbers in each line.
	forward := func(score func(PlayerAttributes) float64) func(*stepPlayer) float64 {
		weight := func(p *stepPlayer) float64 { return math.Pow(score(p.Attributes), tuning.StepsForwardChoice) }
		var attack, midfield float64
		for _, p := range forwards {
			if p.line == PlayerPositionAttack {
				attack += weight(p)
			} else {
				midfield += weight(p)
			}
		}
		return func(p *stepPlayer) float64 {
			switch {
			case attack == 0 || midfield == 0:
				return weight(p)
			case p.line == PlayerPositionAttack:
				return weight(p) / attack * (1 - tuning.StepsMidfieldAttackShare)
			}
			return weight(p) / midfield * tuning.StepsMidfieldAttackShare
		}
	}

	var out []stepRoute
	add := func(kind string, actor, target *stepPlayer, p float64) {
		if actor != nil {
			out = append(out, stepRoute{kind: kind, actor: actor, target: target, weight: tuning.StepsRouteWeights[kind] * p})
		}
	}

	if actor := m.pick(forwards, forward(takeOnScore)); actor != nil {
		p := 1.0
		if opp, cover := facing(actor, att, def, PlayerPositionDefense, tackleScore); opp != nil {
			p = oddsToP(stepOdds(takeOnScore(actor.Attributes), tuning.StepsTackleScale*tackleScore(opp.Attributes), 1/cover))
		}
		add("take_on", actor, nil, p)
	}

	if actor := m.pick(forwards, forward(runScore)); actor != nil {
		p, w := 1.0, 1.0
		switch def.tactics.LineHeight {
		case LineHeightHigh:
			w = tuning.StepsRunVsHighLine
		case LineHeightDeep:
			w = tuning.StepsRunVsDeepLine
		}
		if opp, cover := facing(actor, att, def, PlayerPositionDefense, chaseScore); opp != nil {
			p = oddsToP(stepOdds(runScore(actor.Attributes), tuning.StepsChaseScale*chaseScore(opp.Attributes), runLineOdds(def.tactics.LineHeight)/cover))
		}
		add("run", actor, nil, p*w)
	}

	// A cross needs someone to put it in and someone else to attack it.
	advanced := append(append([]*stepPlayer{}, att.lines[PlayerPositionAttack]...), att.lines[PlayerPositionMidfield]...)
	if len(advanced) >= 2 {
		crosser := m.pick(advanced, func(p *stepPlayer) float64 {
			s := crossScore(p.Attributes)
			if p.lane != 0 {
				s *= 1.5 // wide players do most of the crossing
			}
			return s * s
		})
		target := m.pick(without(advanced, crosser), func(p *stepPlayer) float64 {
			s := aerialScore(p.Attributes)
			w := s * s
			if p.Role == PlayerRoleTargetMan {
				w *= tuning.StepsTargetManWeight
			}
			if p.line == PlayerPositionAttack {
				w *= 2 // forwards are in the box; a midfielder has to arrive
			}
			return w
		})
		if crosser != nil && target != nil {
			p := 1.0
			if opp, cover := facing(crosser, att, def, PlayerPositionDefense, markScore); opp != nil {
				p = oddsToP(stepOdds(crossScore(crosser.Attributes), tuning.StepsCloseDownScale*markScore(opp.Attributes), 1/cover))
			}
			if opp, cover := facing(target, att, def, PlayerPositionDefense, aerialScore); opp != nil {
				p *= oddsToP(stepOdds(aerialScore(target.Attributes), tuning.StepsMarkerScale*aerialScore(opp.Attributes), 1/cover))
			}
			add("cross", crosser, target, p)
		}
	}

	shooters := firstNonEmpty(att.lines[PlayerPositionMidfield], att.lines[PlayerPositionAttack], att.outfield)
	if actor := m.pick(shooters, squared(func(a PlayerAttributes) float64 { return shotScore(a, ChanceTypeLongRange) })); actor != nil {
		cfg := tuning.StepsShots[string(ChanceTypeLongRange)]
		add("long_shot", actor, nil, oddsToP(stepOdds(shotScore(actor.Attributes, ChanceTypeLongRange), cfg.OnTarget, 1)))
	}
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

	var chain []Step
	momentum := 1.0
	assist := ""

	// 1. The ball forward.
	if passer := m.passer(att, route); passer != nil {
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
		step := Step{Kind: StepKindTakeOn, ActorID: actor.ID, Won: true}
		opp, cover := facing(actor, att, def, PlayerPositionDefense, tackleScore)
		if opp != nil {
			step.OpponentID = opp.ID
			mult := m.attackMod(att, actor) * momentum / (defendMod(def, opp, cover) * pressStopFactor(def.tactics.Press))
			step.Won = m.wins(stepOdds(takeOnScore(actor.Attributes), tuning.StepsTackleScale*tackleScore(opp.Attributes), mult))
		}
		att.totals.TakeOns++
		chain = append(chain, step)
		// Past his man, he still has to get past the one covering.
		if cover := coverFor(def, opp); step.Won && cover != nil {
			mult := m.attackMod(att, actor) / (defendMod(def, cover, 1) * pressStopFactor(def.tactics.Press))
			second := Step{Kind: StepKindTakeOn, ActorID: actor.ID, OpponentID: cover.ID,
				Won: m.wins(stepOdds(takeOnScore(actor.Attributes), tuning.StepsCoverScale*tackleScore(cover.Attributes), mult))}
			att.totals.TakeOns++
			chain = append(chain, second)
			step, opp = second, cover
		}
		if step.Won {
			att.totals.TakeOnsWon++
			// Past his man, he squares it to a team-mate or goes alone.
			// Going alone makes the chance his own: no assist.
			if m.r.Float64() < tuning.StepsCutBack {
				mates := without(append(append([]*stepPlayer{}, att.lines[PlayerPositionAttack]...), att.lines[PlayerPositionMidfield]...), actor)
				if mate := m.pick(mates, squared(func(a PlayerAttributes) float64 { return shotScore(a, ChanceTypeOpenPlay) })); mate != nil {
					m.shot(att, def, mate, ChanceTypeOpenPlay, chain, tuning.StepsMomentum, actor.ID, minute, 0)
					return
				}
			}
			m.shot(att, def, actor, ChanceTypeOpenPlay, chain, tuning.StepsMomentum, "", minute, 0)
			return
		}
		m.tackled(att, def, actor, opp, chain, minute)

	case "run":
		step := Step{Kind: StepKindRun, ActorID: actor.ID, Won: true}
		opp, cover := facing(actor, att, def, PlayerPositionDefense, chaseScore)
		if opp != nil {
			step.OpponentID = opp.ID
			mult := m.attackMod(att, actor) * momentum * runLineOdds(def.tactics.LineHeight) / (def.mod(opp) * cover)
			step.Won = m.wins(stepOdds(runScore(actor.Attributes), tuning.StepsChaseScale*chaseScore(opp.Attributes), mult))
		}
		chain = append(chain, step)
		// In behind the last man, unless the other defender gets across.
		if cover := coverFor(def, opp); step.Won && cover != nil {
			mult := m.attackMod(att, actor) * runLineOdds(def.tactics.LineHeight) / def.mod(cover)
			second := Step{Kind: StepKindRun, ActorID: actor.ID, OpponentID: cover.ID,
				Won: m.wins(stepOdds(runScore(actor.Attributes), tuning.StepsCoverScale*chaseScore(cover.Attributes), mult))}
			chain = append(chain, second)
			step, opp = second, cover
		}
		if step.Won {
			m.shot(att, def, actor, ChanceTypeGoalKeeperShot, chain, tuning.StepsMomentum, assist, minute, 0)
			return
		}
		def.totals.Tackles++
		m.stop(def, opp, actor, "caught", chain, minute)

	case "cross":
		step := Step{Kind: StepKindCross, ActorID: actor.ID, Won: true}
		if opp, cover := facing(actor, att, def, PlayerPositionDefense, markScore); opp != nil {
			step.OpponentID = opp.ID
			mult := m.attackMod(att, actor) * momentum / defendMod(def, opp, cover)
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
		// harder to defend.
		cover /= tuning.CornerDeliveryFactor(actor.Attributes.EffectiveCrossing())
		m.aerial(att, def, route.target, marker, cover, ChanceTypeCross, chain, actor.ID, minute, 0)

	case "long_shot":
		m.shot(att, def, actor, ChanceTypeLongRange, chain, momentum, assist, minute, 0)
	}
}

// tackled is a take-on the defender won: cleanly, which ends the attack, or
// with a foul, which may give a free kick in range or a penalty.
func (m *stepMatch) tackled(att, def *stepSide, actor, opp *stepPlayer, chain []Step, minute int) {
	fouls := foulChance(opp.SelectedPlayer, def.tactics.Press)
	if opp.Role == PlayerRoleBallWinner {
		fouls = math.Min(fouls*tuning.StepsBallWinnerFoul, tuning.ContestFoulCap)
	}
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
			m.shot(att, def, taker, ChanceTypePenalty, chain, 1, "", minute, 0)
		}
	case roll < tuning.StepsPenaltyShareOfFouls+tuning.StepsFreeKickShareOfFouls:
		chain = append(chain, Step{Kind: StepKindFoul, ActorID: actor.ID, OpponentID: opp.ID, Won: true, Detail: "free_kick"})
		if taker := m.taker(att, ChanceTypeFreeKick); taker != nil {
			m.shot(att, def, taker, ChanceTypeFreeKick, chain, 1, "", minute, 0)
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
	m.shot(att, def, target, ct, chain, tuning.StepsMomentum, assist, minute, depth)
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
	target := m.pick(without(att.outfield, taker), func(p *stepPlayer) float64 {
		s := aerialScore(p.Attributes)
		w := s * s
		if p.Role == PlayerRoleTargetMan {
			w *= tuning.StepsTargetManWeight
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
func (m *stepMatch) shot(att, def *stepSide, shooter *stepPlayer, ct ChanceType, chain []Step, momentum float64, assist string, minute, depth int) {
	r := m.r
	cfg := tuning.StepsShots[string(ct)]
	s := shotScore(shooter.Attributes, ct)
	mult := m.attackMod(att, shooter) * momentum *
		tempoQualityFactor(att.tactics.Tempo) * pressFatigueFactor(att.tactics.Press, minute)
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
		if blocker, cover := facing(shooter, att, def, line, markScore); blocker != nil {
			beaten := oddsToP(stepOdds(s, markScore(blocker.Attributes), mult/defendMod(def, blocker, cover)))
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
		m.events = append(m.events, GameEvent{
			Type: GameEventTypeGoal, Minute: minute, ChanceType: ct, Steps: chain,
			Event: GoalEvent{PlayerID: shooter.ID, TeamType: att.team, AssistID: assist},
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
		// The loose ball falls to someone else, close in.
		if follower := m.pick(without(att.outfield, shooter), squared(func(a PlayerAttributes) float64 { return shotScore(a, ChanceTypeOpenPlay) })); follower != nil {
			m.shot(att, def, follower, ChanceTypeOpenPlay, nil, 1, "", minute, depth+1)
		}
	}
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
