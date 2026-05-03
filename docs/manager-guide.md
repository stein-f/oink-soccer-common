# Manager's Guide to the Oink Soccer Engine

How matches play out, what each attribute does, and how to build a team that suits your style.

> Looking for the *why* behind v2? See [v1 → v2 differences](v1-vs-v2.md). Looking for the technical architecture? See [architecture.md](architecture.md).

---

## How a match is decided

Every match runs in five phases. The same `(seed, your lineup, opponent lineup)` will always produce the same match — there is no hidden randomness, no clock, no luck of the draw the second time around.

1. **Tempo.** The engine decides how many chances the match will produce. Two attacking formations with fast tempo can rack up 15+ chances; two defensive formations playing slow can produce as few as 5. This is a single decision per match.
2. **Schedule.** Those chance minutes get scattered across the 90 minutes, with a deliberate late-game weighting (more goals come in the second half — same as real football).
3. **Possession.** For each chance, the engine rolls "who has the ball" weighted by both teams' control scores. A team with better control of the midfield gets more chances.
4. **Resolve.** For each chance, the engine picks a chance type (Open Play / Cross / Corner / Long Range / Free Kick / Penalty / 1-on-1), picks which of your players takes the shot, and rolls goal vs miss based on attacker quality vs defender quality. Different chance types reward different player builds — see below.
5. **Injuries.** After the match, the engine rolls injury events for each team. Aggressive opponents and high-press tactics raise injury risk.

---

## Player attributes

Your players have three groups of attributes: **core skills**, **physicals**, and **specialists**. Each one matters in a specific situation; building a balanced team means making sure each chance type has someone good for it.

### Core skills (1-100)

| Attribute | What it does |
|---|---|
| `AttackRating` | How dangerous a player is when shooting (every chance type). The universal "shooting" base. |
| `ControlRating` | How well a midfielder keeps and circulates the ball. Drives team chance creation. |
| `DefenseRating` | Positioning, awareness — how well an outfield defender reads the game. |
| `GoalkeeperRating` | Saves only — the keeper's contribution to team defense. Ignored on outfield players. |
| `AggressionRating` | Drives the *opponent's* injury risk against this player. High-aggression teams cause more injuries. |

### Physicals (1-100)

These were one number in v1 (`SpeedRating`). v2 splits them so a fast attacker doesn't get a free defensive boost.

| Attribute | What it does | Where it shows up |
|---|---|---|
| `Pace` | Top speed. Gets you on the end of through-balls and chasing breakaways. | Open play, crosses, **dominates 1-on-1 breakaways** |
| `Recovery` | Sprint-back speed. Catches you up when the opponent breaks. | Defense, especially with a high defensive line |
| `WorkRate` | Stamina / coverage. The "doing the running" attribute. | Midfield control, especially under high press |

### Specialists (1-100)

These are how a target man and a poacher feel different even when their `AttackRating` is identical. Each one drives a specific chance type. All are optional — if you don't set them, they fall back to a sensible default (legacy rosters keep working).

| Attribute | What it does | Best for |
|---|---|---|
| `Finishing` | Open-play conversion. Composure in the box. | Open Play, 1-on-1 breakaways |
| `Heading` | Aerial duels and headers. | Corners, crosses |
| `Technique` | Set-piece accuracy + long-range power (curve, FK accuracy, long-shot strength). | Long range, free kicks |
| `Composure` | Clutch finishing under pressure. | Penalties |
| `Tackling` | Active dispossession + interception. | Outfield defense |

---

## Chance types — what each one rewards

Every shot in a match is one of seven chance types. The engine rolls which type you get on each chance, and **the type determines which attributes drive the score**. A player who is great on one chance type can be mediocre on another.

| Chance | Roughly how often* | Attacker score formula | Best build |
|---|---|---|---|
| **Open Play** | ~30% | `(atk*2 + finishing + pace) / 4` | Well-rounded forward |
| **Cross** | ~19% | `(atk*2 + heading*2 + pace) / 5` | Striker arriving on a delivery |
| **Corner** | ~12% | `(atk*2 + heading*3) / 5` | **Aerial specialist** — pace doesn't matter |
| **Long Range** | ~12% | `(atk*2 + technique*3) / 5` | Technical midfielder |
| **Free Kick** | ~12% | `(atk + technique*3) / 4` | Set-piece specialist (technique-pure) |
| **Penalty** | ~8% | `(atk*2 + composure*3) / 5` | Clutch finisher — pace doesn't matter |
| **1-on-1 Breakaway** | ~8% | `(atk + finishing + pace*3) / 5` | **Pure speedster** |

\* approximate frequencies; the engine bans back-to-back duplicates (no "Corner. Corner. Corner.") so there's some variation.

**Key insight**: a target man (high `Heading` 92, low `Pace` 55) is your best player on a corner. A speedster (high `Pace` 92, low `Heading`) is your best on a 1-on-1 breakaway. Build a squad with both and the engine will pick the right player for the moment.

### How the goal/miss roll works

For each chance:

```
attacker score = (chance-type formula above) × your formation's chance-quality
                                              × any tactic boost
                                              × per-chance type multiplier
defender score = opponent's team defense × per-chance type multiplier

goal probability = attacker / (attacker + defender)
```

Then a random roll: if `random < probability`, it's a goal. Otherwise a miss.

The skill scaling curve (next section) is what makes a top striker actually feel top.

---

## Why an 87-rated striker is far better than a 78-rated one

The engine applies a **skill curve** to every player score: `(rating / 100)^4`. Linearly, an 87 vs a 78 is barely a 10% gap; on the curve, it's a 41% gap.

| Raw rating | After curve |
|---|---|
| 50 | 6 |
| 70 | 24 |
| 80 | 41 |
| 85 | 52 |
| 87 | 57 |
| 90 | 66 |
| 95 | 81 |
| 100 | 100 |

This is what makes recruiting a top player meaningful — without the curve, the random factor in chance resolution would flatten skill differences and make matches feel like coin flips. With the curve, England (avg ~87) beats Wales (avg ~78) in roughly 7 out of 10 matches in our skill-gap tests, which matches real-world intuition.

---

## Formations

Every formation has a **5-axis trade-off profile**. None strictly dominates another (we ran 5,000-game trials per matchup to verify; the home-win-rate spread across all formation pairs is under 3%).

| Formation | Shape | Possession | Chance Volume | Chance Quality | Defense | Injury Risk |
|---|---|---|---|---|---|---|
| **The Pyramid** | 2-1-1 (defensive) | Neutral | Lower | Slight bonus | **Bonus** | Neutral |
| **The Diamond** | 2-1-1 (balanced) | **Bonus** | Neutral | Neutral | Neutral | Neutral |
| **The Y** | 1-1-2 (attacking) | Neutral | Bonus | Bonus | Penalty | Neutral |
| **The Box** | 2-0-2 (direct) | Neutral | Bonus | Bonus | Penalty | **Higher** |

**How to think about it:**

- **Pyramid** keeps the score down and threatens on the counter. Use it as a road favorite or to protect a lead.
- **Diamond** wins the midfield. The extra midfielder gives you more of the ball — your attackers see more chances.
- **Y** loads the front line. You'll create more chances and convert better, but you'll concede more too.
- **Box** is direct play — two strikers, no midfield. High variance, high goal output, more injuries.

Two attacking formations playing each other typically produce 7-15 chances; two defensive ones produce 5-9. The chance count comes from the *combined* style of both teams.

---

## Tactics — manager levers

These are optional. Leaving them blank gives you the neutral baseline.

### Press

How aggressively your team chases the ball.

| Press level | Effect on opponent's control | Effect on your injury risk | Effect on which midfielders matter | Late-game fatigue |
|---|---|---|---|---|
| Low | +2% (passive — they keep the ball easier) | -5% | Skill-heavy: technicians shine | None |
| Medium / none | Baseline | Baseline | Baseline | None |
| High | -6% (you disrupt their build-up) | +10% | **Work-rate-heavy: stamina midfielders shine** | **Yes — see below** |

**Press fatigue is the real cost of pressing high.** A team that's been chasing all match physically tires:

- Minutes 0-59: fresh, no penalty.
- Minutes 60-74: -10% to your attack quality. Your strikers are blowing.
- Minutes 75-90: -18% to your attack quality. Fully gassed.

Roughly half of all chances fall in the 60+ minute window, so a high-pressing team's average attack output drops by about 10-12% over the full match. The opponent-control boost (+6% possession-pressure) is roughly cancelled out by the late-game fatigue, leaving press as a stylistic *trade* rather than a strict upgrade — you swap "scoring more late" for "letting them score less" plus the next-game injury pressure.

If you press high, two recommendations:
1. Recruit midfielders with high `WorkRate` — they're more effective under press.
2. Don't press high if you're chasing a goal late. The 75+ penalty kicks in exactly when you most need a goal.

### Tempo

How quickly your team plays.

| Tempo | Effect on chance count | Effect on chance quality |
|---|---|---|
| Slow | -8% | +5% (more deliberate shots) |
| Normal | Baseline | Baseline |
| Fast | +10% | -4% (more rushed shots) |

Fast tempo is a volume play; slow tempo is a quality play. The expected goals are roughly equal — pick based on whether your strikers are clinical or your midfield is creative.

### Line Height

How high your defensive line plays.

| Line | Effect on opponent's control | Effect on your defense | Effect on which defenders matter |
|---|---|---|---|
| Deep | +3% (opponent has midfield space to dictate) | +5% (compact, well-organised back line) | **Positional defenders** (high `DefenseRating`) |
| Normal | Baseline | Baseline | Baseline |
| High | -3% (you compress the pitch and pressure them) | -4% (brittle to balls in behind) | **Fast defenders** (high `Recovery`) |

This is a real trade-off — deep gives the opponent more of the ball but lets you defend it well; high suppresses the opponent but leaves you exposed. Pair a deep line with positional defenders, or a high line with fast defenders. The wrong combination leaves you exposed.

### Set-piece taker

Name a specific player to take every Free Kick, Corner, and Penalty for your team. Pick someone with high `Technique` (free kicks), `Heading` (corners), and `Composure` (penalties). A specialist with all three is worth a lot.

---

## Player roles

Optional tags you can attach to a player to reshape their contribution.

| Role | What it does | Best for |
|---|---|---|
| **Captain** | +3% to your whole team's control + defense | One per team — pick a leader |
| **Target Man** | +100% selection weight on corners + crosses | Aerial striker — they'll get the ball when it goes in the air |
| **Playmaker** | +10% to that player's control score | High-`ControlRating` midfielder |
| **Ball Winner** | +10% to that player's defense score | High-`Tackling` midfielder or defender |

You can stack roles across multiple players — a Captain + Playmaker + Ball Winner + Target Man lineup is legal.

---

## Building a team — three concrete examples

### "Counter-attacking" team

Style: Pyramid formation, deep line, slow tempo, low press.

Recruit:
- Goalkeeper with high `GoalkeeperRating` and decent `Recovery` (saves through-balls).
- Two defenders with high `DefenseRating` and `Tackling` (positional, not fast — deep line means they don't need pace).
- One midfielder with high `Technique` and `ControlRating` (long passes for counters).
- Striker with high `Pace`, `Finishing`, and `Composure` (1-on-1s and breakaways are your bread and butter; clinical conversion).

Why it works: low press + deep line = solid defense, lots of misses by the opponent. Slow tempo = you don't waste your few chances. High-`Technique` long-range and high-`Pace` 1-on-1 breakaways are your goal sources.

### "Possession" team

Style: Diamond formation, normal line, normal tempo, medium press.

Recruit:
- Two well-rounded midfielders, one tagged as **Playmaker** (high `ControlRating`).
- Defenders with balanced `DefenseRating` / `Tackling` / `Recovery` — Diamond defends well.
- Striker with high `AttackRating` and `Finishing` for open play.

Why it works: the Diamond's possession bonus (3%) compounds over the match — you'll see 60-65% of chances. Most of your goals come from open play, so finishing matters more than specialist attributes.

### "Aerial bombardment" team

Style: Box formation (or Y), high tempo, high line, target-man specialist.

Recruit:
- Striker with **maxed `Heading`** tagged as **Target Man** (gets ~2× selection on corners and crosses).
- Set-piece taker with high `Technique` (delivers great corners).
- Fast defenders with high `Recovery` (high line means they need to chase).
- Captain on your most experienced player.

Why it works: high tempo creates more chances; many of those will be crosses and corners, where your aerial specialist dominates. The high line is risky but creates more counter-attack opportunities. Heading + Technique combo is a goal machine.

---

## Common pitfalls

**Don't recruit on `OverallRating` alone.** A player with `OverallRating` 85 split across `AttackRating` 90, `Pace` 80, no `Heading` is a different player from one with `AttackRating` 80, `Pace` 75, `Heading` 95. The first is your open-play striker; the second is your corner specialist. They suit different lineups.

**Don't pair tactics with the wrong builds.** A high line with slow defenders is a goal-conceding machine. A high press with low-`WorkRate` midfielders fizzles by the 60th minute. Match your tactics to the players you have.

**Don't ignore the goalkeeper.** Saves are a huge fraction of your defensive total. A 90-rated keeper vs a 75-rated keeper is the difference between conceding 1 and conceding 2 in most matches.

**Don't forget injury risk.** Box formation + high press + injury-prone players is a recipe for a bench full of players you can't field. Spread the load.

**Don't expect a single tactic to fix a weak squad.** The skill curve makes top players genuinely better; tactics shift the trade-offs but don't manufacture quality. A perfectly-tactically-set Wales still beats England maybe 1-2 games in 10.

---

## Verifying a match

Every match is reproducible. The engine takes a seed (derived from an Algorand block hash for verifiability) and produces the same events every time. If you want to re-verify a historical match, the `algorand` package can re-derive the seed from the block round, and feeding it back through `RunGameWithSeed` produces the identical event sequence. There's no hidden randomness, no server-side bias.
