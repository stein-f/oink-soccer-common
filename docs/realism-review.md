# Engine realism review

*2026-11-14. Review of the v2 engine with a view to making matches more
realistic, including defensive events such as tackles.*

**Status:** step 1 is implemented as `ModelContest` (v2.1.0), selectable per
season next to the unchanged classic model. Calibrated on 480 season-18
fixtures × 20 replays:

| Metric | Classic | Contest | Target |
|---|---|---|---|
| Goals / match | 4.98 | 3.16 | 3.0-3.5 |
| Shots / match | 8.5 | 13.0 | 10-14 |
| Conversion | 58.8% | 24.3% | 25-30% |
| Draws | 11.3% | 18.0% (25% in close fixtures) | 18-22% |
| 0-0 | 0.5% | 4.7% | 3-5% |
| Home − away wins | +2.7 pts | +7.8 pts | ≥ +8 |
| Penalties / match | 0.73 | 0.22 | 0.2-0.3 |
| Wins by 5+ (close fixtures) | 5.8% | 0.6% | < 3% |
| Wins by 5+ (mismatches, 8+ rating gap) | 64.5% | ~20% | — |
| Stronger side wins decided games | 77% | 75% | clearly > 50% |
| Formation home-win spread | < 3% | 2.9% | < 3% |

Mismatches are full squads against placeholder teams; a 20-point rating gap is
still a hiding, just not a 9-0. Steps 2 and 3 (reporting stops, saves and
assists as events) are not started.

## Summary

The engine is well built for what it is: deterministic, balanced across
formations, and every attribute has a job. It is not a match simulation,
though. It rolls a handful of **shots** and decides each one with a single coin
weighted by attack against team defence. Everything a viewer might expect
between those shots (tackles, saves, blocks, fouls, assists) does not exist,
and the numbers it produces are far from football:

| Season 18, 480 league matches | Engine | Real football |
|---|---|---|
| Goals per match | 5.04 | ~2.7 |
| Shot conversion | 59.7% | ~10% |
| Long-range shot conversion | 49% | 3-5% |
| Direct free-kick conversion | 55% | ~6% |
| Penalties per match | 0.73 | ~0.3 |
| Draws | 11.0% | ~25% |
| Home / away wins | 43.8% / 45.2% | ~45% / ~30% |
| 0-0 | 0.0% | ~7% |
| 5-0 or 6-0 | 12% | rare |

(Real figures are 11-a-side for orientation. Five-a-side scores higher, but
the gaps above are far beyond format.)

**Recommendation:** rebuild chance resolution as a short chain of contests
(win the ball, create the chance, get the shot away, beat the keeper) instead
of one roll, and recalibrate to targets. That one change produces tackles,
interceptions, blocks, saves and assists as natural by-products, gives
defenders and keepers visible jobs, and brings the numbers into line. Ship it
in three steps so each is safe on its own (see [Plan](#plan)), starting at a
season boundary.

## How a match works today

1. **Volume**: a total of 5-15 chances from a table keyed by the two
   formations' styles, scaled by tempo (`decideMatchTempo`).
2. **Timing**: each chance gets an independent minute from a late-weighted
   histogram (`scheduleMinutes`).
3. **Possession**: each chance goes to a team by the ratio of team control,
   computed once for the whole match (`pickAttackingTeam`).
4. **Chance type**: drawn from fixed global weights (open play 8, cross 5,
   corner 3, long range 3, free kick 3, penalty 2, one-on-one 2), never the
   same type twice in a row (`pickChanceType`).
5. **Shooter**: picked by position weight times attack score
   (`pickAttacker`).
6. **Outcome**: `p = atk / (atk + def)`, where `def` is the whole defending
   team's aggregate (keeper 35%, defenders 40%, midfield 20%, attack 5%).
   Goal or miss (`resolveChance`).
7. **Injuries**: rolled after the match from opponent aggression.

## What makes it unrealistic, and why

### 1. Every chance is a shot, and most go in

Steps 1-6 only ever produce shots, so "chances" and "shots" are the same thing.
With a single `atk/(atk+def)` roll and chance-type boosts around 1.0, typical
probabilities land at 50-70%. That is why matches average 5 goals from 8
shots, why 0-0 never happens (5+ shots at ~60% each), and why long-range
efforts and free kicks go in half the time.

### 2. Defenders and keepers have no individual actions

Defence is one team number. A specific defender never wins the ball, never
blocks a shot; a keeper never makes a save. `Tackling`, the Ball Winner role
and `GoalkeeperRating` all work, but only by nudging that aggregate, so they
are invisible in highlights, commentary and stats. This is the root of "no
tackles".

### 3. Set pieces aren't caused by anything

Free kicks and penalties come from fixed weights, so every match has the same
expected number regardless of who is playing. In football they come from
fouls, which come from defending: aggressive, poorly-positioned defenders give
away more. `AggressionRating` currently only causes post-match injuries.
Corners similarly come from saves and blocks, which don't exist yet.

### 4. Ratings gaps snowball into blowouts

The skill curve `(rating/100)^6` makes elite players dominate (deliberately;
it sharpens 88-vs-83 into ~65%/18%). Combined with high conversion, a stronger
side scores almost every time it shoots, giving the 12% of 5-0 and 6-0
results. There's nothing to slow it down: a team 4-0 up plays exactly as it
did at 0-0.

### 5. No game state

Team scores are computed once per match. There are no score effects (leaders
sitting deeper, trailing teams pushing), no momentum, no in-match fatigue
beyond the high-press penalty, no in-match injuries or substitutions. Minutes
are independent random draws, so nothing builds towards anything. This is a
big part of why draws are rare: nothing pulls a match back towards level.

### 6. Home advantage isn't showing

lost-pigs applies a flat ×1.05 team boost to the home side
(`HomeAdvantageBoost`), yet away teams won slightly more often in the sample.
At this conversion level a 5% edge is lost in the noise; it should be
re-tested once scoring is recalibrated, and probably expressed through
possession and chance creation rather than a blanket boost.

### 7. No assists, no individual defensive stats

There is no provider on any chance, so assists can't be credited. Keeper
saves, tackles and blocks can't be counted either. The new highlights player
currently *invents* a provider for each move to make it watchable.

## Proposed model: a chain of contests

Keep the outer loop (volume, timing, possession) and replace steps 4-6 with a
sequence. Each step is a contest between two named players, decided by the
same `atk/(atk+def)` shape the engine already uses, so it stays familiar,
deterministic and tunable:

```
possession won by team control (as now)
  │
  ├─ 1. Build-up: carrier (Control, WorkRate) vs a picked defender
  │       (Tackling, Defense, Speed; Ball Winner weighted up)
  │       → TACKLE or INTERCEPTION (defender credited)          ── end
  │       → FOUL (defender Aggression)  → free kick, or penalty
  │                                       if in the box; card roll
  │       → OFFSIDE (rare; attacker Speed vs high line)          ── end
  │       → chance created (provider credited)
  │
  ├─ 2. Chance type chosen from how the chance arose (cross from a
  │       wide carrier, through ball from a playmaker, set piece from
  │       a foul or a corner), not from fixed global weights
  │
  ├─ 3. Shot: shooter (chance-type formula, as now) vs nearest defender
  │       → BLOCKED (defender credited; may become a corner)     ── end
  │       → OFF TARGET (wide / over; skill sets the rate)         ── end
  │       → on target
  │
  └─ 4. Keeper: shot quality vs GoalkeeperRating
          → SAVED (keeper credited; may become a corner)          ── end
          → GOAL (shooter, and provider as the assist)
```

What this buys:

- **Tackles, interceptions, blocks and saves** become real events with a
  named player, so highlights can show them and stats can count them.
- **Defenders and keepers matter individually**: a great keeper saves more,
  a Ball Winner wins more ball, an aggressive centre-back gives away
  penalties.
- **Set pieces have causes**: free kicks and penalties from fouls, corners
  from blocks and saves.
- **Realistic numbers fall out of the structure**: a typical attack fails at
  step 1 or 3, so goals drop without hand-tuned fudge factors (the existing
  `DefenseBiasMultiplier` can go, as its own comment already intends).
- **Assists** come for free from step 1's provider.

Two smaller additions that address points 4 and 5 cheaply:

- **Score effects**: after each goal, nudge the leader's tempo down and the
  trailer's chance volume up for the rest of the match. This makes draws and
  comebacks plausible, and takes the edge off blowouts without softening the
  skill curve.
- **Minute ordering**: draw chances in minute order and let the next chance
  depend lightly on the last (a corner from a save comes a minute later).

### Calibration targets

Proposed for five-a-side, tuned with the existing balance harness
(`TestFormationBalance`) extended to report these:

| Metric | Today | Target |
|---|---|---|
| Goals per match | 5.0 | 3.0-3.5 |
| Shots per match | 8.4 | 10-14 |
| Defensive stops (tackles, interceptions) per match | 0 | 6-10 |
| Shot conversion | 60% | 25-30% |
| Keeper saves per match | 0 | 3-5 |
| Penalties per match | 0.73 | 0.2-0.3 |
| Draws | 11% | 18-22% |
| 0-0 | 0% | 3-5% |
| Wins by 5+ | ~12% | under 3% |
| Formation win-rate spread | <3% | <3% (unchanged) |

## What it touches outside the engine

Shots stay `Goal` and `Miss`, so league tables, shot stats
(`CreateGameStats` already ignores other event types) and goal markets keep
working. New information should be **additive**:

- `Miss` gains a detail: how it ended (`saved`, `blocked`, `wide`, `over`,
  `post`, `bar`) and who stopped it (`KeeperID`, `BlockerID`).
- `Goal` gains `AssistID`.
- One new event type for stops that never became a shot, e.g. `Stop` with
  `DefenderID`, `AttackerID` and a kind (`tackle`, `interception`,
  `offside`).
- Optionally `Foul` and `Card` later.

Consumers that branch on event types today and would need changing **before**
an engine that emits new types ships:

| Where | What it does now | Needed |
|---|---|---|
| lost-pigs `domain/highlights/highlights.go` | Errors on any event that isn't a goal or miss, which **fails the whole match run** | Skip or render new types. **Do this first.** |
| lost-pigs `domain/soccer_game.go`, `soccer_tournament_game.go` | Builds highlights and scorers from goals and misses | Pass new types through |
| lost-pigs `claudeai/prompts.go` | Goal and miss commentary for the AI match report | Mention saves, tackles, assists |
| lost-pigs-discord-bot `command/soccer.go` | Reads archive events | Tolerate new types |
| lost-pigs-ui GIF path (`lib/soccer.ts`) | One GIF per event | Skip new types |
| lost-pigs-ui new pitch (`lib/match`) | Animates goals and misses | Use the real miss ending, keeper, blocker and assist; animate tackles |
| League stats (`soccer_league.go`) | Conversion = goals / chances | Decide whether "chances" means shots |
| OINK Predict lines | 200 engine runs per match | Recalibrate automatically; historic lines won't compare |

Determinism is unaffected in principle: every new contest draws from the same
seeded source. But any change to the draw sequence changes every result for a
given seed, so golden snapshots regenerate, and the new engine should start at
a season boundary so no season mixes two engines (verification is already per
engine version).

## Plan

1. **Recalibrate, no new event types** (engine only). Add the chain
   internally, but still emit only goals and misses. Set pieces from fouls,
   score effects, home advantage through possession, targets above. Nothing
   downstream changes. Earliest safe start: season 19.
2. **Add detail to shots** (additive JSON). Miss ending, keeper, blocker,
   assist. Unlocks keeper saves and assists in stats, accurate highlights,
   and better AI reports. Needs the highlights and UI tolerance changes first.
3. **Add stops** (new event type). Tackles, interceptions and offsides as
   their own events, animated on the new pitch, counted for defenders and
   Ball Winners. Fouls and cards can follow the same path.

Step 1 alone fixes the numbers. Steps 2 and 3 are what make defending
visible, which is the "tackles" ask.

## Open questions

- **Target goals per match.** 3.0-3.5 is a guess at "five-a-side but still
  exciting". Lower is more realistic, higher is more fun to watch.
- **How strong the skill curve should stay.** Score effects take the edge off
  blowouts. If upsets are still too rare, the exponent (6.0) is the lever.
- **Cards.** Fun and realistic, but a red card in five-a-side is brutal, and
  suspensions would need team-selection rules.
- **Season boundary.** Step 1 changes every result for a given seed, so it
  should land between seasons with an announcement, as the move to v2 did.
