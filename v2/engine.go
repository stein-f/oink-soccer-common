package soccer

import "math/rand"

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
