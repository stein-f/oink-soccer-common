// Package tuning centralises every numeric constant that shapes the engine's
// behavior. v1 sprinkled these across game.go, team.go, boosts.go and inline
// position-weight maps, which made balance changes hard to reason about. In
// v2 every value lives here, named, documented, and grouped by concern.
//
// This package is internal — most values are implementation details. The few
// constants that need to be visible to downstream consumers (e.g.
// DRDecayPerApplication, used by lost-pigs to render boost-decay UI) are
// re-exported from the root soccer package via aliases so that changing the
// underlying value still requires updating the public name.
package tuning

import "math"

// --- Boost decay -------------------------------------------------------------

// BoostDecay is the per-application decay multiplier for stacked boosts.
// apps=0 → 1.0×, apps=1 → 0.97×, apps=2 → 0.97², …
const BoostDecay = 0.97

// BoostMinMultiplier is a floor so a heavily-stacked boost never decays into
// uselessness or, worse, a debuff.
const BoostMinMultiplier = 0.35

// --- Player out-of-position penalty -----------------------------------------

// OutOfPositionScale is applied to a player's score when they are slotted
// into a position they don't list as playable. v2 keeps this flat for now;
// Phase 5 may make it position-distance-aware (e.g. GK→ATK harsher than
// DEF→MID).
const OutOfPositionScale = 0.85

// --- Injury severity stat reductions ----------------------------------------

// StatsReductionLow / Med / High describe how much an injured player's
// active-match stats are scaled by while carrying an injury of that severity.
// Lower number = larger penalty.
const (
	StatsReductionLow  = 0.95
	StatsReductionMed  = 0.90
	StatsReductionHigh = 0.85
)

// --- Injury probability weights ---------------------------------------------

// Per-player injury chance is rolled as weighted choice between
// {false: NoInjuryWeight, true: 1}. Higher NoInjuryWeight ⇒ rarer injuries.
const (
	NoInjuryWeightDefault     = 30
	NoInjuryWeightInjuryProne = 15 // injury-prone players ⇒ ~2× base injury rate
)

// AggressionMaxNoInjuryReduction caps how much the opponent's aggression can
// reduce the "no injury" weight, expressed as a fraction. 0.5 means
// aggression=100 cuts the no-injury weight in half (doubling injury odds).
const AggressionMaxNoInjuryReduction = 0.5

// --- Defensive bias ---------------------------------------------------------

// DefenseBiasMultiplier was a v1 fudge factor multiplied onto every team's
// defense score to lower scoring rates. v2 keeps it for parity during Phase 1
// but should be removed in Phase 4 when the phase-based simulation gives us
// a principled way to tune scoring volume via tempo + chance quality.
const DefenseBiasMultiplier = 1.05

// --- Player attribute weighting ---------------------------------------------

// Per-attribute weights used to derive a player's per-action score from raw
// ratings. v1 folded SpeedRating into all three (control, attack, defense),
// triple-counting it. Phase 2 will split this into pace / recovery / work
// rate. The current values mirror v1.
//
// Weight tuples are (skill weight, speed weight, divisor).
//
//	control = (controlRating*4 + speedRating) / 5
//	attack  = (attackRating*3 + speedRating) / 4
//	defense = (defenseRating*5 + speedRating) / 6
const (
	ControlSkillWeight = 4
	ControlSpeedWeight = 1
	ControlDivisor     = 5

	AttackSkillWeight = 3
	AttackSpeedWeight = 1
	AttackDivisor     = 4

	DefenseSkillWeight    = 5
	DefenseTacklingWeight = 2
	DefenseSpeedWeight    = 1
	DefenseDivisor        = 6 // legacy GK formula (defense*5 + recovery) / 6
)

// --- Tactic-driven attribute weights ----------------------------------------

// ControlWeights describes how a player's control score is built from their
// ControlRating (skill) and effective work rate (physical). Pressing teams
// shift weight toward work rate; passive teams lean further on skill. The
// neutral baseline mirrors the legacy ControlSkillWeight/ControlSpeedWeight.
type ControlWeights struct {
	Skill    int
	Physical int
	Divisor  int
}

// ControlWeightsForPress returns the (skill, physical, divisor) tuple used
// to build a midfielder's control score given the team's press level.
//
//   - High press: defenders sprint forward, midfielders close space — work
//     rate matters as much as technique.
//   - Medium / none: legacy weighting (skill-heavy).
//   - Low press: passive, possession-based — pure skill dominates.
func ControlWeightsForPress(pressLevel string) ControlWeights {
	switch pressLevel {
	case "high":
		return ControlWeights{Skill: 3, Physical: 2, Divisor: 5}
	case "low":
		return ControlWeights{Skill: 5, Physical: 1, Divisor: 6}
	default: // none, medium
		return ControlWeights{Skill: ControlSkillWeight, Physical: ControlSpeedWeight, Divisor: ControlDivisor}
	}
}

// DefenseWeights describes how an outfield defender's score is built from
// DefenseRating (positioning), Tackling (active dispossess), and Recovery
// (chase / catch-up speed). Goalkeepers don't use this — they stay on the
// legacy GoalkeeperRating + Recovery formula.
type DefenseWeights struct {
	Skill    int // DefenseRating
	Tackling int
	Recovery int
	Divisor  int
}

// DefenseWeightsForLineHeight returns the outfield defense weighting given
// the team's defensive line height.
//
//   - High line: defenders are exposed to balls in behind — Recovery
//     (sprint back to recover) matters far more than positioning.
//   - Deep line: positioning + tackling dominate; pace is rarely tested.
//   - Normal / none: balanced (legacy weighting).
func DefenseWeightsForLineHeight(lineHeight string) DefenseWeights {
	switch lineHeight {
	case "high":
		return DefenseWeights{Skill: 3, Tackling: 2, Recovery: 3, Divisor: 8}
	case "deep":
		return DefenseWeights{Skill: 6, Tackling: 2, Recovery: 0, Divisor: 8}
	default: // none, normal
		return DefenseWeights{Skill: DefenseSkillWeight, Tackling: DefenseTacklingWeight, Recovery: DefenseSpeedWeight, Divisor: DefenseSkillWeight + DefenseTacklingWeight + DefenseSpeedWeight}
	}
}

// --- Skill scaling curve ----------------------------------------------------

// SkillCurveExponent shapes the convex curve applied to a player's per-action
// raw score (control / attack / defense). Higher exponent ⇒ wider gap between
// elite and average players; lower exponent collapses toward the linear
// weighted average.
//
// v1 used a hand-tuned 100-row CSV (see scaling.csv) approximating y = ax^b
// with b ≈ 3.1. v2 uses the closed form (rating/100)^k × 100 — same convex
// shape, no embedded data file. Reference points at k=4.0:
//
//	rating  scaled
//	   50     6.3
//	   70    24.0
//	   80    41.0
//	   85    52.2
//	   90    65.6
//	   95    81.5
//	  100   100.0
//
// Without the curve, raw weighted-average scores cluster too tightly: an
// 87-rated team beats a 78-rated team only ~46% / 31% of the time. With
// k=4.0 it's ~69% / 14% — close to real-world intuition that a clear
// rating gap produces a clear favourite. See skill_gap_test.go for the
// full probe.
const SkillCurveExponent = 4.0

// SkillCurveFloor is the minimum value the curve returns. Avoids divide-by-
// zero in chance resolution if a player has a near-zero raw score.
const SkillCurveFloor = 1.0

// SkillCurve maps a 0-100 raw score onto its convex-scaled counterpart.
// Used by player-level scoring helpers (rawControl/Attack/Defense) before
// state adjustments and role multipliers.
func SkillCurve(raw float64) float64 {
	if raw <= 0 {
		return SkillCurveFloor
	}
	if raw >= 100 {
		return 100
	}
	scaled := math.Pow(raw/100.0, SkillCurveExponent) * 100.0
	if scaled < SkillCurveFloor {
		return SkillCurveFloor
	}
	return scaled
}

// --- Team scoring position weights ------------------------------------------

// PositionWeights describes how much each on-pitch position contributes to a
// team-level score (control, defense). Values must sum to 1.0 to keep the
// team score on a 0-100 scale. Scoring renormalizes over the position groups
// a formation actually fields, so a shape with no midfielders (The Box)
// spreads the midfield weight across the rest rather than scoring it zero.
//
// v1 special-cased the Box formation here (60% attacker weight in control)
// which was a major source of formation imbalance. v2 uses the same weights
// for every formation; tactical differences live in tuning.FormationProfile
// instead (Phase 3).
type PositionWeights struct {
	Goalkeeper float64
	Defense    float64
	Midfield   float64
	Attack     float64
}

var (
	ControlPositionWeights = PositionWeights{Goalkeeper: 0.05, Defense: 0.15, Midfield: 0.65, Attack: 0.15}
	DefensePositionWeights = PositionWeights{Goalkeeper: 0.35, Defense: 0.40, Midfield: 0.20, Attack: 0.05}
)

// --- Attacking-player selection weights -------------------------------------

// AttackerPickWeights describes how likely each on-pitch position is to be
// the player who receives a chance. Heavily favours attackers, then mids.
// v1 had a Box-formation override that zeroed midfielders and gave attackers
// 88; v2 drops the override (Box has no midfielders so the value is moot).
var AttackerPickWeights = map[string]uint{
	"Goalkeeper": 2,
	"Defense":    10,
	"Midfield":   20,
	"Attack":     70,
}

// --- Role focal-point weighting ---------------------------------------------

// PlaymakerControlWeight / BallWinnerDefenseWeight set the relative weight a
// Playmaker (in teamControl) or Ball Winner (in teamDefense) carries within
// their position group when computing the team's score. Other players in
// the same group carry weight 1.0.
//
// Mechanics: aggregation is mean-within-position, then position-weighted.
// Bumping the role-holder's intra-group weight to 2.0 drags the group's
// mean toward their score. Net swing for a 3-player position group: ≈
// ±3.3 points on the group mean depending on whether the role-holder rates
// above or below the position average.
//
// This replaced earlier flat per-player ×1.10 boosts, which gave even
// poorly-rated role-holders a free lift. The weighting model makes both
// roles a real focal-point choice — tag your best and gain, tag your
// worst and lose.
const (
	PlaymakerControlWeight  = 2.0
	BallWinnerDefenseWeight = 2.0
)

// --- Captain quality scaling ------------------------------------------------

// A captain's quality drives two effects, both small:
//
//  1. CaptainTeamBoost — multiplier applied to the team's control + defense.
//     A quality captain lifts the team's average performance; a poor
//     captain (low game-reading + low composure) drags it down.
//  2. CaptainSelfBoost — multiplier applied to the captain's own per-action
//     scores. Wearing the armband motivates a quality leader to play above
//     themselves; a poor captain feels the burden and underperforms.
//
// Both effects scale around `CaptainNeutralQuality` (q=60 ⇒ multiplier 1.0)
// using the same gain so the magnitudes are symmetric and easy to reason
// about. Range for q in [0, 100]: roughly [0.964, 1.024] — small.
//
// This replaced an earlier flat ×1.03 team-wide boost that was identity-
// agnostic (any player tagged gave the same lift). The quality-scaled
// model makes captain identity matter without dominating the balance.
const (
	CaptainNeutralQuality = 60
	CaptainTeamBoostGain  = 0.06
	CaptainSelfBoostGain  = 0.06
)

// CaptainTeamBoost returns the team-wide multiplier (control + defense)
// driven by a captain of the given quality. quality is on a 0-100 scale.
func CaptainTeamBoost(quality int) float64 {
	return 1.0 + float64(quality-CaptainNeutralQuality)/100.0*CaptainTeamBoostGain
}

// CaptainSelfBoost returns the per-action multiplier on the captain's own
// score. Same shape as CaptainTeamBoost — a poor captain is a real drag on
// their own play, a strong captain plays above themselves.
func CaptainSelfBoost(quality int) float64 {
	return 1.0 + float64(quality-CaptainNeutralQuality)/100.0*CaptainSelfBoostGain
}

// --- Corner delivery quality (named SetPieceTaker) --------------------------

// Corner finisher selection is independent of who delivers the corner — the
// finisher is picked by Heading via the standard chance-type selection (see
// chance.go). What a named SetPieceTaker contributes on a corner is *delivery
// quality*: a great taker produces a more dangerous ball into the box,
// raising the chance of any header converting; a poor taker reduces it.
//
// CornerDeliveryFactor scales the corner's effective AttackBoost using the
// taker's EffectiveTechnique. Reference points:
//
//	technique  factor
//	    100    1.16
//	     85    1.10
//	     60    1.00 (neutral baseline — average pro)
//	     40    0.92
//	     20    0.84
//
// Neutral=60 means a backfilled midfielder (no explicit Technique → falls back
// to ControlRating, typically 70-85) still delivers above-neutral, while a
// deliberately-named poor taker is a real penalty (no free lunch). Values
// outside [0.80, 1.20] are clamped to bound the swing.
const (
	CornerDeliveryNeutralTechnique = 60
	CornerDeliveryGain             = 0.40
	CornerDeliveryMinFactor        = 0.80
	CornerDeliveryMaxFactor        = 1.20
)

// CornerDeliveryFactor returns the multiplier applied to a corner's AttackBoost
// when a SetPieceTaker is named. technique should be the taker's
// EffectiveTechnique (Technique attribute, falling back to ControlRating).
func CornerDeliveryFactor(technique int) float64 {
	f := 1.0 + float64(technique-CornerDeliveryNeutralTechnique)/100.0*CornerDeliveryGain
	if f < CornerDeliveryMinFactor {
		return CornerDeliveryMinFactor
	}
	if f > CornerDeliveryMaxFactor {
		return CornerDeliveryMaxFactor
	}
	return f
}

// --- Match-tempo (number of chances) ----------------------------------------

// ChanceRange describes the inclusive [Min, Max] number of chances a match
// will produce, indexed by HomeStyle|AwayStyle. v1's directional table is
// preserved here verbatim so Phase 1 doesn't change game outcomes.
type ChanceRange struct{ Min, Max int }

var FormationChanceRanges = map[string]ChanceRange{
	"HOME:ATT|AWAY:ATT": {Min: 7, Max: 15},
	"HOME:ATT|AWAY:BAL": {Min: 6, Max: 12},
	"HOME:ATT|AWAY:DEF": {Min: 5, Max: 11},

	"HOME:BAL|AWAY:ATT": {Min: 7, Max: 12},
	"HOME:BAL|AWAY:BAL": {Min: 5, Max: 10}, // +1 across — lifts low-volume matchups so home advantage isn't starved
	"HOME:BAL|AWAY:DEF": {Min: 5, Max: 10}, // symmetric with HOME:DEF|AWAY:BAL

	"HOME:DEF|AWAY:ATT": {Min: 6, Max: 11},
	"HOME:DEF|AWAY:BAL": {Min: 5, Max: 10},
	"HOME:DEF|AWAY:DEF": {Min: 5, Max: 9}, // bumped further — Pyramid v Pyramid was the spread floor
}

// FallbackChanceRange is used if a formation-pair isn't in the table above.
var FallbackChanceRange = ChanceRange{Min: 3, Max: 10}

// --- Event-minute distribution ----------------------------------------------

// EventMinuteBucket weights when in the match an event is likely to occur.
// Mirrors v1: late-game weighting reflects fatigue + pushing for goals.
type EventMinuteBucket struct {
	MinMinute int
	MaxMinute int
	Weight    uint
}

var EventMinuteBuckets = []EventMinuteBucket{
	{MinMinute: 1, MaxMinute: 15, Weight: 99},
	{MinMinute: 16, MaxMinute: 30, Weight: 158},
	{MinMinute: 31, MaxMinute: 45, Weight: 142},
	{MinMinute: 46, MaxMinute: 60, Weight: 178},
	{MinMinute: 61, MaxMinute: 75, Weight: 168},
	{MinMinute: 76, MaxMinute: 98, Weight: 254},
}

// --- Formation balance profiles ---------------------------------------------

// FormationProfile is the trade-off matrix for a tactical shape. Every value
// is a multiplier against neutral 1.0; deviating in one axis must be paid for
// in another so that no formation strictly dominates another.
//
// Axes:
//   - Possession      — multiplier on team control (more possession ⇒ more chances created on average)
//   - ChanceCreation  — multiplier on the number of chances the team generates per minute
//   - ChanceQuality   — multiplier on conversion probability per chance (better positions, cleaner shots)
//   - DefSolidity     — multiplier on team defense
//   - InjuryRisk      — multiplier on per-player injury probability (higher = riskier shape)
//
// Phase 4's balance test will tune these via 5k-game simulations until every
// (home × away) matchup produces win-rates within ±3% (Q4).
type FormationProfile struct {
	Possession     float64
	ChanceCreation float64
	ChanceQuality  float64
	DefSolidity    float64
	InjuryRisk     float64
}

// FormationProfiles is keyed by string(FormationType) so this package stays
// import-cycle-free of the public soccer package.
//
// Initial values are deliberate trade-offs (see comments). They will be tuned
// in Phase 4 once the engine exists and we can run the balance harness.
var FormationProfiles = map[string]FormationProfile{
	// "The Pyramid" (2-1-1) — defensive shape; relies on defensive solidity
	// + clinical counter-attacks. Neutral chance creation (the low total
	// volume comes from the formation-style chance range, not the profile)
	// and a modest quality bonus that turns rare attacks into real threats.
	"The Pyramid": {Possession: 1.00, ChanceCreation: 1.00, ChanceQuality: 1.03, DefSolidity: 1.02, InjuryRisk: 1.00},

	// "The Diamond" (2-1-1) — balanced shape; small possession edge from
	// the extra midfielder. Sets the baseline.
	"The Diamond": {Possession: 1.03, ChanceCreation: 1.00, ChanceQuality: 1.00, DefSolidity: 1.00, InjuryRisk: 1.00},

	// "The Y" (1-1-2) — attacking shape; the second striker drives both
	// chance volume and quality. Pays in defense but the front-three
	// presence keeps even defensive opponents honest.
	"The Y": {Possession: 1.00, ChanceCreation: 1.03, ChanceQuality: 1.02, DefSolidity: 0.97, InjuryRisk: 1.00},

	// "The Box" (2-0-2) — direct-play shape; the no-midfield setup
	// produces direct attacking thrusts and chance volume from the two
	// strikers. Pays in injury risk (more transitions = more tackles).
	//
	// Possession is above neutral to offset a scoring artifact: with no
	// midfield group, weight renormalization roughly triples the keeper's
	// control weight, dragging raw team control ~14% below a comparable
	// four-group shape. The same renormalization removes the midfield's
	// weak defense from the average, so Box's raw defense lands ~19% high —
	// DefSolidity claws most of that back. Tuned against the balance
	// harness (±3% win-rate spread, RUN_BALANCE_STRICT=1).
	"The Box": {Possession: 1.12, ChanceCreation: 1.05, ChanceQuality: 1.05, DefSolidity: 0.90, InjuryRisk: 1.05},
}

// NeutralProfile is returned for unknown formations so the engine can never
// nil-deref on a profile lookup.
var NeutralProfile = FormationProfile{
	Possession: 1, ChanceCreation: 1, ChanceQuality: 1, DefSolidity: 1, InjuryRisk: 1,
}

// LookupFormationProfile returns the profile for a formation name (using
// string(FormationType)), falling back to NeutralProfile when the formation
// isn't known.
func LookupFormationProfile(name string) FormationProfile {
	if p, ok := FormationProfiles[name]; ok {
		return p
	}
	return NeutralProfile
}
