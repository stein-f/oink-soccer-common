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
	return weighted(p.AttackRating, p.SpeedRating,
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
		return weighted(p.GoalkeeperRating, p.SpeedRating,
			tuning.DefenseSkillWeight, tuning.DefenseSpeedWeight, tuning.DefenseDivisor)
	}
	w := tuning.DefenseWeightsForLineHeight(string(tactics.LineHeight))
	num := p.DefenseRating*w.Skill + p.EffectiveTackling()*w.Tackling + p.SpeedRating*w.Recovery
	return math.Round(float64(num) / float64(w.Divisor))
}

// playerControl applies the skill curve, out-of-position penalty, and injury
// reduction to a raw control score. This is what the engine actually consumes
// when summing a team's contribution from each player. The team's own Tactics
// drives the underlying attribute weighting (press level shifts skill ↔ work
// rate emphasis).
//
// The Playmaker role is *not* a per-player multiplier here — it's a focal-
// point lever applied at the team-aggregation step (see teamControl), where
// the playmaker's score is weighted more heavily within their position group.
// This means tagging a poor player as Playmaker drags the team's control
// down rather than giving them a free boost.
//
// The curve is applied at the player level so that elite vs average player
// differentials are amplified before team-level aggregation and chance
// resolution — see tuning.SkillCurve for the rationale.
func playerControl(sp SelectedPlayer, tactics Tactics) float64 {
	return adjustForState(sp, tuning.SkillCurve(rawControl(sp.Attributes, tactics)))
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
//
// The Ball Winner role is *not* a per-player multiplier here — like
// Playmaker, it's a focal-point lever applied at team aggregation (see
// teamDefense), where the Ball Winner's score is weighted more heavily
// within their position group. Tagging a poor defender drags the team's
// defense down rather than giving them a free boost.
func playerDefense(sp SelectedPlayer, tactics Tactics) float64 {
	return adjustForState(sp, tuning.SkillCurve(rawDefense(sp.Attributes, tactics)))
}

// captainBoost returns the team-wide multiplier (control + defense) driven
// by the captain's quality. Returns 1.0 if no captain is present.
//
// A quality captain lifts the team; a poor captain drags it. The scaling
// is small (≈ ±2.4% at the extremes) — captain identity matters but doesn't
// dominate. See tuning.CaptainTeamBoost for the formula.
func captainBoost(lineup GameLineup) float64 {
	for _, p := range lineup.Players {
		if p.Role == PlayerRoleCaptain {
			return tuning.CaptainTeamBoost(captainQuality(p.Attributes))
		}
	}
	return 1.0
}

// captainQuality returns a 0-100 score representing how well this player
// fits the captain role. Drives both team-wide and self multipliers.
//
// The proxy: a player's primary skill (GoalkeeperRating for keepers,
// ControlRating for outfielders — game intelligence + decision-making)
// averaged with EffectiveComposure (calm under pressure). When Composure
// isn't set, EffectiveComposure backfills to ControlRating, which keeps
// legacy rosters from being penalised arbitrarily.
//
// Any player can be captain; we rank them by what we can measure. A
// composed, intelligent senior is the archetype.
func captainQuality(p PlayerAttributes) int {
	primary := p.ControlRating
	if isGoalkeeper(p) {
		primary = p.GoalkeeperRating
	}
	return (primary + p.EffectiveComposure()) / 2
}

// adjustForState scales a raw score by the out-of-position penalty (if the
// player is slotted somewhere they can't play), the injury multiplier (if
// they're carrying an injury), and the captain self-boost (if this player
// wears the armband — small ± based on captain quality, since a poor
// captain feels the burden while a quality captain plays above themselves).
func adjustForState(sp SelectedPlayer, raw float64) float64 {
	score := raw
	if sp.IsOutOfPosition() {
		score *= tuning.OutOfPositionScale
	}
	score *= injuryScale(sp.Injury)
	if sp.Role == PlayerRoleCaptain {
		score *= tuning.CaptainSelfBoost(captainQuality(sp.Attributes))
	}
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
//
// Playmakers contribute to their position group's mean with extra weight
// (tuning.PlaymakerControlWeight) — they're the focal point of the team's
// possession. A high-rated Playmaker drags the group's mean toward their
// score; a low-rated Playmaker drags it the other way. This makes Playmaker
// a real choice instead of a free boost: tag your best controller and you
// gain, tag a weak player and you lose.
func teamControl(lineup GameLineup) float64 {
	tactics := lineup.Team.Tactics
	return rolePositionAverage(lineup.Players, tuning.ControlPositionWeights, func(sp SelectedPlayer) (float64, float64) {
		score := playerControl(sp, tactics)
		weight := 1.0
		if sp.Role == PlayerRolePlaymaker {
			weight = tuning.PlaymakerControlWeight
		}
		return score, weight
	})
}

// teamDefense mirrors teamControl's structure: position-weighted average
// of player defense scores, with a focal-point bump for Ball Winners.
// A quality Ball Winner amplifies the team's defensive shape; a weak one
// drags it down — exactly like Playmaker on the control side.
func teamDefense(lineup GameLineup) float64 {
	tactics := lineup.Team.Tactics
	return rolePositionAverage(lineup.Players, tuning.DefensePositionWeights, func(sp SelectedPlayer) (float64, float64) {
		score := playerDefense(sp, tactics)
		weight := 1.0
		if sp.Role == PlayerRoleBallWinner {
			weight = tuning.BallWinnerDefenseWeight
		}
		return score, weight
	})
}

// rolePositionAverage groups players by their selected position. Each player
// contributes (score, weight) to their group; the group's contribution is
// sum(score*weight) / sum(weight). Position groups are then combined using
// the supplied position weights.
//
// All players carrying weight 1.0 reduces to a simple mean — the same shape
// as a position-weighted average. Roles like Playmaker (in teamControl) and
// Ball Winner (in teamDefense) use heavier weights to act as focal points
// within their group.
func rolePositionAverage(players []SelectedPlayer, w tuning.PositionWeights, get func(SelectedPlayer) (float64, float64)) float64 {
	type bucket struct{ sum, totalW float64 }
	var gk, def, mid, atk bucket
	for _, p := range players {
		score, weight := get(p)
		b := &gk
		switch p.SelectedPosition {
		case PlayerPositionDefense:
			b = &def
		case PlayerPositionMidfield:
			b = &mid
		case PlayerPositionAttack:
			b = &atk
		}
		b.sum += score * weight
		b.totalW += weight
	}
	return weightedMean(gk.sum, gk.totalW)*w.Goalkeeper +
		weightedMean(def.sum, def.totalW)*w.Defense +
		weightedMean(mid.sum, mid.totalW)*w.Midfield +
		weightedMean(atk.sum, atk.totalW)*w.Attack
}

func weightedMean(sum, totalW float64) float64 {
	if totalW == 0 {
		return 0
	}
	return sum / totalW
}
