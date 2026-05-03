package soccer

import (
	"math"
	"math/rand"
	"sort"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
)

// simulateMatch is the v2 engine. It returns events in chronological order
// and the per-team injury list.
//
// The simulation runs in distinct phases (vs v1's flat "roll N independent
// dice"):
//
//  1. Tempo:    determine how many chances the match will produce, derived
//     from the formation styles (more attacking shapes ⇒ more
//     chances).
//  2. Schedule: scatter chance minutes across the match using the v1
//     event-minute distribution (late-game weighting).
//  3. Possess:  for each chance, decide which team has the ball based on
//     possession-weighted team control.
//  4. Resolve:  for each chance, pick the chance type, attacker, and
//     outcome (goal/miss). Outcome weights honour the chance type
//     (penalties are easy, long-range hard) and the formations'
//     profile multipliers.
//  5. Injuries: roll injuries per team based on opponent aggression and
//     opponent-formation injury risk.
//
// Determinism: the function is a pure function of (rand, home, away). No
// time.Now(), no globals, no I/O.
func simulateMatch(r *rand.Rand, home, away GameLineup) ([]GameEvent, Injuries) {
	homeProfile := formationProfileFor(home.Team.Formation)
	awayProfile := formationProfileFor(away.Team.Formation)

	homeTactics := home.Team.Tactics
	awayTactics := away.Team.Tactics

	// Tactics modulate the chance volume *per team*, but we generate a
	// single combined count to keep events interleaved chronologically.
	tempoFactor := (tempoChanceFactor(homeTactics.Tempo) + tempoChanceFactor(awayTactics.Tempo)) / 2.0
	totalChances := decideMatchTempo(r, home.Team.Formation, away.Team.Formation, tempoFactor)
	minutes := scheduleMinutes(r, totalChances)

	// Compute team scores once — they don't change minute-to-minute.
	// Pressing reduces *opponent* control; line height does the same.
	homeControl := teamControl(home) * homeProfile.Possession * captainBoost(home) * teamBoost(r, home)
	awayControl := teamControl(away) * awayProfile.Possession * captainBoost(away) * teamBoost(r, away)
	homeControl *= pressControlFactor(awayTactics.Press) * lineHeightControlFactor(awayTactics.LineHeight)
	awayControl *= pressControlFactor(homeTactics.Press) * lineHeightControlFactor(homeTactics.LineHeight)

	homeDefense := teamDefense(home) * homeProfile.DefSolidity * captainBoost(home) * tuning.DefenseBiasMultiplier * teamBoost(r, home)
	awayDefense := teamDefense(away) * awayProfile.DefSolidity * captainBoost(away) * tuning.DefenseBiasMultiplier * teamBoost(r, away)
	homeDefense *= lineHeightDefenseFactor(homeTactics.LineHeight)
	awayDefense *= lineHeightDefenseFactor(awayTactics.LineHeight)

	events := make([]GameEvent, 0, totalChances)
	var prevType ChanceType

	for i := 0; i < totalChances; i++ {
		// Possession: which team gets this chance?
		attacker := pickAttackingTeam(r, homeControl, awayControl)
		var (
			attackingLineup  GameLineup
			attackingProfile FormationProfile
			attackingTactics Tactics
			defendingDefense float64
		)
		if attacker == TeamTypeHome {
			attackingLineup = home
			attackingProfile = homeProfile
			attackingTactics = homeTactics
			defendingDefense = awayDefense
		} else {
			attackingLineup = away
			attackingProfile = awayProfile
			attackingTactics = awayTactics
			defendingDefense = homeDefense
		}

		ct := pickChanceType(r, prevType)
		prevType = ct

		ap := pickAttackerWithTactics(r, attackingLineup, ct, attackingTactics)
		ev := resolveChance(r, ap, attacker, ct, attackingProfile, attackingTactics, defendingDefense, minutes[i])
		events = append(events, ev)
	}

	// Injuries: own injury risk scales with own press level too.
	homeAggression := teamAverageAggression(home)
	awayAggression := teamAverageAggression(away)
	homeInjuries := rollInjuries(r, home, awayAggression, awayProfile.InjuryRisk*pressInjuryFactor(homeTactics.Press), home.Team.ID)
	awayInjuries := rollInjuries(r, away, homeAggression, homeProfile.InjuryRisk*pressInjuryFactor(awayTactics.Press), away.Team.ID)

	return events, Injuries{HomeTeamInjuries: homeInjuries, AwayTeamInjuries: awayInjuries}
}

// decideMatchTempo picks the total number of chances using the truth-table
// from tuning, then scales by the supplied tempoFactor (combined home+away
// tempo tactic). With both teams on neutral tempo, factor=1.0 and behaviour
// matches v1 exactly.
func decideMatchTempo(r *rand.Rand, homeF, awayF FormationType, tempoFactor float64) int {
	key := "HOME:" + formationStyleKey(homeF) + "|AWAY:" + formationStyleKey(awayF)
	rng, ok := tuning.FormationChanceRanges[key]
	if !ok {
		rng = tuning.FallbackChanceRange
	}
	base := rng.Min
	if rng.Max > rng.Min {
		base = r.Intn(rng.Max-rng.Min+1) + rng.Min
	}
	return scaleChances(r, base, tempoFactor)
}

// scaleChances multiplies an integer chance count by a non-integer factor,
// keeping the expected value intact: any fractional part is added with
// probability equal to the fraction. E.g. base=6 factor=1.10 ⇒ scaled=6.6,
// returns 7 with 60% probability and 6 with 40% probability. This avoids
// the rounding-to-zero problem with small tempo deltas.
func scaleChances(r *rand.Rand, base int, factor float64) int {
	if factor == 1.0 || factor == 0 {
		return base
	}
	scaled := float64(base) * factor
	whole := int(scaled)
	frac := scaled - float64(whole)
	if frac > 0 && r.Float64() < frac {
		whole++
	}
	if whole < 1 {
		whole = 1
	}
	return whole
}

// formationStyleKey buckets each formation into ATT / BAL / DEF for the
// chance-range table. Box is treated as BAL despite being attacking-shaped
// because its chance *quality* (not volume) is its lever.
func formationStyleKey(f FormationType) string {
	switch f {
	case FormationTypePyramid:
		return "DEF"
	case FormationTypeY:
		return "ATT"
	case FormationTypeDiamond, FormationTypeBox:
		return "BAL"
	default:
		return "BAL"
	}
}

// scheduleMinutes scatters a sorted slice of minutes across the match using
// the weighted minute distribution (late-game weighted).
func scheduleMinutes(r *rand.Rand, count int) []int {
	out := make([]int, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, sampleMinute(r))
	}
	sort.Ints(out)
	return out
}

func sampleMinute(r *rand.Rand) int {
	var totalW uint
	for _, b := range tuning.EventMinuteBuckets {
		totalW += b.Weight
	}
	pick := uint(r.Intn(int(totalW)))
	var cum uint
	for _, b := range tuning.EventMinuteBuckets {
		cum += b.Weight
		if pick < cum {
			return r.Intn(b.MaxMinute-b.MinMinute+1) + b.MinMinute
		}
	}
	last := tuning.EventMinuteBuckets[len(tuning.EventMinuteBuckets)-1]
	return last.MaxMinute
}

// pickAttackingTeam runs the possession dice for one chance.
func pickAttackingTeam(r *rand.Rand, homeWeight, awayWeight float64) TeamType {
	if homeWeight <= 0 && awayWeight <= 0 {
		// Symmetric fallback so we always return something.
		if r.Intn(2) == 0 {
			return TeamTypeHome
		}
		return TeamTypeAway
	}
	if r.Float64()*(homeWeight+awayWeight) < homeWeight {
		return TeamTypeHome
	}
	return TeamTypeAway
}

// resolveChance rolls the goal/miss outcome and assembles the GameEvent.
// Tactics affect chance quality (faster tempo ⇒ rushed shots).
func resolveChance(r *rand.Rand, attacker SelectedPlayer, team TeamType, ct ChanceType, attackingProfile FormationProfile, attackingTactics Tactics, defendingDefense float64, minute int) GameEvent {
	atk := playerAttackForChance(attacker, ct)
	atk *= attackingProfile.ChanceCreation * attackingProfile.ChanceQuality
	atk *= chanceTypeAttackBoost(ct)
	atk *= tempoQualityFactor(attackingTactics.Tempo)
	atk *= pressFatigueFactor(attackingTactics.Press, minute)

	def := defendingDefense * chanceTypeDefenseScale(ct)

	// Floor scores so weighted-rand always sees positive values.
	if atk < 1 {
		atk = 1
	}
	if def < 1 {
		def = 1
	}

	// Goal probability = atk / (atk + def). r.Float64() < p ⇒ goal.
	p := atk / (atk + def)
	isGoal := r.Float64() < p

	ev := GameEvent{Minute: minute, ChanceType: ct}
	if isGoal {
		ev.Type = GameEventTypeGoal
		ev.Event = GoalEvent{PlayerID: attacker.ID, TeamType: team}
	} else {
		ev.Type = GameEventTypeMiss
		ev.Event = MissEvent{PlayerID: attacker.ID, TeamType: team}
	}
	return ev
}

// pickAttackerWithTactics picks an attacker honoring tactical overrides:
// a SetPieceTaker is preferred for free kicks / corners / penalties, and
// a TargetMan gets a selection-weight bonus on corners + crosses.
func pickAttackerWithTactics(r *rand.Rand, lineup GameLineup, ct ChanceType, tactics Tactics) SelectedPlayer {
	if tactics.SetPieceTaker != "" && isSetPieceChance(ct) {
		for _, p := range lineup.Players {
			if p.ID == tactics.SetPieceTaker {
				return p
			}
		}
	}
	return pickAttacker(r, lineup, ct)
}

func isSetPieceChance(ct ChanceType) bool {
	switch ct {
	case ChanceTypeFreeKick, ChanceTypeCorner, ChanceTypePenalty:
		return true
	}
	return false
}

// teamBoost compounds Team-typed boosts on a lineup. Position-specific
// boosts are honored downstream in the team-score helpers (Phase 5 work).
func teamBoost(r *rand.Rand, lineup GameLineup) float64 {
	total := 1.0
	for _, b := range lineup.ItemBoosts {
		if b.BoostType != BoostTypeTeam {
			continue
		}
		total *= rollBoost(r, b)
	}
	return total
}

// rollBoost samples a value from [Min, Max] and applies diminishing returns
// when Applications > 1. Mirrors v1 semantics: positive boosts decay only
// the excess over 1.0 so they never become a debuff; debuffs decay whole.
func rollBoost(r *rand.Rand, b Boost) float64 {
	base := b.MinBoost + r.Float64()*(b.MaxBoost-b.MinBoost)
	if b.Applications <= 1 {
		return base
	}
	m := math.Pow(tuning.BoostDecay, float64(b.Applications))
	if m < tuning.BoostMinMultiplier {
		m = tuning.BoostMinMultiplier
	}
	if base >= 1.0 {
		return 1.0 + (base-1.0)*m
	}
	return base * m
}
