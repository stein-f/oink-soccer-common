// Package soccer is the v2 rewrite of the oink-soccer match simulation engine.
//
// v2 is a clean-room rebuild that preserves the public surface used by the
// downstream lost-pigs project while replacing the internal model with a
// phase-based simulation, balanced formations, and explicit manager tactics.
//
// See docs/rebuild-plan.md at the repo root for the rebuild plan and progress.
//
// During the rebuild, RunGameWithSeed returns ErrNotImplemented. The package
// declares the full public surface so downstream consumers can compile-check
// against v2 incrementally.
package soccer
