# v2 Engine Architecture

This document describes how the v2 simulation works end-to-end. For the public API contract, see `docs/api.md`.

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

Every per-player score (`playerControl`, `playerAttack`, `playerDefense`) is then run through `tuning.SkillCurve(raw) = (raw/100)^6 × 100` before team aggregation. This amplifies the gap between elite and average players — at k=4.0 an 88-vs-83 matchup produced only ~57% / 24% home/away; k=6.0 sharpens it to ~65% / 18%, and 6.0 is the deliberate ceiling (above it press inverts — see `tuning.SkillCurveExponent`). Reference points: 70 → 12, 80 → 26, 87 → 43, 90 → 53, 100 → 100.

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
| `Press: low/medium/high` | Opponent control × 1.02 / 1.0 / 0.94 (medium is the explicit neutral, identical to unset). Own injury risk × 0.95 / 1.0 / 1.10. **High press also triggers in-match fatigue** — own attack quality × 0.90 in 60-74min, × 0.82 in 75+min. Without this, the only cost of high press would be the next-game injury bump (mitigatable with recovery items), making it a free lunch. | Shifts control formula's skill ↔ workrate balance (see Attribute model). |
| `Tempo: slow/normal/fast` | Total chances × 0.92 / 1.0 / 1.10. Own chance quality × 1.05 / 1.0 / 0.96 (faster = rushed). | (no attribute shift) |
| `LineHeight: deep/normal/high` | Opponent control × 1.03 / 1.0 / 0.97 (deep cedes the midfield; high compresses the pitch). Own defense × 1.05 / 1.0 / 0.96 (deep is compact; high is brittle to balls in behind). | Shifts outfield defense formula's positioning ↔ speed balance. |
| `SetPieceTaker: PlayerID` | Named player takes every Free Kick + Penalty (taker = scorer). On Corners the named player *delivers* — they're excluded from the finisher pool, and their `Technique` scales the chance's AttackBoost via `tuning.CornerDeliveryFactor` (≈ ×0.84 at technique=20, ×1.16 at technique=100). The corner finisher is still picked normally by Heading + position. | (no attribute shift) |

## Contest model (season 19+)

`RunGameWithModel(..., ModelContest)` runs `simulateContestMatch` (`contest.go`) instead of `simulateMatch`. The classic flow above is untouched. Per match:

```
possessions = uniform(ContestPossessions) × tempo          (same for every formation)
for each possession, in minute order:
    team    ← possession weights^0.5, with score effects (trailers see more of the ball)
              and home advantage (ContestHomePossession)
    build-up: carrier (control; Playmaker more often) vs stopper (defense; Ball Winner more often)
              stopped → maybe a foul (stopper's aggression, press) → free kick or penalty
              not stopped → open chance: Open Play / Cross / Long Range / one-on-one
                            (more one-on-ones against a high line)
    shot:     shooter (chance-type formula, as classic) vs blocker → blocked (maybe a corner)
              → on target? (difficulty by chance type)
              → keeper (GoalkeeperRating) → saved (maybe a corner) or goal
```

Every duel is `a^e / (a^e + b^e)` with `e = ContestDuelExponent` (0.45): a match now passes through several contests, and at full skill-curve strength they compound into certainties. Leaders ease off from three goals up (`ContestEaseOff`). All constants live in `internal/tuning/contest.go`; `cmd/calibrate` replays real line-ups to tune them, and `TestContest_FormationBalance` keeps formations within 3%.

Only shots are emitted (Goal / Miss with ChanceType), exactly like the classic model. Stops, fouls, blocks and saves happen but aren't reported yet.

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

## Step model (season 19+)

`RunGameWithModel(..., ModelSteps)` runs `simulateStepsMatch` (`steps.go`). The classic and contest flows are untouched. An attack is a chain of contests between two named players:

```
possessions = uniform(StepsPossessions) × tempo          (same for every formation)
who attacks = each side's quality, home advantage         (see "the stronger side" below)
route       = take-on | solo run | run in behind | cross | long ball | long shot, one option per
              player who could make it, weighted by what suits him and whether it would work
1. pass     passer (midfield, or defence if there is none) v the opponent's midfield
            (a solo run has no pass: the runner beats a midfielder instead; a long ball has
            none either: it goes over the midfield and is settled where it drops)
2. route    forward v the defender facing him, then v the defender covering if there is one;
            a cross is then a header v the marker; a long ball is the forward v the defender
            on him, then the runner on to his flick-on v the cover
3. shot     shooter v a blocker, then on target or not (off the woodwork it can come back out)
4. save     shot v the keeper: goal, held or parried (which can give a corner or a rebound)
```

- **Who does what follows the players.** `routes` lists every (kind of attack, player) option. Within a line a kind of attack goes to the player it suits, by `(score / StepsTypicalScore)^StepsAptitude × P(works)^StepsChoiceByOdds`: a tricky winger takes people on, a big man gets the crosses, and a slow forward only runs at a slower defender. How often the side does that kind of thing at all follows the same two (`StepsAptitudeMix`, `StepsOddsMix`), so a side leans to what its players are good at and what this opponent is weak to. The options depend only on who is on the pitch, so they are worked out once a match.
- **The long ball.** A defender hits it over the midfield for a forward to win against the man on him (`holdUpScore`: strength, then heading; the ball itself counts through the passer's long passing). Nobody intercepts it, so `StepsHighBallScale` makes the defender the favourite. Won, it is flicked on for a strike partner (a midfielder, for a lone forward) who has only the covering defender to beat, and a side with one defender has a midfielder drop in (`StepsTrackBackScale`); sometimes it is knocked down for a midfielder's shot (`StepsKnockDown`). Lost, it is a `Stop` with detail `header`. A side with no midfield goes long twice as often as one with (`StepsLongBallWithMidfield`), a Target Man is the one aimed at, and going long is the way over a high press (`StepsLongBallVsHighPress`).
- **No route simply pays more.** The scales are set so that a typical side scores about as often from a take-on, a run in behind, a cross or a long ball (about 0.11 goals an attempt; a long shot and a solo run pay less and are rarer). A through ball is the hardest pass (`StepsThroughBall`) and the race is the defender's to lose (`StepsChaseScale`). When this drifts, the attributes behind the better route become the only ones worth having: the check is the attribute report.
- **Tackling and strength.** A tackle is tackling first, then pace and strength; the defender covering a run in behind makes a tackle too, not a second race. A block is marking and tackling (`blockScore`). A poor tackler who wins the ball gives away a foul more often (`StepsCleanTackle`). Strength is half of every header and two thirds of a long ball.
- **Moments.** A *solo run* is a route of its own: past the midfielder who meets him, then the defence, and he always finishes it himself (`soloScore`: dribbling, pace, strength). After a take-on the dribbler squares it only if a team-mate is the better finisher (`StepsCutBack`, scaled by the two finishers). A *rebound* off a parry or the woodwork falls to whoever reads it (`poachScore`: positioning, finishing), nearly always a forward, except from a corner when everyone is up. A side one goal down from `StepsKeeperUpFromMinute` sends its *keeper* up for corners. A goal from distance is labelled a *worldie* by the scorer's shot power (`worldieChance`); the label changes no odds. `GoalEvent.Detail` carries worldie | solo | rebound | keeper.

- **Odds.** Every contest is `(a / b)^StepsDuelExponent × modifiers`, on plain 0-100 blends of the player-on-player attributes (`passScore`, `takeOnScore`, `aerialScore`, ...). No skill curve: the sharpening happens once. A 20-point edge is worth about 20 points over a coin toss.
- **The stronger side mostly wins.** A side's share of the attacks is the mean overall rating of the players it fields, raised to `StepsPossessionExponent` (three points better ≈ 61% of the ball), times the opponent's press and line as in the classic model. Overall is used because it is the number managers buy on and it knows nothing about formations. Without this the ball was shared evenly and results were more of a coin flip than the classic model's; with it the favourite wins more and loses less than on the classic model at every rating gap. There is no catch-up for a side that is behind (`StepsScoreEffect` 0).
- **Who meets whom.** Each line is spread across the pitch in line-up order (one player: centre; two: left and right). A contest is met by the nearest player of the defending line that faces it; the attacking left faces the defending right.
- **Formations are numbers, not multipliers.** A defending line that outnumbers the attackers in front of it has a spare man (`StepsSupport`), one that is outnumbered is stretched (`StepsStretched`), an empty line is covered from the best other line, out of place (`StepsLineCover`), and a side with no midfield builds with a long ball (`StepsLongBall`). A forward who beats his man still has to beat the other defender of a two-man line (`StepsCoverScale`). A lone defender has no cover unless his side has two midfielders, when the better tackler of them sits in front of him (`StepsScreenScale`): that is what the Diamond has instead of a second defender. The forwards make a fixed share of a side's take-ons and runs between them (`StepsMidfieldAttackShare`), so a lone striker carries as much of the attack as two do. `FormationProfile` is not read at all: not even for injuries, which here depend on the opponent's aggression and a side's own press and on nothing about either shape.
- **Team-wide multipliers** (captain, items, position boosts, the home boost) reach the odds and the possession weight through `StepsBoostExponent`, set so an item is worth what it is on the classic model.
- **Tactics are trades, not upgrades.** Each setting is sized to be worth about nothing on average across the squads a season deals; what it is worth to any one of them depends on who it has and who it is playing. The checks are `cmd/calibrate -tactics` (invented squads: the direction) and the last table of `-formations` (real squads: the average).
  - *Press* trades time for space. Pressing high takes time from the man on the ball (his pass, his cross, his shot from distance: `StepsPressSqueeze`, scaled by the work rate of the man closing him down) and some of the ball (`StepsPressBall`), and leaves room behind for a take-on, a run in behind or a flick-on (`StepsPressSpace`); long balls go over it more often. It tires the side that does it: its own attacks fade from the hour (`StepsPressFade`, `StepsPressFadeLate`), divided by its legs (`legs`: mean work rate against `StepsTypicalWorkRate`, squared), so a side that can run presses for nothing and one that can't shouldn't. Sitting off is the reverse (`StepsSitOffTime`, `StepsSitOffBall`, `StepsSitOffSpace`). Press a side that wants time on the ball; sit off one that runs at you.
  - *Tempo* is the attacking side's half of the same trade. Fast catches the defence before it is set (`StepsFastBreak` on take-ons and runs) and costs care on the ball (`StepsFastCare` on the pass, the header a cross is aimed at, a shot from distance); slow is the reverse. Quick dribblers want it fast; passers, crossers and a target man want it slow. It also sets how many attacks the match has, for both sides.
  - *Line.* A high line squeezes the pass and is run in behind more, by through balls and flick-ons alike; a deep one is the reverse, so the right line depends on how quick the defenders are and how quick the forwards they face.
  - *Roles.* A Playmaker plays more of the passes, a Target Man is aimed at more (crosses, corners and long balls), a Ball Winner makes up half the ground a stretched line is short and fouls more.

Constants live in `internal/tuning/steps.go`. `cmd/calibrate` is how they are tuned and checked:

| Report | Question |
|---|---|
| `-archives <dir>` | realism (goals, shots, draws...), and how often the stronger side wins by rating gap, on real fixtures |
| `-tactics N` | what each tactic setting is worth to each kind of squad and against each kind of opponent: a setting that pays for everyone is not a choice |
| `-attributes N` | what +5 of each attribute is worth (`-attribute-delta` to change the 5), to catch a dominant one or a worthless one |
| `-levers N` | what an item boost, each tactic and each role is worth |
| `-formations <dir>` | the formations gate: every team's best line-up per shape against a mixed field |

Formation balance is judged by the formations gate on real squads, and on squads dealt afresh from the season's dataset (`-deal`), not on one identical squad. The classic and contest models both passed the identical-squad test while one formation was the worst choice for every real team; and identical squads flatter a shape that needs two good players in two lines, because real squads rarely have them. A shape's worth also depends on what the rest of the league plays, so the gate measures against a mixed field and prints the head-to-head table. `testdata.TypicalTeam` / `ModestTeam` are median FC 27 players by position, for tests that need real squad shapes.
