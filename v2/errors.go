package soccer

import "errors"

// ErrNilRandSource is returned by RunGameWithSeed when called with a nil
// *rand.Rand. v2 makes the random source mandatory because seeding is the
// determinism contract; callers must build their own source explicitly so
// the seed is visible in their code.
var ErrNilRandSource = errors.New("soccer: rand source is required")

// ErrUnknownModel is returned by RunGameWithModel for a model it doesn't know.
var ErrUnknownModel = errors.New("soccer: unknown engine model")
