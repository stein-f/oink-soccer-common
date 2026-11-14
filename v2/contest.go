package soccer

import (
	"math"
	"math/rand"
	"sort"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
)

// The contest model (ModelContest, season 19+).
//
// Each possession is a chain of contests between named players:
//
//  1. build-up: a carrier against a defender. The defender may win the ball
//     (the attack ends) or foul (a free kick or, rarely, a penalty);
//  2. the chance: an open-play type from the attack, or the set piece the
//     foul gave;
//  3. the shot: the shooter against a blocker, then whether it's on target;
//  4. the keeper: the shot against the goalkeeper.
//
// Blocks and saves sometimes give a corner, which is played straight away.
// Only shots are emitted (a goal or a miss, with its chance type), exactly
// like the classic model, so every consumer keeps working. Stops and fouls
// are played but not yet reported.

// contestSide is everything about one team that the contests read, worked
// out once per match.
type contestSide struct {
	team      TeamType
	lineup    GameLineup
	profile   FormationProfile
	tactics   Tactics
	itemBoost float64
	posBoosts positionMultipliers
	captain   float64
	control   float64 // possession weight
	duel      float64 // multiplies every score this side brings to a duel
	keeper    *SelectedPlayer
}

func newContestSide(r *rand.Rand, team TeamType, lineup GameLineup) (contestSide, []BoostRoll) {
	side := contestSide{
		team:    team,
		lineup:  lineup,
		profile: formationProfileFor(lineup.Team.Formation),
		tactics: lineup.Team.Tactics,
		captain: captainBoost(lineup),
		duel:    1.0,
	}
	if team == TeamTypeHome {
		side.duel = tuning.ContestHomeDuel
	}
	var rolls []BoostRoll
	side.itemBoost, rolls = teamBoost(r, lineup)
	var posRolls []BoostRoll
	side.posBoosts, posRolls = rollPositionBoosts(r, lineup)
	for i := range lineup.Players {
		if lineup.Players[i].SelectedPosition == PlayerPositionGoalkeeper {
			side.keeper = &lineup.Players[i]
			break
		}
	}
	return side, append(rolls, posRolls...)
}

func simulateContestMatch(r *rand.Rand, home, away GameLineup) ([]GameEvent, Injuries, MatchBoosts) {
	tempoFactor := (tempoChanceFactor(home.Team.Tactics.Tempo) + tempoChanceFactor(away.Team.Tactics.Tempo)) / 2.0
	rng := tuning.ContestPossessions
	possessions := scaleChances(r, r.Intn(rng.Max-rng.Min+1)+rng.Min, tempoFactor)
	minutes := scheduleMinutes(r, possessions)

	h, homeRolls := newContestSide(r, TeamTypeHome, home)
	a, awayRolls := newContestSide(r, TeamTypeAway, away)
	boosts := MatchBoosts{Home: homeRolls, Away: awayRolls}

	// Possession weights, as in the classic model: control, shape, captain,
	// items, and the opponent's press and line.
	h.control = teamControl(home, h.posBoosts) * h.profile.Possession * h.captain * h.itemBoost *
		tuning.ContestHomePossession *
		pressControlFactor(a.tactics.Press) * lineHeightControlFactor(a.tactics.LineHeight)
	a.control = teamControl(away, a.posBoosts) * a.profile.Possession * a.captain * a.itemBoost *
		pressControlFactor(h.tactics.Press) * lineHeightControlFactor(h.tactics.LineHeight)

	m := contestMatch{r: r}
	for _, minute := range minutes {
		// Score effects: the side behind sees more of the ball.
		hw := math.Pow(h.control*scoreEffect(m.homeGoals-m.awayGoals), tuning.ContestPossessionExponent)
		aw := math.Pow(a.control*scoreEffect(m.awayGoals-m.homeGoals), tuning.ContestPossessionExponent)
		if pickAttackingTeam(r, hw, aw) == TeamTypeHome {
			m.possession(&h, &a, minute)
		} else {
			m.possession(&a, &h, minute)
		}
	}

	homeInjuries := rollInjuries(r, home, teamAverageAggression(away), a.profile.InjuryRisk*pressInjuryFactor(h.tactics.Press), home.Team.ID)
	awayInjuries := rollInjuries(r, away, teamAverageAggression(home), h.profile.InjuryRisk*pressInjuryFactor(a.tactics.Press), away.Team.ID)
	return m.events, Injuries{HomeTeamInjuries: homeInjuries, AwayTeamInjuries: awayInjuries}, boosts
}

// scoreEffect is the possession multiplier for a side leading by `lead`
// goals (negative when behind).
func scoreEffect(lead int) float64 {
	shift := tuning.ContestScoreEffect * float64(lead)
	if shift > tuning.ContestScoreEffectCap {
		shift = tuning.ContestScoreEffectCap
	} else if shift < -tuning.ContestScoreEffectCap {
		shift = -tuning.ContestScoreEffectCap
	}
	return 1 - shift
}

type contestMatch struct {
	r                    *rand.Rand
	events               []GameEvent
	homeGoals, awayGoals int
}

// lead is how many goals side t is ahead by (negative when behind).
func (m *contestMatch) lead(t TeamType) int {
	if t == TeamTypeHome {
		return m.homeGoals - m.awayGoals
	}
	return m.awayGoals - m.homeGoals
}

// easeOff is the attacking multiplier for a side that is comfortably ahead.
func easeOff(lead int) float64 {
	if lead < tuning.ContestComfortableLead {
		return 1
	}
	cut := tuning.ContestEaseOff * float64(lead-tuning.ContestComfortableLead+1)
	if cut > tuning.ContestEaseOffCap {
		cut = tuning.ContestEaseOffCap
	}
	return 1 - cut
}

// possession plays one attack by `att` against `def`.
func (m *contestMatch) possession(att, def *contestSide, minute int) {
	r := m.r
	carrier, ok := pickWeighted(r, att.lineup.Players, func(p SelectedPlayer) float64 {
		w := tuning.ContestCarrierWeights.ForPosition(string(p.SelectedPosition)) * playerControl(p, att.tactics)
		if p.Role == PlayerRolePlaymaker {
			w *= tuning.ContestPlaymakerCarryWeight
		}
		return w
	})
	if !ok {
		return
	}
	stopper, hasStopper := pickStopper(r, def)

	if hasStopper {
		a := playerControl(carrier, att.tactics) * att.profile.ChanceCreation * att.captain *
			att.itemBoost * att.posBoosts.forPosition(carrier.SelectedPosition) * att.duel *
			easeOff(m.lead(att.team))
		d := defenderScore(def, stopper) * pressStopFactor(def.tactics.Press)
		if r.Float64() < duel(tuning.ContestStopScale*d, a) {
			// Won the ball, cleanly or not.
			if r.Float64() < foulChance(stopper, def.tactics.Press) {
				roll := r.Float64()
				switch {
				case roll < tuning.ContestPenaltyShareOfFouls:
					m.shot(att, def, ChanceTypePenalty, minute, 0)
				case roll < tuning.ContestPenaltyShareOfFouls+tuning.ContestShootingFreeKickShareOfFouls:
					m.shot(att, def, ChanceTypeFreeKick, minute, 0)
				}
			}
			return
		}
	}

	m.shot(att, def, pickOpenChanceType(r, def.tactics.LineHeight), minute, 0)
}

// shot plays a chance of type ct to its end: blocked, off target, saved or a
// goal, emitting one event. Blocks and saves can give a corner, played at once.
func (m *contestMatch) shot(att, def *contestSide, ct ChanceType, minute, depth int) {
	r := m.r
	cfg := tuning.ContestShots[string(ct)]
	shooter := pickAttackerWithTactics(r, att.lineup, ct, att.tactics)

	s := playerAttackForChance(shooter, ct) * att.profile.ChanceQuality *
		tempoQualityFactor(att.tactics.Tempo) * pressFatigueFactor(att.tactics.Press, minute) *
		cornerDeliveryFactor(att.lineup, ct, att.tactics) *
		att.itemBoost * att.posBoosts.forPosition(shooter.SelectedPosition) * att.duel *
		easeOff(m.lead(att.team))
	if s < 1 {
		s = 1
	}

	cornerChance := 0.0
	goal := false
	if blocker, ok := pickStopper(r, def); ok && cfg.Block > 0 && r.Float64() < cfg.Block*duel(defenderScore(def, blocker), s) {
		cornerChance = tuning.ContestCornerAfterBlock
	} else if r.Float64() < duel(s, cfg.OnTarget) {
		k := tuning.ContestFallbackKeeper
		if def.keeper != nil {
			k = playerDefense(*def.keeper, def.tactics) * def.captain * def.itemBoost *
				def.posBoosts.forPosition(def.keeper.SelectedPosition) * def.duel
		}
		if r.Float64() < duel(s, cfg.Keeper*k) {
			goal = true
		} else {
			cornerChance = tuning.ContestCornerAfterSave
		}
	}

	ev := GameEvent{Minute: minute, ChanceType: ct}
	if goal {
		ev.Type = GameEventTypeGoal
		ev.Event = GoalEvent{PlayerID: shooter.ID, TeamType: att.team}
		if att.team == TeamTypeHome {
			m.homeGoals++
		} else {
			m.awayGoals++
		}
	} else {
		ev.Type = GameEventTypeMiss
		ev.Event = MissEvent{PlayerID: shooter.ID, TeamType: att.team}
	}
	m.events = append(m.events, ev)

	if cornerChance > 0 && depth < tuning.ContestMaxFollowUps && r.Float64() < cornerChance {
		m.shot(att, def, ChanceTypeCorner, minute, depth+1)
	}
}

// defenderScore is a defender's (or blocker's) side of a duel: their defense
// score under the team's line, scaled by shape, captain and items.
func defenderScore(def *contestSide, p SelectedPlayer) float64 {
	return playerDefense(p, def.tactics) * def.profile.DefSolidity * def.captain *
		def.itemBoost * def.posBoosts.forPosition(p.SelectedPosition) *
		lineHeightDefenseFactor(def.tactics.LineHeight) * def.duel
}

// duel is the chance that a beats b in one contest, softened by
// tuning.ContestDuelExponent: a^e / (a^e + b^e).
func duel(a, b float64) float64 {
	if a <= 0 {
		return 0
	}
	if b <= 0 {
		return 1
	}
	ae := math.Pow(a, tuning.ContestDuelExponent)
	return ae / (ae + math.Pow(b, tuning.ContestDuelExponent))
}

func pickStopper(r *rand.Rand, def *contestSide) (SelectedPlayer, bool) {
	return pickWeighted(r, def.lineup.Players, func(p SelectedPlayer) float64 {
		w := tuning.ContestStopperWeights.ForPosition(string(p.SelectedPosition)) * playerDefense(p, def.tactics)
		if p.Role == PlayerRoleBallWinner {
			w *= tuning.ContestBallWinnerStopWeight
		}
		return w
	})
}

// foulChance is the share of a defender's challenges that are fouls.
func foulChance(p SelectedPlayer, press PressLevel) float64 {
	agg := float64(p.Attributes.AggressionRating)
	if agg <= 0 {
		agg = tuning.ContestFoulAggressionPivot
	}
	f := tuning.ContestFoulBase * math.Pow(agg/tuning.ContestFoulAggressionPivot, 1.5)
	switch press {
	case PressLevelHigh:
		f *= tuning.ContestHighPressFoul
	case PressLevelLow:
		f *= tuning.ContestLowPressFoul
	}
	if f > tuning.ContestFoulCap {
		f = tuning.ContestFoulCap
	}
	return f
}

func pressStopFactor(p PressLevel) float64 {
	switch p {
	case PressLevelHigh:
		return tuning.ContestHighPressStop
	case PressLevelLow:
		return tuning.ContestLowPressStop
	default:
		return 1.0
	}
}

// contestOpenChanceOrder pins the iteration order of the open-chance weights.
var contestOpenChanceOrder = []ChanceType{
	ChanceTypeOpenPlay,
	ChanceTypeCross,
	ChanceTypeLongRange,
	ChanceTypeGoalKeeperShot,
}

func pickOpenChanceType(r *rand.Rand, defendingLine LineHeight) ChanceType {
	weights := make([]float64, len(contestOpenChanceOrder))
	var total float64
	for i, ct := range contestOpenChanceOrder {
		w := float64(tuning.ContestOpenChanceWeights[string(ct)])
		if ct == ChanceTypeGoalKeeperShot {
			switch defendingLine {
			case LineHeightHigh:
				w *= tuning.ContestBreakawayVsHighLine
			case LineHeightDeep:
				w *= tuning.ContestBreakawayVsDeepLine
			}
		}
		weights[i] = w
		total += w
	}
	pick := r.Float64() * total
	for i, w := range weights {
		if pick < w {
			return contestOpenChanceOrder[i]
		}
		pick -= w
	}
	return contestOpenChanceOrder[len(contestOpenChanceOrder)-1]
}

// pickWeighted picks a player in proportion to weight, iterating in ID order
// so the result is deterministic. ok is false when nobody has any weight.
func pickWeighted(r *rand.Rand, players []SelectedPlayer, weight func(SelectedPlayer) float64) (SelectedPlayer, bool) {
	sorted := make([]SelectedPlayer, len(players))
	copy(sorted, players)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	weights := make([]float64, len(sorted))
	var total float64
	for i, p := range sorted {
		w := weight(p)
		if w > 0 {
			weights[i] = w
			total += w
		}
	}
	if total <= 0 {
		return SelectedPlayer{}, false
	}
	pick := r.Float64() * total
	for i, w := range weights {
		if pick < w {
			return sorted[i], true
		}
		pick -= w
	}
	return sorted[len(sorted)-1], true
}
