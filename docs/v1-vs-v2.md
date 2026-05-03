# v1 → v2: What Changed and Why

The match engine got a clean-room rewrite. This document covers what's different and why we did it. If you just want to play, the [Manager's Guide](manager-guide.md) is what you need. This doc is for people asking "why did matches feel different after the upgrade" or "can I still trust v1 outcomes."

---

## The short version

v2 is a better simulation of football. The same lineups will produce broadly similar match scores, but match *flow* is more realistic, formation choice actually matters, player builds are more meaningful, and tactics genuinely change outcomes. The technical contract (`(seed, lineups) → events`) is preserved — same seed, same input, same output, every time, just like v1.

---

## Why we rebuilt

v1 was the first cut of the engine. It worked, it was deterministic, and it was fast — but as we played more matches and looked at outcomes, four problems stacked up:

1. **One attribute, three uses.** v1 had a single `SpeedRating` that got folded into a player's control score, attack score, *and* defense score. A fast attacker silently boosted their team's defense. There was no way to model a slow target man without nerfing them across the board.
2. **Formation imbalance.** "The Box" formation was strictly better than the others — it had a special case in the team-control calculation that gave its attackers 60% of the weight. Our internal balance test reported a ~9% home-win-rate spread across formations, well above what felt fair.
3. **Control was too dominant.** A 4-point swing in a midfielder's `ControlRating` could move match outcomes by 10%+ — much more than any other attribute. Ratings near the top end (85+) compressed because the engine used them linearly.
4. **No tactical levers.** Managers couldn't choose how to play. Every team played the same way; the only choice was the formation.

Patching v1 in place would have meant unwinding compounding bugs (the speed triple-count, the Box special-case, the control dominance) — easier to start from a clean design.

---

## What's preserved

The locked surface that downstream consumers depend on:

- `RunGameWithSeed(rand, home, away)` is still the entry point.
- `GameEvent`, `GoalEvent`, `MissEvent`, `Injuries`, `GameStats`, `FormationConfig`, `PlayerAttributes`, etc. — same types, same fields.
- Every match is still a pure function of `(seed, lineups)`. Same seed = same events, forever.
- The Algorand block-hash → seed derivation works identically.
- Existing rosters work without modification — every new attribute on `PlayerAttributes` is optional and falls back to a sensible composite.
- Historical matches can still be re-verified.

You don't need to change any data to use v2. You *can* enrich rosters with the new attributes to unlock more of the engine's depth.

---

## What's different

### 1. Player attributes are split

**v1**: One `SpeedRating`, folded into all three of control/attack/defense.

**v2**: Three physicals, plus five specialists. Each one matters for a specific situation.

| v1 attribute | v2 equivalent | Used for |
|---|---|---|
| `SpeedRating` | `Pace` | Open-play attacking + 1-on-1 breakaways |
| (same) | `Recovery` | Defense (especially high line) |
| (same) | `WorkRate` | Midfield control (especially high press) |
| `AttackRating` | (still there) | All shooting |
| (new) | `Finishing` | Open-play conversion |
| (new) | `Heading` | Corners + crosses |
| (new) | `Technique` | Long range + free kicks |
| (new) | `Composure` | Penalties |
| `DefenseRating` | (still there) | Outfield positioning |
| (new) | `Tackling` | Outfield dispossession |
| `GoalkeeperRating` | (unchanged) | Saves only |

**Backward compatibility**: any of the new attributes left at zero falls back to a reasonable composite (e.g. `Heading` falls back to `AttackRating`). Old rosters work as-is — they just behave like every striker is equally good in the air, which is roughly what v1 did anyway.

### 2. The shot itself depends on the situation

**v1**: Every chance was scored the same way — `attackRating × something + speedRating × something`.

**v2**: Each chance type runs its own formula. A corner uses `(attack*2 + heading*3)`. A penalty uses `(attack*2 + composure*3)`. A 1-on-1 breakaway uses `(attack + finishing + pace*3)`.

This is why builds matter now. In v1, "best striker" meant "highest `AttackRating` + `SpeedRating`." In v2, your best striker on corners is your aerial specialist; your best on breakaways is your speedster; your best on penalties is your clutch finisher. The engine picks the right player for each chance.

### 3. Tactics actually do something

**v1**: No tactics. Every team played the same way.

**v2**: Four optional levers per team — Press, Tempo, Line Height, Set-Piece Taker — plus four player roles (Captain, Target Man, Playmaker, Ball Winner). Each lever has trade-offs:

- High press disrupts the opponent's build-up, raises your own injury risk, **shifts which midfielders matter** (work-rate over technique), and triggers in-match fatigue (your attack quality drops 10-18% after the 60th minute). The fatigue is what makes press a real choice rather than a free lunch — without it, the only cost would be a next-game injury bump that recovery items can fix.
- High line compresses the pitch but **shifts which defenders matter** (recovery over positioning).
- Fast tempo creates more chances but each one is slightly worse quality.
- Naming a set-piece taker means that player takes every free kick, corner, and penalty.

The same lineup can play very differently depending on tactical choices.

### 4. The skill curve is back

**v1**: Hand-tuned 100-row CSV (`scaling.csv`) approximating `y = a × x^b` with `b ≈ 3.1`. This curve made an 87-rated striker meaningfully better than a 78-rated one.

**v2**: Closed-form `(rating / 100)^4` applied at the player level, before team aggregation. Same idea, no embedded data file. England (~87) beats Wales (~78) in roughly 7 of 10 matches — close to real-world intuition.

We initially tried v2 *without* the curve. The result: an 87 vs 78 matchup was 47% / 23% / 30% — almost a coin flip. With the curve it's 72% / 15% / 13%. The curve is what makes player quality feel like it matters.

### 5. Formations are actually balanced

**v1**: Box formation strictly dominated — its win rate vs other formations was ~9% above the average. The "Box bug" was a special-case in team-control scoring.

**v2**: Every formation has a 5-axis trade-off profile (Possession / Chance Volume / Chance Quality / Defense / Injury Risk). A unit test pins the invariant that no formation strictly dominates another, and a 5,000-game balance harness measures the win-rate spread across all 16 (home × away) formation pairs. That spread is now under 3%.

| Formation | Possession | Chance Volume | Chance Quality | Defense | Injury Risk |
|---|---|---|---|---|---|
| Pyramid | 1.00 | 1.00 | 1.03 | 1.02 | 1.00 |
| Diamond | 1.03 | 1.00 | 1.00 | 1.00 | 1.00 |
| Y | 1.00 | 1.03 | 1.02 | 0.97 | 1.00 |
| Box | 1.00 | 1.05 | 1.05 | 0.96 | 1.05 |

Pick the shape that suits your players, not the shape that "wins."

### 6. Match flow is phase-based

**v1**: Matches were "roll N independent dice" — each chance was a self-contained event with no shared structure.

**v2**: Each match runs five explicit phases — tempo, schedule, possession, resolve, injuries. Chance volume comes from the *combined* tempo of both teams, possession is rolled per chance based on team control, and the chance type drives both who takes the shot and how it's scored. The simulation feels more like a football match and less like a slot machine.

### 7. Chance types actually behave differently

**v1**: Chance types (Penalty, Long Range, Corner, etc.) existed as enum values but were never used — every shot was scored the same.

**v2**: Each chance type has:
- A different attacker-selection bias (defenders go up for corners; mids fire from distance; only attackers on breakaways).
- A different attack-score formula (per the table above).
- A different conversion modifier (penalties × 1.5 attacker boost / × 0.5 defender; long range × 0.7 / × 1.2).
- A target-man bonus on corners and crosses if you've tagged a player.

You'll see them in match commentary now ("Corner... headed home by Smith, target-man specialist").

---

## What's gone

A handful of v1 helpers aren't carried over because they weren't used by anything outside the engine itself:

- `RunGame(home, away)` (no-seed variant) — replaced by `RunGameWithSeed` only. v1's no-seed variant was nondeterministic.
- `DetermineTeamChances`, `DetermineChanceType`, `GetTeamBoost`, `CalculateTeamControlScore`, `CalculateTeamDefenseScore` — internal helpers, now unexported.
- `ScalingFunction`, `DiminishingMultiplier` — internal math, replaced by named tuning constants.
- `cmd/allocation` runner — being rewritten against v2 (next season).

If you import `oink-soccer-common` directly, the [API contract](api.md) lists every supported symbol.

---

## What if I have v1 match data?

It's still valid. v1 is frozen at the repo root, bug-fix only, and continues to work for any historical match you want to re-verify. v2 runs side-by-side as a separate Go module (`v2/`) with its own snapshots and tests.

If you re-run an old match through v2, the events *will* differ — different attribute model, different chance scoring, different formation profiles. That's the point. Use v1 for historical re-verification; use v2 for new matches.

---

## Open work

A few things are still in progress:

- **Allocation pipeline refresh.** v2 reads richer attributes from the FIFA dataset (`Heading`, `Composure`, `Technique`, `Tackling`, etc.), but rosters allocated under v1 only have the legacy attributes populated. Once the current season finishes, we'll re-allocate against the new columns and the engine will fully reflect each player's specialist build instead of relying on backfills. Until then, every player effectively has "average" specialist attributes.

- **v2.0.0 tag.** Now that the formation balance target is met, the engine is ready to be tagged. Coming soon.

---

## Where to go next

- [Manager's Guide](manager-guide.md) — concrete advice on building a team and using tactics.
- [Architecture](architecture.md) — technical detail of the simulation flow.
- [Public API](api.md) — exact contract for downstream consumers.
- [Rebuild plan](rebuild-plan.md) — internal changelog and decision history.
