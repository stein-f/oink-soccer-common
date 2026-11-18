package tuning

// Constants for ModelSteps (season 19+). The classic and contest models'
// constants are untouched.
//
// The step model resolves each attack as a chain of contests between two
// named players, chosen by where they stand on the pitch. Every contest has
// the same shape: the attacker's odds are (a / b)^StepsDuelExponent, where a
// and b are the two players' scores for that contest (plain 0-100 blends of
// their attributes, no skill curve), times whatever modifiers apply. The
// constants below are those modifiers and the scales that set how hard each
// contest is. They are tuned with cmd/calibrate.

// StepsPossessions is how many attacks a match has (before tempo). Every
// formation gets the same number: shape is expressed by who meets whom, not
// by volume. More than the contest model's because an attack here has more
// places to break down before it becomes a shot.
var StepsPossessions = ChanceRange{Min: 24, Max: 32}

// StepsDuelExponent sets how much a rating edge is worth in one contest.
// At 3.0 a player rated 90 beats one rated 70 about 68% of the time, and a
// 60 against a 75 about 34%: clearly favoured, never a foregone conclusion.
// A possession is three or four contests, so edges compound across it.
const StepsDuelExponent = 3.0

// Home advantage: the home side has a little more of the ball and its
// players' odds are a little better in every contest.
const (
	StepsHomePossession = 1.14
	StepsHomeDuel       = 1.00
)

// StepsBoostExponent softens the team-wide multipliers (captain, items, the
// position boosts, home advantage) before they reach a contest's odds. They
// were sized for the classic model, where a +5% item touched possession once
// and a shot once. Here the same multiplier is in every contest of an attack
// and every contest defending one, so at full strength a +5% item would be
// worth several times what it is sold as.
const StepsBoostExponent = 0.5

// StepsMomentum multiplies the attacker's odds in the contest after one he
// won: the defender is beaten, the shot is from a better place.
const StepsMomentum = 1.12

// --- Who meets whom ----------------------------------------------------------

// A contest is met by the defending line that faces it. How well depends on
// numbers, and this is where formations differ. A line that outnumbers the
// attackers it faces has a spare man (StepsSupport on the defender's odds);
// one that is outnumbered is stretched (StepsStretched); a line with nobody
// in it is covered by a player from another line, out of his place
// (StepsLineCover). A side's four outfielders are an extra man somewhere and
// a man short somewhere else, whatever its shape, which is what keeps the
// formations level. Tune these on real squads (see the formations report in
// cmd/calibrate), not on one invented squad.
const (
	StepsSupport   = 1.18
	StepsStretched = 0.85
	StepsLineCover = 0.36
)

// StepsOutOfPosition multiplies the odds of a player fielded where he can't
// play.
const StepsOutOfPosition = 0.80

// --- Build-up ----------------------------------------------------------------

// StepsInterceptScale weights the man trying to cut out a pass: below 1,
// because most passes arrive.
const StepsInterceptScale = 0.78

// StepsLongBall multiplies the passer's odds when the ball goes from the
// defence straight to the forwards. A side with no midfield builds this way.
const StepsLongBall = 0.90

// StepsPlaymakerPassWeight is how much more often a Playmaker is the one who
// plays the pass; StepsTargetManWeight how much more often a Target Man is
// the one a cross is aimed at.
const (
	StepsPlaymakerPassWeight = 1.8
	StepsTargetManWeight     = 2.0
)

// --- The final third ---------------------------------------------------------

// StepsRouteWeights is the base mix of what a side tries once it has the
// ball forward. Each is then weighted by how likely it is to work for the
// players involved, so a side with a quick forward against a slow defender
// runs in behind more, without ever doing only that.
var StepsRouteWeights = map[string]float64{
	"take_on":   4.0,
	"cross":     3.0,
	"run":       2.0,
	"long_shot": 1.0,
}

// The defending line's height changes how often a side looks to run in
// behind it, and how the race goes.
const (
	StepsRunVsHighLine     = 1.6
	StepsRunVsDeepLine     = 0.6
	StepsRunOddsVsHighLine = 1.15
	StepsRunOddsVsDeepLine = 0.88
)

// StepsCutBack is how often a forward who has beaten his man squares it to a
// team-mate instead of shooting himself.
const StepsCutBack = 0.40

// StepsMidfielderForward is how much less often than a forward a midfielder
// is the one who takes a man on or runs in behind.
const StepsMidfielderForward = 0.4

// Of a defender's fouls on a player taking him on: the share inside the box
// (a penalty) and the share close enough to shoot from (a direct free
// kick). The rest restart play without a shot.
const (
	StepsPenaltyShareOfFouls  = 0.20
	StepsFreeKickShareOfFouls = 0.35
)

// Scales on the defender's score in each final-third contest. Above 1 the
// contest favours the defender between equals.
const (
	StepsTackleScale    = 1.06 // a take-on
	StepsChaseScale     = 1.12 // a run in behind
	StepsCloseDownScale = 0.62 // stopping a cross at source: most get in
	StepsMarkerScale    = 1.08 // the header in the box
)

// --- The shot ----------------------------------------------------------------

// StepsShot shapes one kind of shot, on plain 0-100 scores:
//   - Block: the most a blocker can stop; P(blocked) = Block · P(blocker beats shot)
//   - OnTarget: the score that hits the target half the time
//   - Keeper: how much the keeper's score counts once it is on target
type StepsShot struct {
	Block    float64
	OnTarget float64
	Keeper   float64
}

// StepsShots by chance type (keyed by the ChanceType string).
var StepsShots = map[string]StepsShot{
	"Open Play":       {Block: 0.32, OnTarget: 58, Keeper: 1.22},
	"Cross":           {Block: 0, OnTarget: 66, Keeper: 1.25},
	"Corner":          {Block: 0, OnTarget: 70, Keeper: 1.30},
	"Long Range":      {Block: 0.45, OnTarget: 90, Keeper: 1.55},
	"Free Kick":       {Block: 0.30, OnTarget: 84, Keeper: 1.45},
	"Penalty":         {Block: 0, OnTarget: 32, Keeper: 0.60},
	"Goalkeeper Shot": {Block: 0, OnTarget: 52, Keeper: 1.02},
}

// StepsFallbackKeeper is the keeper score when a line-up has no goalkeeper.
const StepsFallbackKeeper = 25.0

// Of shots that miss the target: the share that hit the woodwork and the
// share that go over. The rest go wide.
const (
	StepsWoodworkShare = 0.12
	StepsOverShare     = 0.33
)

// What can follow a shot or a cleared cross.
const (
	StepsCornerAfterBlock     = 0.35
	StepsCornerAfterParry     = 0.40
	StepsCornerAfterClearance = 0.22
	StepsReboundAfterParry    = 0.14
	// StepsMaxFollowUps stops a corner leading to a corner forever.
	StepsMaxFollowUps = 2
)

// StepsHeldScale weights the shot against the keeper's handling when a save
// is made: P(held) = P(handling beats shot · scale). The rest are parried.
const StepsHeldScale = 0.95

// StepsMaxStops is how many attacks stopped before a shot are reported as
// events in a match. Every stop is counted in the totals; only tackles and
// last-man recoveries are candidates for an event, and a replay shouldn't
// grow from nine clips to twenty.
const StepsMaxStops = 3
