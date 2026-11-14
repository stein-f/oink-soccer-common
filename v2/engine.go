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
