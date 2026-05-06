# v2 Engine Architecture

This document describes how the v2 simulation works end-to-end. For the public API contract, see `docs/api.md`. For the rebuild history, see `docs/rebuild-plan.md`.

## Package layout

```
v2/
├── doc.go              package overview
├── engine.go           RunGameWithSeed (public entry point)
├── errors.go           ErrNilRandSource
├── enums.go            TeamType, PlayerPosition, FormationType, ChanceType, …
├── player.go           PlayerAttributes, SelectedPlayer, Effective* accessors
├── team.go             Team, GameLineup
├── tactics.go          Tactics, PlayerRole, lever multipliers
├── boost.go            Boost, DRDecayPerApplication
├── formation.go        FormationConfig + Profile
├── injuries.go         Injury catalogue + roll logic
├── chance.go           ChanceType profiles, attacker selection
├── scoring.go          per-player + per-team scoring helpers (unexported)
├── match.go            simulateMatch (the engine itself)
├── algorand/           Algorand block-hash → *rand.Rand
├── allocation/         player-to-NFT allocation (separate, deterministic)
├── internal/tuning/    every magic number in one place
├── testdata/
│   ├── fixtures.go     StrongTeam, WeakTeam (mirror v1 ratings)
│   └── golden/
│       ├── v1-baseline/   v1 outputs for reference (informational)
│       └── v2/            v2 regression snapshots (load-bearing)
└── cmd/snapshot/       v2 snapshot regenerator
```

The root `soccer` package contains everything the downstream consumer (`lost-pigs`) needs. `internal/tuning` is private — change a value there and the engine recompiles cleanly.

## Match simulation flow

```
RunGameWithSeed(rand, home, away)
    ↓
simulateMatch
    ├─ 1. Tempo
    │     decideMatchTempo(rand, homeF, awayF, tempoFactor) → totalChances
    │     (uses tuning.FormationChanceRanges + Tactics.Tempo)
    │
    ├─ 2. Schedule
    │     scheduleMinutes(rand, totalChances) → []int (sorted, late-weighted)
    │     (uses tuning.EventMinuteBuckets)
    │
    ├─ 3. Score teams (once)
    │     teamControl(home) × Possession × CaptainBoost × TeamBoost
    │       × OpponentPress × OpponentLineHeight
    │     teamDefense(home) × DefSolidity × CaptainBoost × DefenseBias
    │       × OwnLineHeight
    │
    ├─ 4. For each chance (i = 0 .. totalChances-1):
    │       attacker = pickAttackingTeam(rand, homeControl, awayControl)
    │       chanceType = pickChanceType(rand, prevType)   // no consecutive duplicates
    │       attackerPlayer = pickAttackerWithTactics(rand, attackingLineup, chanceType, tactics)
    │       event = resolveChance(rand, ..., minute[i])
    │           atk = playerAttack(p) × ChanceCreation × ChanceQuality
    │                                  × ChanceTypeAttackBoost
    │                                  × TempoQualityFactor
    │           def = defendingDefense × ChanceTypeDefenseScale
    │           p   = atk / (atk + def)
    │           goal? rand.Float64() < p
    │
    └─ 5. Injuries
          rollInjuries(home, awayAggression, awayInjuryRisk × homePressInjury, ...)
          rollInjuries(away, homeAggression, homeInjuryRisk × awayPressInjury, ...)
```

Every randomness draw uses the supplied `*rand.Rand`. There are no clocks, no globals, no I/O. The function is a pure function of `(rand, home, away)`.

## Attribute model

> Per-attribute reference with every formula and tactic interaction: [attributes.md](attributes.md). The section below is the high-level rationale.

v1 folded `SpeedRating` into all three of control, attack, and defense — silently triple-counting it. v2 originally split the physical dimension into three orthogonal attributes (Pace / Recovery / WorkRate); Pace and Recovery were later consolidated back into a single `SpeedRating` for simpler UX, leaving `WorkRate` as the only separate physical (it drives midfield control under the Press tactic).

### Physicals

```
control_default  = (controlRating * 4 + workRate)    / 5
defense_outfield = (defenseRating * 5 + tackling*2 + speedRating) / 8
defense_keeper   = (goalkeeperRating * 5 + speedRating) / 6
attack_default   = (attackRating * 3 + speedRating)  / 4
```

`WorkRate` is the only optional physical field on `PlayerAttributes`. When zero, `EffectiveWorkRate()` falls back to `SpeedRating` so legacy rosters work unchanged.

### Specialist attributes (chance-type-specific)

Each chance type's attacker score uses a dedicated formula that reads a specialist attribute. All specialists are optional with sensible backfills (see `EffectiveFinishing` / `EffectiveHeading` / `EffectiveTechnique` / `EffectiveComposure` / `EffectiveTackling`).

| Chance type | Attacker score formula |
|-------------|------------------------|
| OpenPlay | `(atk*2 + finishing + speed) / 4` |
| Cross | `(atk*2 + heading*2 + speed) / 5` |
| Corner | `(atk*2 + heading*3) / 5` |
| LongRange | `(atk*2 + technique*3) / 5` |
| FreeKick | `(atk + technique*3) / 4` |
| Penalty | `(atk*2 + composure*3) / 5` |
| GoalkeeperShot | `(atk + finishing + speed*3) / 5` |

Backfills: `Heading→AttackRating`, `Finishing→AttackRating`, `Technique→ControlRating`, `Composure→ControlRating`, `Tackling→DefenseRating`.

### Skill curve

Every per-player score (`playerControl`, `playerAttack`, `playerDefense`) is then run through `tuning.SkillCurve(raw) = (raw/100)^4 × 100` before team aggregation. This amplifies the gap between elite and average players — without the curve, an 87-rated team beats a 78-rated team only 47% of the time; with it, ~70%. Reference points: 70 → 24, 80 → 41, 87 → 57, 90 → 66, 100 → 100.

### Tactic-driven attribute weighting

The `default` formulas above apply when `Tactics` is the zero value. Press shifts the control formula; line height shifts the outfield defense formula:

| Tactic | Control formula | Outfield defense formula |
|--------|----------------|--------------------------|
| `Press: low` | `(ctrl*5 + workRate) / 6` | (unaffected) |
| `Press: medium / none` | `(ctrl*4 + workRate) / 5` (legacy) | (unaffected) |
| `Press: high` | `(ctrl*3 + workRate*2) / 5` | (unaffected) |
| `LineHeight: deep` | (unaffected) | `(def*6 + tackling*2 + speed*0) / 8` |
| `LineHeight: normal / none` | (unaffected) | `(def*5 + tackling*2 + speed) / 8` (legacy) |
| `LineHeight: high` | (unaffected) | `(def*3 + tackling*2 + speed*3) / 8` |

Goalkeeper defense is tactic-invariant.

### Per-player adjustments

After the curve, every player score has these modifiers applied:

- **Out of position**: ×0.85 if `SelectedPosition` isn't in the player's `Positions` (or `PrimaryPosition`).
- **Injury**: × the injury's `StatsReduction`.

### Role bonuses

- **Playmaker** → focal-point weighting in `teamControl` aggregation. Within their position group's mean, the Playmaker contributes with `tuning.PlaymakerControlWeight` (= 2.0) vs everyone else's 1.0. Drags the group mean toward the Playmaker's score — boosts team control if they're the strongest controller, lowers it if they're weak. Not a per-player ×1.10 (a previous implementation, since corrected because it gave even poor players a free lift).
- **BallWinner** → focal-point weighting in `teamDefense` aggregation, mirror of Playmaker. `tuning.BallWinnerDefenseWeight` (= 2.0) drags the position group's defense mean toward the Ball Winner's score. Same correction as Playmaker; not a per-player ×1.10.
- **TargetMan** → ×2.0 selection weight on corners + crosses (selection only, not score)
- **Captain** → quality-scaled. Quality `q = (primarySkill + EffectiveComposure) / 2` (primarySkill = `GoalkeeperRating` for keepers, `ControlRating` otherwise). Two small effects, both via `tuning`: (1) `CaptainTeamBoost(q)` is the team-wide multiplier on control + defense, ≈ 1 + (q-60)/100 × 0.06 (range ≈ [0.964, 1.024]); (2) `CaptainSelfBoost(q)` is the per-action multiplier on the captain's own scores, applied in `adjustForState`. Replaced an earlier flat ×1.03 that was identity-agnostic.

## Formation profiles

Every formation has a 5-axis trade-off profile (`internal/tuning.FormationProfile`). These are the *tuned* values that produce a < 3% home-win-rate spread in the balance harness:

| Formation | Possession | ChanceCreation | ChanceQuality | DefSolidity | InjuryRisk |
|-----------|-----------|----------------|---------------|-------------|------------|
| Pyramid   | 1.00 | 1.00 | 1.03 | 1.02 | 1.00 |
| Diamond   | 1.03 | 1.00 | 1.00 | 1.00 | 1.00 |
| Y         | 1.00 | 1.03 | 1.02 | 0.97 | 1.00 |
| Box       | 1.00 | 1.05 | 1.05 | 0.96 | 1.05 |

A unit test (`TestNoFormationStrictlyDominates`) pins the invariant that no formation beats another on every axis. The legacy `FormationConfig.DefenseModifier` / `ControlModifier` / `AttackModifier` fields are derived from `Profile` for backwards compatibility with the lost-pigs frontend.

`tuning.FormationChanceRanges` further controls the chance volume per formation-style pairing (ATT/BAL/DEF) — the floors were lifted during balance tuning so defensive matchups don't get starved of chances.

## Chance types

Every event carries a `ChanceType`. v1 emitted these but never used them; v2 makes each type behave differently along three dimensions: who's likely to take the shot, what attributes drive the shot's quality, and how easy the chance is to convert.

| Type | AttackBoost | DefenseScale | Position bias | Score formula |
|------|-------------|--------------|---------------|---------------|
| OpenPlay | 1.00 | 1.00 | default (ATK 70 / MID 20 / DEF 10 / GK 2) | `(atk*2 + finishing + speed) / 4` |
| Cross | 0.95 | 1.05 | ATK heavy | `(atk*2 + heading*2 + speed) / 5` |
| Corner | 0.90 | 1.10 | ATK + MID + some DEF (defenders go up) | `(atk*2 + heading*3) / 5` |
| LongRange | 0.70 | 1.20 | MID heavy | `(atk*2 + technique*3) / 5` |
| FreeKick | 0.85 | 1.10 | MID + ATK split | `(atk + technique*3) / 4` |
| Penalty | 1.50 | 0.50 | ATK heavy | `(atk*2 + composure*3) / 5` |
| GoalkeeperShot | 1.20 | 0.70 | ATK only (1-on-1 / breakaway) | `(atk + finishing + speed*3) / 5` |

`pickChanceType` bans the immediately-previous type so commentary doesn't repeat ("CORNER. CORNER. CORNER.").

## Tactics

Each lever shifts a specific multiplier. Defaults (zero value) mean "no effect".

| Lever | Multiplier effect | Attribute weighting effect |
|-------|-------------------|----------------------------|
| `Press: low/medium/high` | Opponent control × 1.02 / 0.98 / 0.94. Own injury risk × 0.95 / 1.0 / 1.10. **High press also triggers in-match fatigue** — own attack quality × 0.90 in 60-74min, × 0.82 in 75+min. Without this, the only cost of high press would be the next-game injury bump (mitigatable with recovery items), making it a free lunch. | Shifts control formula's skill ↔ workrate balance (see Attribute model). |
| `Tempo: slow/normal/fast` | Total chances × 0.92 / 1.0 / 1.10. Own chance quality × 1.05 / 1.0 / 0.96 (faster = rushed). | (no attribute shift) |
| `LineHeight: deep/normal/high` | Opponent control × 1.03 / 1.0 / 0.97 (deep cedes the midfield; high compresses the pitch). Own defense × 1.05 / 1.0 / 0.96 (deep is compact; high is brittle to balls in behind). | Shifts outfield defense formula's positioning ↔ speed balance. |
| `SetPieceTaker: PlayerID` | Named player takes every Free Kick + Penalty (taker = scorer). On Corners the named player *delivers* — they're excluded from the finisher pool, and their `Technique` scales the chance's AttackBoost via `tuning.CornerDeliveryFactor` (≈ ×0.84 at technique=20, ×1.16 at technique=100). The corner finisher is still picked normally by Heading + position. | (no attribute shift) |

## Determinism contract

The engine's only inputs are `(rand, home, away)`. The contract:

1. The engine never calls `time.Now()`, never reads env vars, never opens files.
2. Iteration over maps is always wrapped in a sorted key list (see `chance.go`, `allocation.go`).
3. `InjuryEvent.Expires` is left as zero. Callers attach their own clock with `ResolveInjuryExpiry`.
4. The engine accepts `*rand.Rand` — the standard library's deterministic source. v2 dropped v1's `RunGame` no-seed variant entirely.

This is the contract that lets a match outcome be re-verified later from just the Algorand round number.

## Allocation

`v2/allocation` is a separate, smaller engine that runs once per season:

```
NewPool(candidates, rules) → *Pool         // index by (position, level) + aggressive bucket
Allocate(rand, pool, assets) → []Assignment // pure function
```

Same determinism contract: same `(seed, candidates, rules, assets)` ⇒ same output. The Algorand block-hash → `*rand.Rand` derivation lives in `v2/algorand` so it's reusable between match seeding and allocation seeding.
