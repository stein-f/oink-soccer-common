package soccer

import (
	"errors"
	"math/rand"
)

// ErrNoPenaltyTakers is returned by RunShootoutWithSeed when a lineup has no
// players to take penalties.
var ErrNoPenaltyTakers = errors.New("soccer: shootout requires players on both teams")

// PenaltyDirection is the side of the goal a penalty is aimed at. The set is
// dictated by the available artwork (left / mid / right), not by the physics.
type PenaltyDirection string

const (
	PenaltyDirectionLeft  PenaltyDirection = "Left"
	PenaltyDirectionMid   PenaltyDirection = "Mid"
	PenaltyDirectionRight PenaltyDirection = "Right"
)

// PenaltyResult is the binary outcome of a penalty. There is no distinct
// "saved" value: the artwork only distinguishes scored from missed, and a save
// and an off-target shot both map to Missed (commentary text may still differ).
type PenaltyResult string

const (
	PenaltyResultScored PenaltyResult = "Scored"
	PenaltyResultMissed PenaltyResult = "Missed"
)

// penaltyDirections pins iteration / index order for deterministic direction
// selection.
var penaltyDirections = []PenaltyDirection{
	PenaltyDirectionLeft,
	PenaltyDirectionMid,
	PenaltyDirectionRight,
}

// PenaltyOutcome is the result of a single penalty. It is intentionally
// self-contained and independent of any shootout so that a lone penalty (e.g. a
// future in-match spot kick) can flow through the same highlight pipeline
// unchanged. Team colour (home=red, away=blue) is derived by the consumer from
// TeamType — it is not baked in here.
type PenaltyOutcome struct {
	TakerID   string           `json:"taker_id"`
	KeeperID  string           `json:"keeper_id"`
	TeamType  TeamType         `json:"team_type"`
	Direction PenaltyDirection `json:"direction"`
	Result    PenaltyResult    `json:"result"`
}

// IsGoal reports whether the penalty was scored.
func (p PenaltyOutcome) IsGoal() bool {
	return p.Result == PenaltyResultScored
}

// TakePenaltyWithSeed resolves a single penalty between a taker and a keeper,
// deterministically from the supplied random source. It is attribute-driven and
// knows nothing about shootouts or cups, so it is reusable for any spot kick.
//
// Conversion mirrors the in-match penalty model: the taker's score pairs their
// AttackRating with Composure (the clutch-finisher attribute) and applies the
// penalty AttackBoost; the keeper's score is their goalkeeper save ability
// scaled down by the penalty DefenseScale. Goal probability is atk/(atk+def) —
// the same formula the match engine uses for open play — which yields the high
// conversion rates expected of penalties.
func TakePenaltyWithSeed(r *rand.Rand, taker, keeper SelectedPlayer, teamType TeamType) PenaltyOutcome {
	atk := playerAttackForChance(taker, ChanceTypePenalty) * chanceTypeAttackBoost(ChanceTypePenalty)
	def := playerDefense(keeper, Tactics{}) * chanceTypeDefenseScale(ChanceTypePenalty)

	if atk < 1 {
		atk = 1
	}
	if def < 1 {
		def = 1
	}

	scored := r.Float64() < atk/(atk+def)
	direction := penaltyDirections[r.Intn(len(penaltyDirections))]

	result := PenaltyResultMissed
	if scored {
		result = PenaltyResultScored
	}

	return PenaltyOutcome{
		TakerID:   taker.ID,
		KeeperID:  keeper.ID,
		TeamType:  teamType,
		Direction: direction,
		Result:    result,
	}
}

// ShootoutResult is the outcome of a penalty shootout. Kicks is the full ordered
// stream of penalties (home, away, home, away, …) so consumers can replay it as
// highlights.
type ShootoutResult struct {
	HomeScore int              `json:"home_score"`
	AwayScore int              `json:"away_score"`
	Winner    TeamType         `json:"winner"`
	Kicks     []PenaltyOutcome `json:"kicks"`
}

const (
	// shootoutRegulationKicks is the best-of-5 phase length: each team takes up
	// to five kicks. The phase ends the instant one team's lead is unassailable —
	// i.e. exceeds what the other can still reach with its remaining kicks — so a
	// decided shootout doesn't pointlessly play out the remaining penalties.
	shootoutRegulationKicks = 5
	// shootoutMaxSuddenDeathRounds caps sudden death so a pathological seed can't
	// loop forever; if reached while still level the final kick's rand breaks it.
	shootoutMaxSuddenDeathRounds = 100
)

// RunShootoutWithSeed runs a penalty shootout between two lineups,
// deterministically from the supplied random source. It is a thin loop over
// TakePenaltyWithSeed: each team's players take in turn (every player takes a
// penalty), best-of-5 regulation followed by sudden death until one team leads
// after an equal number of kicks.
func RunShootoutWithSeed(r *rand.Rand, home, away GameLineup) (ShootoutResult, error) {
	if r == nil {
		return ShootoutResult{}, ErrNilRandSource
	}

	homeTakers := penaltyTakerOrder(home)
	awayTakers := penaltyTakerOrder(away)
	if len(homeTakers) == 0 || len(awayTakers) == 0 {
		return ShootoutResult{}, ErrNoPenaltyTakers
	}

	homeKeeper := findShootoutKeeper(home)
	awayKeeper := findShootoutKeeper(away)

	var result ShootoutResult

	kick := func(takers []SelectedPlayer, keeper SelectedPlayer, team TeamType, index int) {
		taker := takers[index%len(takers)]
		outcome := TakePenaltyWithSeed(r, taker, keeper, team)
		result.Kicks = append(result.Kicks, outcome)
		if outcome.IsGoal() {
			if team == TeamTypeHome {
				result.HomeScore++
			} else {
				result.AwayScore++
			}
		}
	}

	// Regulation: up to five kicks each, alternating home then away, ending the
	// instant one team is mathematically safe — its lead exceeds what the other
	// can still reach with the kicks it has left. This is the standard rule: a
	// shootout that's already decided (e.g. 4-0 after seven kicks) stops there
	// rather than playing out penalties that cannot change the result.
	homeTaken, awayTaken := 0, 0
	decided := func() bool {
		homeRemaining := shootoutRegulationKicks - homeTaken
		awayRemaining := shootoutRegulationKicks - awayTaken
		return result.HomeScore > result.AwayScore+awayRemaining ||
			result.AwayScore > result.HomeScore+homeRemaining
	}
	for i := 0; i < shootoutRegulationKicks; i++ {
		kick(homeTakers, awayKeeper, TeamTypeHome, i)
		homeTaken++
		if decided() {
			break
		}
		kick(awayTakers, homeKeeper, TeamTypeAway, i)
		awayTaken++
		if decided() {
			break
		}
	}

	// Sudden death: paired kicks (both teams take, then compare) until decided.
	for round := 0; result.HomeScore == result.AwayScore && round < shootoutMaxSuddenDeathRounds; round++ {
		index := shootoutRegulationKicks + round
		kick(homeTakers, awayKeeper, TeamTypeHome, index)
		kick(awayTakers, homeKeeper, TeamTypeAway, index)
	}

	// Crown the winner. If still level after the sudden-death cap (effectively
	// impossible with a real source), break the tie from the same source.
	switch {
	case result.HomeScore > result.AwayScore:
		result.Winner = TeamTypeHome
	case result.AwayScore > result.HomeScore:
		result.Winner = TeamTypeAway
	default:
		if r.Intn(2) == 0 {
			result.Winner = TeamTypeHome
		} else {
			result.Winner = TeamTypeAway
		}
	}

	return result, nil
}

// penaltyTakerOrder returns the order in which a lineup's players take penalties:
// outfield players first (in lineup order), then any goalkeeper last — the
// conventional ordering, though every player takes a kick.
func penaltyTakerOrder(lineup GameLineup) []SelectedPlayer {
	outfield := make([]SelectedPlayer, 0, len(lineup.Players))
	var keepers []SelectedPlayer
	for _, p := range lineup.Players {
		if p.SelectedPosition == PlayerPositionGoalkeeper || isGoalkeeper(p.Attributes) {
			keepers = append(keepers, p)
			continue
		}
		outfield = append(outfield, p)
	}
	return append(outfield, keepers...)
}

// findShootoutKeeper returns the player who keeps goal during the shootout: the
// goalkeeper if present, otherwise the first player as a fallback.
func findShootoutKeeper(lineup GameLineup) SelectedPlayer {
	for _, p := range lineup.Players {
		if p.SelectedPosition == PlayerPositionGoalkeeper || isGoalkeeper(p.Attributes) {
			return p
		}
	}
	return lineup.Players[0]
}
