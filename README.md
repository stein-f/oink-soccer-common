# oink-soccer-common

The deterministic soccer match-simulation engine that powers Oink Soccer.

This repository contains two engines:

- **v1** (root of the module) — original engine, frozen for bug fixes only.
- **v2** (`v2/` subdirectory, separate Go module) — clean-room rebuild with a phase-based simulation, balanced formation profiles, specialist player attributes, and explicit manager tactics.

New work should target v2.

## Documentation

- **[Manager's Guide](docs/manager-guide.md)** — start here if you want to understand how to build a team. Covers attributes, chance types, tactics, formations, and worked examples.
- **[v1 → v2 differences](docs/v1-vs-v2.md)** — what changed between engines and why we rebuilt.
- **[Architecture](docs/architecture.md)** — technical deep-dive on the v2 simulation flow.
- **[Public API](docs/api.md)** — the locked public surface for consumers.
- **[Rebuild plan](docs/rebuild-plan.md)** — internal changelog and decision history.

## Using v2

```go
import (
    "math/rand"
    soccer "github.com/stein-f/oink-soccer-common/v2"
)

src := rand.New(rand.NewSource(seed))
events, injuries, err := soccer.RunGameWithSeed(src, homeLineup, awayLineup)
```

The full public API is documented in [`docs/api.md`](docs/api.md).

To re-derive `seed` from an Algorand block hash:

```go
import "github.com/stein-f/oink-soccer-common/v2/algorand"

client := algorand.NewClient(nil)
bs, err := client.FetchBlockSeed(ctx, round)
events, injuries, err := soccer.RunGameWithSeed(bs.Source, home, away)
```

## Engine model (v2 in one paragraph)

Each match runs in five phases: **tempo** (how many chances), **schedule** (when), **possession** (which team gets each chance), **resolve** (chance type + attacker + goal/miss), **injuries**. Every formation has a 5-axis trade-off profile (Possession / ChanceCreation / ChanceQuality / DefSolidity / InjuryRisk) — none strictly dominates another. Player attributes split physical ratings into Pace / Recovery / WorkRate, plus specialist attributes (Heading / Composure / Technique / Finishing / Tackling) that drive specific chance types — a target man with high Heading owns corners, a clutch finisher with high Composure owns penalties. Each `ChanceType` runs its own scoring formula referencing the right attributes. Optional `Tactics` (press, tempo, line height, set-piece taker) and `PlayerRole` (captain, target man, playmaker, ball winner) let managers shape outcomes; press and line height also shift which player attributes are most valuable. A skill curve `(rating/100)^4` amplifies player-quality differentials. The whole thing is a pure function of `(seed, lineups)`.

## Allocation

Season-start NFT allocation lives in `v2/allocation`:

```go
import "github.com/stein-f/oink-soccer-common/v2/allocation"

pool := allocation.NewPool(candidates, allocation.DefaultRules())
assignments, err := allocation.Allocate(rand, pool, assets)
```

Same `(seed, candidates, assets)` always produces the same allocation, so anyone can re-verify the result from just the Algorand block hash.

## Running the v1 examples

The v1 examples still work for historical reference:

```sh
go run cmd/simulate/main.go        # 10k-game distribution stats
go run cmd/verify/main.go          # re-verify a specific historical game
go run cmd/allocation/runner/main.go  # season allocation (CSV output)
```

## Testing

v1 and v2 are separate Go modules and test independently:

```sh
make test          # v1 (root)
cd v2 && make test # v2
```

v2 includes a balance harness that runs 5,000 trials per (home × away) formation pair to measure win-rate spread:

```sh
cd v2 && go test -v -run TestFormationBalance ./...
```

Set `RUN_BALANCE_STRICT=1` to fail the test when the spread exceeds the ±3% target (the engine currently sits at ~2.98% spread, which is under target; the gate stays in place so per-trial noise doesn't randomly trip CI).

## Snapshots

v2 commits 20 golden JSON snapshots at `v2/testdata/golden/v2/` as a regression guard. Regenerate with:

```sh
cd v2 && go run ./cmd/snapshot
```

A code change that diffs the snapshots is either intentional (regenerate + explain in the PR) or a regression (fix it). See `v2/testdata/golden/README.md`.
