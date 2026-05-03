package soccer

import (
	"math"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
)

// rawControl returns a player's control score from raw attributes only —
// no out-of-position penalty, no injury reduction. It folds together a
// player's ControlRating and EffectiveWorkRate using weights that depend on
// the team's press level.
//
// Press shifts which attribute matters most: high press demands work rate
// (close space, win the ball back), low press leans on technique (keep the
// ball, pick passes). Neutral / no-tactic falls back to the legacy weights.
//
// In v1 every "raw" score folded SpeedRating in, which meant a fast attacker
// got a "free" boost to control + defense. v2 splits the physical attribute
// per action, so increasing WorkRate only affects control, increasing Pace
// only affects attack, and increasing Recovery only affects defense.
func rawControl(p PlayerAttributes, tactics Tactics) float64 {
	w := tuning.ControlWeightsForPress(string(tactics.Press))
	return weighted(p.ControlRating, p.EffectiveWorkRate(), w.Skill, w.Physical, w.Divisor)
}

// rawAttack returns a player's attack score from raw attributes only, using
// the default open-play formula (skill*3 + pace) / 4. Most callers should
// prefer rawAttackForChance, which delegates to the chance type's own
// AttackScore function so set pieces, breakaways, and long range each
// reward different player builds.
func rawAttack(p PlayerAttributes) float64 {
	return weighted(p.AttackRating, p.EffectivePace(),
		tuning.AttackSkillWeight, tuning.AttackSpeedWeight, tuning.AttackDivisor)
}

// rawAttackForChance computes the attacker's raw score for a given chance
// type by calling that type's AttackScore function. Falls back to the
// default open-play formula if the chance type isn't registered or doesn't
// declare its own scoring function.
func rawAttackForChance(p PlayerAttributes, ct ChanceType) float64 {
	if profile, ok := chanceTypeProfiles[ct]; ok && profile.AttackScore != nil {
		return profile.AttackScore(p)
	}
	return defaultAttackScore(p)
}

// rawDefense returns a player's defense score from raw attributes only.
//
// Goalkeepers are scored on their GoalkeeperRating + Recovery — saves are
// the goalkeeper's contribution to the defensive total, and tackling
// doesn't apply. Goalkeeper scoring is tactic-agnostic.
//
// Outfield defenders combine DefenseRating (positioning / awareness),
// Tackling (the actual dispossess / intercept work), and Recovery (the
// physical chase). The line-height tactic shifts the weights: a high line
// leans on Recovery (chase balls in behind), a deep line leans on
// DefenseRating + Tackling (positioning + duels). Neutral falls back to the
// balanced legacy weighting.
func rawDefense(p PlayerAttributes, tactics Tactics) float64 {
	if isGoalkeeper(p) {
		return weighted(p.GoalkeeperRating, p.EffectiveRecovery(),
			tuning.DefenseSkillWeight, tuning.DefenseSpeedWeight, tuning.DefenseDivisor)
	}
	w := tuning.DefenseWeightsForLineHeight(string(tactics.LineHeight))
	num := p.DefenseRating*w.Skill + p.EffectiveTackling()*w.Tackling + p.EffectiveRecovery()*w.Recovery
	return math.Round(float64(num) / float64(w.Divisor))
}

// playerControl applies the skill curve, out-of-position penalty, injury
// reduction, and any role-based bonus to a raw control score. This is what
// the engine actually consumes when summing a team's contribution from each
// player. The team's own Tactics drives the underlying attribute weighting
// (press level shifts skill ↔ work rate emphasis).
//
// The curve is applied at the player level so that elite vs average player
// differentials are amplified before team-level aggregation and chance
// resolution — see tuning.SkillCurve for the rationale.
func playerControl(sp SelectedPlayer, tactics Tactics) float64 {
	score := adjustForState(sp, tuning.SkillCurve(rawControl(sp.Attributes, tactics)))
	if sp.Role == PlayerRolePlaymaker {
		score *= 1.10
	}
	return score
}

// playerAttack returns the chance-type-agnostic attack score (open-play
// weights). Kept for tests and any caller that doesn't yet know the chance
// type. Engine call sites should use playerAttackForChance instead.
func playerAttack(sp SelectedPlayer) float64 {
	return adjustForState(sp, tuning.SkillCurve(rawAttack(sp.Attributes)))
}

// playerAttackForChance applies the curve, state adjustments, and roles to
// the chance-type-specific raw attack score. This is what makes player builds
// matter: the same player produces different scores on a corner vs a 1-on-1.
func playerAttackForChance(sp SelectedPlayer, ct ChanceType) float64 {
	return adjustForState(sp, tuning.SkillCurve(rawAttackForChance(sp.Attributes, ct)))
}

// playerDefense applies the skill curve + state adjustments to the raw
// defense score. The team's Tactics (specifically LineHeight) shifts the
// underlying attribute weighting between positioning and recovery.
func playerDefense(sp SelectedPlayer, tactics Tactics) float64 {
	score := adjustForState(sp, tuning.SkillCurve(rawDefense(sp.Attributes, tactics)))
	if sp.Role == PlayerRoleBallWinner {
		score *= 1.10
	}
	return score
}

// captainBoost returns a small team-wide multiplier when the lineup carries
// a captain. Returns 1.0 if no captain is present.
func captainBoost(lineup GameLineup) float64 {
	for _, p := range lineup.Players {
		if p.Role == PlayerRoleCaptain {
			return 1.03
		}
	}
	return 1.0
}

// adjustForState scales a raw score by the out-of-position penalty (if the
// player is slotted somewhere they can't play) and the injury multiplier
// (if they're carrying a high-severity injury).
func adjustForState(sp SelectedPlayer, raw float64) float64 {
	score := raw
	if sp.IsOutOfPosition() {
		score *= tuning.OutOfPositionScale
	}
	score *= injuryScale(sp.Injury)
	return score
}

// injuryScale returns the stat-reduction multiplier for an injury event.
// v1 only applied the multiplier for high-severity injuries (StatsReduction
// < StatsReductionHighSeverity, which is identity-comparison nonsense). v2
// applies whatever multiplier the injury declares — low-severity injuries
// nick a few percent off, high-severity take a bigger chunk. If there's no
// injury, returns 1.0.
func injuryScale(e *InjuryEvent) float64 {
	if e == nil {
		return 1.0
	}
	if e.Injury.StatsReduction <= 0 || e.Injury.StatsReduction > 1 {
		return 1.0
	}
	return e.Injury.StatsReduction
}

// weighted computes (skill*skillW + physical*physW) / divisor and rounds.
// Rounding mirrors v1 so attribute scores stay integer-valued for display.
func weighted(skill, physical, skillW, physW, divisor int) float64 {
	return math.Round(float64(skill*skillW+physical*physW) / float64(divisor))
}

func isGoalkeeper(p PlayerAttributes) bool {
	if p.PrimaryPosition == PlayerPositionGoalkeeper {
		return true
	}
	for _, pos := range p.Positions {
		if pos == PlayerPositionGoalkeeper {
			return true
		}
	}
	return false
}

// teamControl is the position-weighted average of each player's control
// score, computed under the team's own tactics. v1 special-cased the Box
// formation (60% attacker weight); v2 uses the same uniform weights for
// every formation — formation differences live in their tuning profiles,
// not in scoring.
func teamControl(lineup GameLineup) float64 {
	tactics := lineup.Team.Tactics
	return positionWeightedAverage(lineup.Players, tuning.ControlPositionWeights, func(sp SelectedPlayer) float64 {
		return playerControl(sp, tactics)
	})
}

func teamDefense(lineup GameLineup) float64 {
	tactics := lineup.Team.Tactics
	return positionWeightedAverage(lineup.Players, tuning.DefensePositionWeights, func(sp SelectedPlayer) float64 {
		return playerDefense(sp, tactics)
	})
}

// positionWeightedAverage groups players by their selected position, takes
// the mean score within each group, then combines those means using the
// supplied position weights. A position with no players contributes zero.
func positionWeightedAverage(players []SelectedPlayer, w tuning.PositionWeights, score func(SelectedPlayer) float64) float64 {
	var gkSum, defSum, midSum, atkSum float64
	var gkN, defN, midN, atkN int
	for _, p := range players {
		s := score(p)
		switch p.SelectedPosition {
		case PlayerPositionGoalkeeper:
			gkSum += s
			gkN++
		case PlayerPositionDefense:
			defSum += s
			defN++
		case PlayerPositionMidfield:
			midSum += s
			midN++
		case PlayerPositionAttack:
			atkSum += s
			atkN++
		}
	}
	return mean(gkSum, gkN)*w.Goalkeeper +
		mean(defSum, defN)*w.Defense +
		mean(midSum, midN)*w.Midfield +
		mean(atkSum, atkN)*w.Attack
}

func mean(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
