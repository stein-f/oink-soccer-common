# Attribute glossary

Every player attribute the engine consumes, what it represents in real football, and exactly which algorithms it shows up in. Use this as a recruitment / role-decision reference.

All attributes are on a **0-100** scale. The engine applies a skill curve `(rating/100)^4 × 100` per-player before team aggregation, so an 87-rated attribute is *much* stronger than 78. See `tuning.SkillCurve` in the engine.

> **There is no separate `Passing` attribute.** Passing is folded into `ControlRating` — it represents the broader "ball retention, on-ball decisions, distribution, vision" composite. A high-passing playmaker is a high-`ControlRating` midfielder. Other absences are at the bottom of this doc under [What about... ?](#what-about-).

---

## Quick reference

| Attribute | Code | Drives | Backfill |
|---|---|---|---|
| `GoalkeeperRating` | GKP | Keeper saves | — |
| `DefenseRating` | DEF | Outfield defense (positioning) | — |
| `ControlRating` | CTL | Team control / passing / build-up | — |
| `AttackRating` | ATK | All shooting (every chance type) | — |
| `SpeedRating` | SPD | Sprints + defensive chase | — |
| `AggressionRating` | AGG | Injury risk on opponents | — |
| `WorkRate` | WOR | Midfield engine (especially under press) | `SpeedRating` |
| `Finishing` | FIN | Open Play + 1-on-1 conversion | `AttackRating` |
| `Heading` | HED | Cross + Corner conversion | `AttackRating` |
| `Technique` | TEC | Long Range + Free Kick + corner *delivery* | `ControlRating` |
| `Composure` | COM | Penalty conversion + Captain quality | `ControlRating` |
| `Tackling` | TAC | Outfield defense (dispossession) | `DefenseRating` |

---

## Core (always populated)

### `GoalkeeperRating` (GKP)
Shot-stopping. Used only by goalkeepers; outfielders ignore it.

- **Formula:** `defense_keeper = (goalkeeperRating*5 + speedRating) / 6` (`scoring.go: rawDefense`).
- **Tactics:** none — keeper scoring is tactic-invariant.
- **Roles:** Captain quality uses GoalkeeperRating instead of ControlRating when the captain is a keeper.

### `DefenseRating` (DEF)
Positioning, marking, reading the game. The base of outfield defense.

- **Formula:** `defense_outfield = (defenseRating*5 + tackling*2 + speedRating) / 8` (default weighting).
- **Tactics:** `LineHeight` shifts the weights — deep line `(defense*6 + tackling*2 + speed*0)/8` rewards positioning, high line `(defense*3 + tackling*2 + speed*3)/8` flattens defense and rewards speed.
- **Roles:** Ball Winner's focal-point weighting amplifies a strong-defense player's contribution to `teamDefense`.

### `ControlRating` (CTL) — *also covers passing*
Ball retention, passing accuracy, on-ball decisions, vision. The "midfield IQ" composite. **There is no separate Passing attribute** — it lives here.

- **Formula:** `control = (controlRating*4 + workRate) / 5` (default).
- **Tactics:** `Press` shifts the balance — high press `(ctrl*3 + workRate*2)/5` puts more weight on `WorkRate`; low press `(ctrl*5 + workRate)/6` leans further into raw skill.
- **Roles:** Captain quality uses ControlRating (or GoalkeeperRating for keepers) + Composure. Playmaker amplifies a strong-control player's contribution to `teamControl`.
- **Backfills:** Technique and Composure default to ControlRating when their explicit fields are unset.

### `AttackRating` (ATK)
Shooting power, the universal "scoring" base. Appears in every chance type's attacker formula.

- **Formulas (chance type → contribution):**
  - Open Play: `(atk*2 + finishing + speed) / 4`
  - Cross: `(atk*2 + heading*2 + speed) / 5`
  - Corner: `(atk*2 + heading*3) / 5`
  - Long Range: `(atk*2 + technique*3) / 5`
  - Free Kick: `(atk + technique*3) / 4`
  - Penalty: `(atk*2 + composure*3) / 5`
  - 1-on-1 Breakaway: `(atk + finishing + speed*3) / 5`
- **Backfills:** Finishing and Heading default to AttackRating when unset.

### `SpeedRating` (SPD)
Sprint speed and defensive chase-back. The single physical for both attacking transitions and defensive recovery.

- **Formulas:**
  - Outfield defense: `(defense*5 + tackling*2 + speed) / 8` (default).
  - Keeper defense: `(goalkeeperRating*5 + speed) / 6`.
  - Open Play: `(atk*2 + finishing + speed) / 4`.
  - Cross: `(atk*2 + heading*2 + speed) / 5`.
  - 1-on-1 Breakaway: `(atk + finishing + speed*3) / 5` — speed-dominant.
- **Tactics:** `LineHeight: high` triples Speed's weight in outfield defense; `LineHeight: deep` zeroes it.
- **Backfills:** WorkRate defaults to SpeedRating when unset.

### `AggressionRating` (AGG)
How aggressively the player engages — drives **opponent** injury risk.

- **Formula:** Each team's average aggression scales the other team's per-player injury probability via `tuning.AggressionMaxNoInjuryReduction` (max 50% reduction in the "no injury" weight at aggression=100).
- **Tactics:** `Press: high` adds its own injury bump independently — they stack.

---

## Specialist (optional v2 attributes)

Each specialist drives a single chance type. All optional — when zero the engine backfills from a v1 equivalent so legacy rosters keep behaving consistently.

### `Finishing` (FIN)
Open-play conversion, composure in the box.

- **Backfill:** `AttackRating`.
- **Formulas:** Open Play `(atk*2 + finishing + speed) / 4` and 1-on-1 Breakaway `(atk + finishing + speed*3) / 5`.

### `Heading` (HED)
Aerial duels and headed shots.

- **Backfill:** `AttackRating`.
- **Formulas:** Cross `(atk*2 + heading*2 + speed) / 5`, Corner `(atk*2 + heading*3) / 5`.
- **Roles:** Target Man bonus (×2 selection weight on Cross + Corner) is most valuable when paired with high Heading. Tagging a low-Heading striker as Target Man wastes the slot — they'll be picked more but score the same.

### `Technique` (TEC)
Set-piece accuracy + long-range power (curve, FK accuracy, long-shot strength). Also drives **corner delivery quality**.

- **Backfill:** `ControlRating`.
- **Formulas:** Long Range `(atk*2 + technique*3) / 5`, Free Kick `(atk + technique*3) / 4`.
- **Tactics:** When a `SetPieceTaker` is named, their Technique scales corner conversion via `cornerDeliveryFactor` — roughly ×0.84 at technique=20 to ×1.16 at technique=100. The taker themselves doesn't head their delivery; the finisher is picked separately by Heading.

### `Composure` (COM)
Clutch finishing under pressure.

- **Backfill:** `ControlRating`.
- **Formulas:** Penalty `(atk*2 + composure*3) / 5`.
- **Roles:** Half of captain quality (`(primarySkill + composure) / 2`). A composed leader tilts both team-wide and self captain multipliers.

### `Tackling` (TAC)
Active dispossession + interception.

- **Backfill:** `DefenseRating`.
- **Formula:** outfield defense `(defense*5 + tackling*2 + speed) / 8` (default). Tackling weight stays at ×2 across all line heights.
- **Roles:** Ball Winner amplifies the player's defense score (which Tackling is a major component of).

### `WorkRate` (WOR)
Sustained running, pressing engine, midfield support.

- **Backfill:** `SpeedRating`.
- **Formula:** control `(controlRating*4 + workRate) / 5` (default).
- **Tactics:** `Press` is the big lever — high press shifts midfield scoring to `(ctrl*3 + workRate*2) / 5`, doubling WorkRate's weight; low press flattens it to `(ctrl*5 + workRate) / 6`.

---

## Display-only

Fields on `PlayerAttributes` that the simulation doesn't consume.

### `OverallRating`
Headline number set by the allocation pipeline. Engine reads raw attributes directly — bumping `OverallRating` doesn't change match outcomes. UI uses it for sort/display only.

### `PlayerLevel`
Tier classification (Bronze / Silver / Professional / World class / Legendary). Drives the card colour in the UI and rarity in allocation. Engine doesn't read it.

### `PrimaryPosition` / `Positions` / `Tag`
Eligibility metadata. The engine applies `tuning.OutOfPositionScale` (×0.85) when `SelectedPosition` isn't in the player's allowed positions. Tag is informational.

### `BasedOnPlayer` / `BasedOnPlayerURL`
Display-only — the FIFA player a card is themed around.

---

## Per-action multipliers (not attributes, but they shape the score)

These aren't player attributes but show up in the same scoring math, so worth noting:

- **Out-of-position penalty:** ×0.85 if a player's `SelectedPosition` isn't in their `Positions` list.
- **Injury reduction:** stat reductions per severity — Low ×0.95, Medium ×0.90, High ×0.85.
- **Captain self-boost:** ±2.4% on the captain's own scores, scaled by their quality.
- **Captain team-boost:** ±2.4% team-wide on control + defense, scaled by captain quality.
- **Playmaker / Ball Winner intra-position weight:** ×2.0 within their position group when computing `teamControl` / `teamDefense`.
- **Target Man selection weight:** ×2.0 selection weight on Cross + Corner (selection only — score still uses Heading).
- **Corner delivery factor:** ×0.80–1.20 based on the named SetPieceTaker's Technique.

---

## What about... ?

Attributes you might expect from FIFA / Football Manager that aren't separate fields here. The engine collapses them into the simpler set above:

- **Passing** → `ControlRating`. Passing accuracy + vision + on-ball IQ all roll up here.
- **Dribbling** → split between `ControlRating` (carrying / build-up) and `Technique` (close control on shots).
- **Stamina** → `WorkRate` covers running output. Per-team press fatigue is modelled separately and isn't an attribute.
- **Strength / Jumping** → `Heading`. Aerial contests are where these matter, so they're folded in.
- **Reflexes / Diving / Handling / Kicking / Positioning (GK)** → `GoalkeeperRating`. The GK score is tactic-invariant and one number captures it.
- **Long Shots** → `Technique`. Same attribute that drives Free Kicks.
- **Shot Power / Volleys / Curve** → split between `AttackRating` (the universal shooting base) and the chance-type specialists (Heading for volleyed crosses, Technique for curlers).
- **Pace, Recovery** → `SpeedRating`. Earlier v2 had these as separate fields; consolidated for a simpler attribute model. See [v1-vs-v2.md](v1-vs-v2.md).
- **Defensive Awareness, Marking, Standing/Sliding Tackle** → split between `DefenseRating` (positioning, marking, reading) and `Tackling` (active dispossession).
- **Mentality / Composure / Penalty Power** → `Composure` for the clutch component; `AttackRating` for the power.
- **Leadership / Reactions / Vision (off-ball)** → not separately modelled. Captain quality uses `ControlRating` + `Composure` as the proxy for "leader-type player."
