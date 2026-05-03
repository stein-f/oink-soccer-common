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
	if r == nil {
		return nil, Injuries{}, ErrNilRandSource
	}
	events, injuries := simulateMatch(r, home, away)
	return events, injuries, nil
}
