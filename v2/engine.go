package soccer

import (
	"fmt"
	"math/rand"
)

// RunGameWithSeed simulates a deterministic match between two lineups using
// the supplied random source. The same (seed, home, away) inputs always
// produce the same output — the function uses no clocks, no globals, and
// no I/O.
//
// The returned event slice is in chronological order by minute. Each event
// carries the new ChanceType field (introduced in v2) so consumers can
// render richer commentary.
//
// Injuries are returned with their DurationDays populated; the absolute
// expiry timestamp is left as the zero time so the engine itself stays
// deterministic. Callers should attach a clock with ResolveInjuryExpiry.
func RunGameWithSeed(r *rand.Rand, home GameLineup, away GameLineup) ([]GameEvent, Injuries, error) {
	outcome, err := RunGameWithSeedDetailed(r, home, away)
	return outcome.Events, outcome.Injuries, err
}

// MatchOutcome is a full run: the events and injuries RunGameWithSeed
// returns, plus every item boost as it was rolled for the match.
type MatchOutcome struct {
	Events   []GameEvent `json:"events"`
	Injuries Injuries    `json:"injuries"`
	Boosts   MatchBoosts `json:"boosts"`
	// Totals counts every contest of the match, including the attacks that
	// never became an event. Only ModelSteps fills it.
	Totals *MatchTotals `json:"totals,omitempty"`
}

// MatchTotals is each side's count of what it did over the match.
type MatchTotals struct {
	Home TeamTotals `json:"home"`
	Away TeamTotals `json:"away"`
	// HomePlayers and AwayPlayers are the same counts for each player who
	// started, by player ID, with his goals and assists. They add up to the
	// side's totals, except for what no player did: a possession is the
	// side's, and so is a save made with nobody in goal.
	HomePlayers map[string]PlayerTotals `json:"home_players,omitempty"`
	AwayPlayers map[string]PlayerTotals `json:"away_players,omitempty"`
}

// PlayerTotals counts one player's part in a match, attacks that never
// became an event included. A tackle is an attack he ended himself: a take-on
// stopped or a runner caught. An aerial won is a header won at either end.
type PlayerTotals struct {
	Goals            int `json:"goals,omitempty"`
	Assists          int `json:"assists,omitempty"`
	Shots            int `json:"shots,omitempty"`
	ShotsOnTarget    int `json:"shots_on_target,omitempty"`
	Woodwork         int `json:"woodwork,omitempty"`
	Passes           int `json:"passes,omitempty"`
	PassesCompleted  int `json:"passes_completed,omitempty"`
	TakeOns          int `json:"take_ons,omitempty"`
	TakeOnsWon       int `json:"take_ons_won,omitempty"`
	Crosses          int `json:"crosses,omitempty"`
	CrossesCompleted int `json:"crosses_completed,omitempty"`
	AerialsWon       int `json:"aerials_won,omitempty"`
	Tackles          int `json:"tackles,omitempty"`
	Interceptions    int `json:"interceptions,omitempty"`
	Clearances       int `json:"clearances,omitempty"`
	Blocks           int `json:"blocks,omitempty"`
	Saves            int `json:"saves,omitempty"`
	Fouls            int `json:"fouls,omitempty"`
}

// TeamTotals counts one side's contests. The attacking counts (passes,
// take-ons, crosses, shots) are what the side tried going forward; the
// defending ones (tackles, interceptions, blocks, saves, fouls) are what it
// did without the ball.
type TeamTotals struct {
	Possessions      int `json:"possessions"`
	Passes           int `json:"passes"`
	PassesCompleted  int `json:"passes_completed"`
	TakeOns          int `json:"take_ons"`
	TakeOnsWon       int `json:"take_ons_won"`
	Crosses          int `json:"crosses"`
	CrossesCompleted int `json:"crosses_completed"`
	AerialsWon       int `json:"aerials_won"`
	Shots            int `json:"shots"`
	ShotsOnTarget    int `json:"shots_on_target"`
	Woodwork         int `json:"woodwork"`
	Tackles          int `json:"tackles"`
	Interceptions    int `json:"interceptions"`
	Clearances       int `json:"clearances"`
	Blocks           int `json:"blocks"`
	Saves            int `json:"saves"`
	Fouls            int `json:"fouls"`
}

// Model selects how chances are resolved. Results are only reproducible from
// a seed with the same model, so a season is played on one model throughout.
type Model string

const (
	// ModelClassic is the original model (seasons 1-18): a handful of shots,
	// each decided by one roll of attack against the whole defending team.
	ModelClassic Model = ""
	// ModelContest resolves possessions as a chain of contests between
	// players (build-up, shot, keeper), with set pieces caused by fouls,
	// blocks and saves. See docs/realism-review.md.
	ModelContest Model = "contest"
	// ModelSteps (season 19+) resolves each attack as a chain of steps
	// between specific players, chosen by where they stand: a pass against
	// the man trying to cut it out, a winger against the defender on his
	// side, a header against the centre-back. The steps are reported on each
	// event. See steps.go.
	ModelSteps Model = "steps"
)

// RunGameWithModel plays a match under the given model. ModelClassic is
// exactly RunGameWithSeedDetailed.
func RunGameWithModel(r *rand.Rand, home, away GameLineup, m Model) (MatchOutcome, error) {
	if r == nil {
		return MatchOutcome{}, ErrNilRandSource
	}
	switch m {
	case ModelClassic:
		return RunGameWithSeedDetailed(r, home, away)
	case ModelContest:
		events, injuries, boosts := simulateContestMatch(r, home, away)
		return MatchOutcome{Events: events, Injuries: injuries, Boosts: boosts}, nil
	case ModelSteps:
		return simulateStepsMatch(r, home, away), nil
	default:
		return MatchOutcome{}, fmt.Errorf("%w: %q", ErrUnknownModel, m)
	}
}

// RunGameWithSeedDetailed is RunGameWithSeed with the rolled boosts reported.
// It consumes the random source identically, so both entry points play the
// same match for the same inputs.
func RunGameWithSeedDetailed(r *rand.Rand, home GameLineup, away GameLineup) (MatchOutcome, error) {
	if r == nil {
		return MatchOutcome{}, ErrNilRandSource
	}
	events, injuries, boosts := simulateMatch(r, home, away)
	return MatchOutcome{Events: events, Injuries: injuries, Boosts: boosts}, nil
}
