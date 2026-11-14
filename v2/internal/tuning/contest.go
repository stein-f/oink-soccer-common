package tuning

// Constants for ModelContest (season 19+). The classic model's constants in
// tuning.go are untouched: changing them would change seasons 1-18.
//
// The contest model resolves each possession as a chain of contests between
// two named players: a carrier against a defender (build-up), a shooter
// against a blocker, then the shot against the keeper. Each contest has the
// engine's usual shape, p = a / (a + k·d), so the constants below are mostly
// the k that weights one side. They were tuned with cmd/calibrate against
// replays of real season-18 line-ups; the targets are in
// docs/realism-review.md.

// ContestPossessions is how many attacks a match has (before tempo). The
// classic model varied volume by formation style; here every formation gets
// the same volume and expresses its style through its profile (possession,
// chance creation and quality, defensive solidity), which keeps formations
// balanced once each attack is a chain of contests.
var ContestPossessions = ChanceRange{Min: 16, Max: 22}

// ContestDuelExponent softens each contest: p = a^e / (a^e + d^e). A match
// passes through three or four contests where the classic model had one, and
// with the skill curve's steep scores every one of them at full strength
// compounds into near-certain wins and near-certain shut-outs. e < 1 keeps
// the better player favoured in each duel while the chain as a whole stays
// about as decisive as the classic model.
const ContestDuelExponent = 0.45

// ContestPossessionExponent softens the possession split the same way:
// P(home has it) = h^e / (h^e + a^e). Team control comes through the skill
// curve, so a side 25 rating points better would otherwise see ~90% of the
// ball; real dominant sides have nearer 70%.
const ContestPossessionExponent = 0.5

// Home advantage inside the model: the home side gets a little more of the
// ball and plays its duels a little better. (lost-pigs also applies a team
// boost to the home side; the targets were set with that boost in place.)
const (
	ContestHomePossession = 1.25
	ContestHomeDuel       = 1.06
)

// --- Build-up ----------------------------------------------------------------

// ContestCarrierWeights picks who carries the ball forward, by position,
// before scaling by each player's control score.
var ContestCarrierWeights = PositionWeights{Goalkeeper: 0.02, Defense: 0.18, Midfield: 0.50, Attack: 0.30}

// ContestStopperWeights picks who steps in to stop an attack (and who throws
// themselves in front of a shot), by position, before scaling by each
// player's defense score. Keepers have their own contest.
var ContestStopperWeights = PositionWeights{Goalkeeper: 0, Defense: 0.45, Midfield: 0.40, Attack: 0.15}

// ContestPlaymakerCarryWeight / ContestBallWinnerStopWeight are how much more
// often a Playmaker carries the ball and a Ball Winner makes the challenge.
const (
	ContestPlaymakerCarryWeight = 1.8
	ContestBallWinnerStopWeight = 1.8
)

// ContestStopScale weights the defender in the build-up contest:
// P(stopped) = k·d / (a + k·d). Higher means more attacks snuffed out.
const ContestStopScale = 0.75

// ContestFoulBase is the share of stops that are fouls for a defender of
// average aggression (ContestFoulAggressionPivot). It scales with
// (aggression / pivot)^1.5, capped at ContestFoulCap.
const (
	ContestFoulBase            = 0.16
	ContestFoulAggressionPivot = 65.0
	ContestFoulCap             = 0.45
)

// Of fouls: the share given away in the box (a penalty) and the share close
// enough for a shot at goal (a direct free kick). The rest restart play
// without a shot.
const (
	ContestPenaltyShareOfFouls          = 0.12
	ContestShootingFreeKickShareOfFouls = 0.35
)

// Pressing wins the ball back more often but gives away more fouls.
const (
	ContestHighPressStop = 1.08
	ContestLowPressStop  = 0.94
	ContestHighPressFoul = 1.30
	ContestLowPressFoul  = 0.85
)

// ContestOpenChanceWeights is the mix of chances an attack creates. Corners,
// free kicks and penalties never come from here: they come from blocks,
// saves and fouls.
var ContestOpenChanceWeights = map[string]uint{
	"Open Play":       8,
	"Cross":           5,
	"Long Range":      3,
	"Goalkeeper Shot": 2,
}

// A defending team's line height changes how often an attack ends one-on-one
// with the keeper: a high line leaves space in behind, a deep one doesn't.
const (
	ContestBreakawayVsHighLine = 1.6
	ContestBreakawayVsDeepLine = 0.6
)

// --- The shot ----------------------------------------------------------------

// ContestShot shapes one kind of shot:
//   - Block: the most a blocker can stop, scaled by the duel
//     P(blocked) = Block · b / (s + b)
//   - OnTarget: how hard it is to hit the target, P(on) = s / (s + OnTarget)
//   - Keeper: how much the keeper's score counts,
//     P(goal | on target) = s / (s + Keeper · k)
//
// s is the shooter's chance-type score (0-100 after the skill curve), b the
// blocker's defense score and k the keeper's.
type ContestShot struct {
	Block    float64
	OnTarget float64
	Keeper   float64
}

// ContestShots by chance type (keyed by the ChanceType string; the tuning
// package can't import the engine's types).
var ContestShots = map[string]ContestShot{
	"Open Play":       {Block: 0.35, OnTarget: 19, Keeper: 0.62},
	"Cross":           {Block: 0.30, OnTarget: 25, Keeper: 0.72},
	"Corner":          {Block: 0.30, OnTarget: 29, Keeper: 0.72},
	"Long Range":      {Block: 0.60, OnTarget: 42, Keeper: 1.10},
	"Free Kick":       {Block: 0.40, OnTarget: 31, Keeper: 0.82},
	"Penalty":         {Block: 0, OnTarget: 0.8, Keeper: 0.02},
	"Goalkeeper Shot": {Block: 0, OnTarget: 14, Keeper: 0.80},
}

// ContestFallbackKeeper is the keeper score used when a line-up has no
// goalkeeper: an outfielder in goal.
const ContestFallbackKeeper = 8.0

// After a block or a save the ball sometimes goes out for a corner.
const (
	ContestCornerAfterBlock = 0.40
	ContestCornerAfterSave  = 0.30
	// ContestMaxFollowUps stops a corner leading to a corner forever.
	ContestMaxFollowUps = 2
)

// --- Game state --------------------------------------------------------------

// ContestScoreEffect: per goal of lead, the leader's possession weight drops
// by this share and the trailer's rises by it (capped at
// ContestScoreEffectCap). Leaders sit deeper, trailers push.
const (
	ContestScoreEffect    = 0.10
	ContestScoreEffectCap = 0.35
)

// Once a side is ContestComfortableLead goals up it eases off: every goal of
// lead from there takes ContestEaseOff off its attacking play (carrying and
// shooting), down to at most ContestEaseOffCap. Without it a mismatch keeps
// scoring at full tilt and 7-0s are routine.
const (
	ContestComfortableLead = 3
	ContestEaseOff         = 0.22
	ContestEaseOffCap      = 0.60
)

// ForPosition is the weight for a position name ("Goalkeeper", "Defense",
// "Midfield", "Attack"); anything else weighs nothing.
func (w PositionWeights) ForPosition(pos string) float64 {
	switch pos {
	case "Goalkeeper":
		return w.Goalkeeper
	case "Defense":
		return w.Defense
	case "Midfield":
		return w.Midfield
	case "Attack":
		return w.Attack
	default:
		return 0
	}
}
