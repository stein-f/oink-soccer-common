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
// by volume. Set for about 3.6 goals and 13 shots a match on real line-ups.
var StepsPossessions = ChanceRange{Min: 20, Max: 27}

// StepsDuelExponent sets how much a rating edge is worth in one contest.
// At 3.5 a player rated 90 beats one rated 70 about 71% of the time, and a
// 60 against a 75 about 31%: clearly favoured, never a foregone conclusion.
// A possession is three or four contests, so edges compound across it.
const StepsDuelExponent = 3.5

// Home advantage: the home side has a little more of the ball. Its players
// get no edge of their own in a contest (StepsHomeDuel 1.0): lost-pigs
// already gives the home line-up a team boost, which reaches every contest.
const (
	StepsHomePossession = 1.22
	StepsHomeDuel       = 1.00
)

// StepsPossessionExponent sets how much of the ball the better side sees. A
// side's possession weight is the mean overall rating of the players it
// fields, raised to this power. At 12, a side three points better on average
// has about 61% of the attacks and one eight points better about 77%.
//
// This is what makes the stronger side win. With the ball shared evenly a
// much better side still gets only half the attacks, and results are far
// more of a coin flip than the classic model's, where a dominant side had
// nearly all of the ball. Overall rating is used, rather than the passing
// contest that starts each attack, because it is the number on a player's
// card: managers buy on it, and it must stay the best single guide to who
// wins. It also knows nothing about formations, so it cannot favour one.
const StepsPossessionExponent = 12.0

// StepsScoreEffect: per goal of lead, the leader's share of the ball drops by
// this much and the trailer's rises by it, capped at StepsScoreEffectCap.
// Kept small. The contest model used 0.10, which hands a beaten side the ball
// until it draws level: realistic, but it turns wins for the better side into
// draws, and the better side winning is what this game wants most.
const (
	StepsScoreEffect    = 0.0
	StepsScoreEffectCap = 0.0
)

// StepsBoostExponent softens the team-wide multipliers (captain, items, the
// position boosts, home advantage) before they reach a contest's odds. They
// were sized for the classic model, where a +5% item touched possession once
// and a shot once. Here the same multiplier is in every contest of an attack
// and every contest defending one, so at full strength a +5% item would be
// worth several times what it is sold as.
const StepsBoostExponent = 0.62

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
	StepsSupport   = 1.15
	StepsStretched = 0.80
	StepsLineCover = 0.48
)

// StepsMidfieldNumbers scales the spare-man and stretched effects where the
// pass forward is contested. Every attack goes through that contest and only
// some reach each one in the final third, so at full strength a man over in
// midfield is worth a little more than a man over anywhere else. Tuned with
// the formations gate so the two-midfielder shape is level with the rest.
const StepsMidfieldNumbers = 0.9

// A Ball Winner goes to the ball. Fielded in a stretched line he makes up
// StepsBallWinnerCover of the ground it is short (0.5: half the stretch),
// and he gives away fouls StepsBallWinnerFoul times as often. Worth it for a
// good tackler in a one-man defence; a liability for a clumsy one.
const (
	StepsBallWinnerCover = 0.5
	StepsBallWinnerFoul  = 1.35
)

// The defending line's height, where the pass forward is contested: a high
// line squeezes the space the passer has, a deep one gives it up. The price
// of a high line is paid at the other end, when a forward runs in behind it
// (StepsRunVsHighLine and the odds beside it), so which line suits a side
// depends on how quick its defenders are.
const (
	StepsPassVsHighLine = 0.90
	StepsPassVsDeepLine = 1.10
)

// StepsCoverScale is a second defender's weight when he covers for a
// team-mate who has just been beaten: a forward who gets past his man, or in
// behind him, still has the other defender to deal with if there is one.
// Below 1 because the covering man arrives late. This is what a second
// defender is for: a one-man defence, once beaten, is beaten.
const StepsCoverScale = 0.55

// StepsOutOfPosition multiplies the odds of a player fielded where he can't
// play.
const StepsOutOfPosition = 0.80

// --- Build-up ----------------------------------------------------------------

// StepsInterceptScale weights the man trying to cut out a pass: below 1,
// because most passes arrive.
const StepsInterceptScale = 0.78

// StepsLongBall multiplies the passer's odds when the ball goes from the
// defence straight to the forwards. A side with no midfield builds this way.
const StepsLongBall = 0.98

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
	// A run from deep, past a midfielder and then the defence. Rare for
	// most; a quick, strong dribbler up against slower men does it often.
	"solo": 0.6,
}

// Who does what. Every kind of attack is shared out between the players who
// could make it by what suits each of them and whether it would work.
//
//   - StepsTypicalScore is the score an ordinary top-division player has at
//     the thing he is picked for. Scores are measured against it.
//   - StepsAptitude sets how much of a kind of attack goes to the player it
//     suits best. At 4 a 90 does it about 2.7 times as often as a 70 beside
//     him: the tricky winger takes people on, the big man gets the crosses.
//   - StepsChoiceByOdds does the same for whether it would work against the
//     man each of them faces: a slow forward doesn't run at a quick defender,
//     but he will at one slower still.
//   - StepsAptitudeMix sets how much a side leans towards the kinds of attack
//     its players are good at, over the ones they aren't, and StepsOddsMix
//     how much towards the ones that would work against this opponent.
const (
	StepsTypicalScore = 72.0
	StepsAptitude     = 4.0
	StepsChoiceByOdds = 1.0
	StepsAptitudeMix  = 3.0
	StepsOddsMix      = 1.5
)

// The defending line's height changes how often a side looks to run in
// behind it, and how the race goes.
const (
	StepsRunVsHighLine     = 1.35
	StepsRunVsDeepLine     = 0.75
	StepsRunOddsVsHighLine = 1.10
	StepsRunOddsVsDeepLine = 0.92
)

// StepsCutBack is how often a forward who has beaten his man squares it to a
// team-mate instead of shooting himself, when the two finish equally well.
// It is scaled by how much better (or worse) the team-mate is, squared,
// between StepsCutBackLeast and StepsCutBackMost times: a poor finisher looks
// for a better one, a good one shoots.
const (
	StepsCutBack      = 0.25
	StepsCutBackLeast = 0.4
	StepsCutBackMost  = 2.5
)

// StepsMidfieldAttackShare is the share of a side's take-ons and runs that
// its midfielders make; the forwards make the rest between them. It is a
// share of the side's attacks, not a weight per player, so a lone striker
// carries as much of his side's attack as two strikers carry of theirs, and
// a shape with one forward doesn't end up attacking mostly with midfielders.
const StepsMidfieldAttackShare = 0.25

// Of a defender's fouls on a player taking him on: the share inside the box
// (a penalty) and the share close enough to shoot from (a direct free
// kick). The rest restart play without a shot.
const (
	StepsPenaltyShareOfFouls  = 0.26
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
	"Open Play":       {Block: 0.32, OnTarget: 58, Keeper: 1.25},
	"Cross":           {Block: 0, OnTarget: 66, Keeper: 1.25},
	"Corner":          {Block: 0, OnTarget: 70, Keeper: 1.30},
	"Long Range":      {Block: 0.45, OnTarget: 90, Keeper: 1.55},
	"Free Kick":       {Block: 0.30, OnTarget: 84, Keeper: 1.45},
	"Penalty":         {Block: 0, OnTarget: 32, Keeper: 0.60},
	"Goalkeeper Shot": {Block: 0, OnTarget: 52, Keeper: 1.18},
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
	StepsReboundAfterParry    = 0.18
	// Off the post or the bar, how often it comes back out to an attacker.
	StepsReboundAfterWoodwork = 0.30
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

// Who gets to a rebound. It follows a player's sense for where the ball will
// drop (see poachScore), and where he plays: a forward is in the box already
// (StepsPoacherAttack), a defender almost never is (StepsPoacherDefense).
// From a corner everybody is up, and neither applies.
const (
	StepsPoacherAttack  = 3.0
	StepsPoacherDefense = 0.15
)

// A side a goal down from StepsKeeperUpFromMinute sends its keeper up for
// corners. StepsKeeperUpWeight is how often the delivery finds him, against
// an ordinary outfielder's 1, and StepsKeeperUpHeader is the least his header
// is worth as a shot: he is a big man nobody planned for. Keeper goals are
// meant to happen a few times a season, not a few times a round.
const (
	StepsKeeperUpFromMinute = 82
	StepsKeeperUpWeight     = 4.0
	StepsKeeperUpHeader     = 66.0
)

// A goal from distance is a "worldie" when it was struck with enough power
// (see worldieChance): never below a shot-power blend of StepsWorldieFrom,
// and StepsWorldieShare of the time for a perfect striker. It is a label for
// the replay and the record, not a change to the odds.
const (
	StepsWorldieFrom  = 50.0
	StepsWorldieShare = 1.0
)
