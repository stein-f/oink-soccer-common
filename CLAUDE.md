# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repo layout

This repo holds two engines:

- **v1** at the repo root — frozen, bug-fix only.
- **v2** in `v2/` — clean-room rebuild, separate Go module (`github.com/stein-f/oink-soccer-common/v2`). All new work goes here.

When working in this repo, default to v2 unless the user explicitly asks about v1. The two test suites are independent; cd into `v2/` to work on v2.

## Commands

```sh
# v1 (root)
make test

# v2
cd v2 && make test
cd v2 && make snapshot-v1            # regenerate v1-baseline snapshots (informational)
cd v2 && go run ./cmd/snapshot       # regenerate v2 golden snapshots (load-bearing)
cd v2 && go test -v -run TestFormationBalance ./...   # balance harness (logs win-rate spread)
cd v2 && RUN_BALANCE_STRICT=1 go test ./...           # fail if balance spread > 3%

# calibrating a model against real line-ups (see v2/cmd/calibrate)
cd v2 && go run ./cmd/calibrate -archives <dir of match archives> -model steps [-enrich ../cmd/allocation/fifa_players_22.csv]
cd v2 && go run ./cmd/calibrate -model steps -attributes 4000     # what +5 of each attribute is worth (-attribute-delta 12 for a clearer read)
cd v2 && go run ./cmd/calibrate -model steps -tactics 4000        # is each tactic a real choice: its worth by kind of squad and opponent
cd v2 && go run ./cmd/calibrate -model steps -levers 12000        # what boosts, tactics and roles are worth
cd v2 && go run ./cmd/calibrate -model steps -formations <dir of team files> \
    [-enrich <dataset.csv>] [-deal ../cmd/allocation/s18/eligible_assets.csv -deal-dataset ../cmd/allocation/fifa_players_fc27.csv] \
    [-lineups cache.json]   # cache the slow line-up search while tuning constants
cd v2 && go test -bench RunGame -run xxx .            # cost per match, by model
```

Formation balance is judged with `-formations` on real squads (and on squads dealt afresh with `-deal`), not with `TestFormationBalance`: the identical-squad test passed on the classic and contest models while the Box was the worst choice for every real team.

## Where to start

- `docs/rebuild-plan.md` — phase-by-phase plan + progress + decision log. **Read this first.**
- `docs/architecture.md` — v2 simulation flow, attribute model, formation profiles, chance types, tactics.
- `docs/api.md` — v2 public surface contract.
- `docs/rebuild.md` — original rebuild brief (do not edit).

## v2 architecture (one screen)

```
v2/
├── engine.go        RunGameWithSeed (public entry point)
├── match.go         simulateMatch — 5 phases: tempo, schedule, possess, resolve, injuries
├── scoring.go       per-player + per-team scoring (unexported)
├── chance.go        ChanceType profiles + attacker selection
├── injuries.go      injury catalogue + roll
├── tactics.go       Tactics, PlayerRole, multipliers
├── formation.go     FormationConfig + Profile (5-axis trade-off)
├── algorand/        block-hash → *rand.Rand
├── allocation/      season NFT allocation (separate, deterministic)
├── internal/tuning/ every magic number, named + documented
└── testdata/golden/ v1-baseline (informational) + v2 (load-bearing) snapshots
```

The engine is a pure function of `(rand, home, away)`. No `time.Now()`, no globals, no I/O. Determinism is the contract that lets matches be re-verified from an Algorand block hash.

## Downstream consumer

`lost-pigs` at `/Users/steinfletcher/code/ooyc/lost-pigs` consumes this engine. The locked v2 public surface is documented in `docs/api.md` — anything outside that list is implementation detail.

## Open work

- Player dataset: seasons are pinned via `fifacsv.DatasetPath` (FIFA 22 ≤ s18, FC 27 from s19 via `cmd/convert_fc27`); `FC27FirstSeason` must match lost-pigs `domain.FC27FirstSeason`.
- `v2.0.0` tag: balance now passes ±3% strict, so the tag is unblocked.

## Conventions

- Tests live alongside code (`foo.go` + `foo_test.go`).
- Internal scoring helpers are unexported; tests live in `package soccer` (no `_test` suffix) so they can drive them.
- Map iteration is always wrapped in a sorted-key list. Map iteration order is randomised in Go and would break determinism.
- `gofmt -s -w .` before committing — `make test` will fail on unformatted code.
