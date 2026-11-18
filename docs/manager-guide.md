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

> The full reference (every attribute, every formula it touches, FIFA-style fields that *aren't* here like Passing) lives in [attributes.md](attributes.md). The summary below is enough for picking players; reach for the reference when you need the maths.

### Core skills (1-100)

| Attribute | What it does |
|---|---|
| `AttackRating` | How dangerous a player is when shooting (every chance type). The universal "shooting" base. |
| `ControlRating` | How well a midfielder keeps and circulates the ball. Drives team chance creation. |
| `DefenseRating` | Positioning, awareness — how well an outfield defender reads the game. |
| `GoalkeeperRating` | Saves only — the keeper's contribution to team defense. Ignored on outfield players. |
| `AggressionRating` | Drives the *opponent's* injury risk against this player. High-aggression teams cause more injuries. |

### Physicals (1-100)

| Attribute | What it does | Where it shows up |
|---|---|---|
| `SpeedRating` | Top speed — both attacking sprints and defensive chase-back. | Open play, crosses, **dominates 1-on-1 breakaways**, plus outfield defense (especially with a high line) |
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
| **Open Play** | ~30% | `(atk*2 + finishing + speed) / 4` | Well-rounded forward |
| **Cross** | ~19% | `(atk*2 + heading*2 + speed) / 5` | Striker arriving on a delivery |
| **Corner** | ~12% | `(atk*2 + heading*3) / 5` | **Aerial specialist** — speed doesn't matter |
| **Long Range** | ~12% | `(atk*2 + technique*3) / 5` | Technical midfielder |
| **Free Kick** | ~12% | `(atk + technique*3) / 4` | Set-piece specialist (technique-pure) |
| **Penalty** | ~8% | `(atk*2 + composure*3) / 5` | Clutch finisher — speed doesn't matter |
| **1-on-1 Breakaway** | ~8% | `(atk + finishing + speed*3) / 5` | **Pure speedster** |

\* approximate frequencies; the engine bans back-to-back duplicates (no "Corner. Corner. Corner.") so there's some variation.

**Key insight**: a target man (high `Heading` 92, low `SpeedRating` 55) is your best player on a corner. A speedster (high `SpeedRating` 92, low `Heading`) is your best on a 1-on-1 breakaway. Build a squad with both and the engine will pick the right player for the moment.

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

The engine applies a **skill curve** to every player score: `(rating / 100)^6`. Linearly, an 87 vs a 78 is barely a 12% gap; on the curve the 87 scores nearly double the 78 (43 vs 23).

| Raw rating | After curve |
|---|---|
| 50 | 2 |
| 70 | 12 |
| 80 | 26 |
| 85 | 38 |
| 87 | 43 |
| 90 | 53 |
| 95 | 74 |
| 100 | 100 |

This is what makes recruiting a top player meaningful — without the curve, the random factor in chance resolution would flatten skill differences and make matches feel like coin flips. With the curve, a squad averaging 88 beats a squad averaging 83 about 65% of the time, losing only ~18% — a clear favourite tops the table far more often than not, but never with certainty. (See `tuning.SkillCurveExponent` for why 6.0 is the deliberate ceiling.)

---

## Formations

Every formation has a **5-axis trade-off profile**. None strictly dominates another (we ran 5,000-game trials per matchup to verify; the home-win-rate spread across all formation pairs is under 3%).

| Formation | Shape | Possession | Chance Volume | Chance Quality | Defense | Injury Risk |
|---|---|---|---|---|---|---|
| **The Pyramid** | 2-1-1 (defensive) | Neutral | Lower | Slight bonus | **Bonus** | Neutral |
| **The Diamond** | 1-2-1 (balanced)  | **Bonus** | Neutral | Neutral | Neutral | Neutral |
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
| Medium / none (default) | Baseline | Baseline | Baseline | None |
| High | -6% (you disrupt their build-up) | +10% | **Work-rate-heavy: stamina midfielders shine** | **Yes — see below** |

Medium is the explicit neutral — identical to leaving press unset, mirroring tempo Normal and line height Normal.

Like line height for defenders, press also changes *which attributes* your midfielders are judged on for team control:

| Press | `ControlRating` | `WorkRate` |
|---|---|---|
| Low | ≈83% | ≈17% |
| None / Medium | 80% | 20% |
| High | 60% | 40% — engine matters |

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
| High | -3% (you compress the pitch and pressure them) | -4% (brittle to balls in behind) | **Fast defenders** (high `SpeedRating`) |

The flat modifiers are the small part. The real lever is that line height changes *which attributes* your defenders are judged on:

| Line | `DefenseRating` | `Tackling` | `SpeedRating` |
|---|---|---|---|
| Deep | 75% | 25% | 0% — pace never tested |
| Normal | 62.5% | 25% | 12.5% |
| High | 37.5% | 25% | 37.5% — pace critical |

This is a real trade-off — deep gives the opponent more of the ball but lets you defend it well; high suppresses the opponent but leaves you exposed. Pair a deep line with positional defenders, or a high line with fast defenders. The wrong combination loses far more than the headline -4% — a slow back line playing high is judged on pace it doesn't have, while slow positional defenders playing deep give up nothing.

### Set-piece taker

Name a specific player to take your team's Free Kicks, Corners, and Penalties. The taker isn't always the player who scores — what they actually do depends on the chance type:

- **Free Kicks (direct):** the taker takes the shot. Pick high `Technique`.
- **Penalties:** the taker takes the shot. Pick high `Composure`.
- **Corners:** the taker *delivers* the corner; somebody else heads it home. The taker's `Technique` boosts the chance of any header converting (great delivery → more dangerous chance), but the finisher is picked separately by `Heading` + position. The taker is excluded from the finisher pool — they can't head their own delivery.

Leave it blank and set pieces fall to whoever the chance lands on — the normal weighted attacker pick, not necessarily your best taker — and corners get no delivery boost (neutral 1.0). Naming a taker is strictly an upgrade.

Corner delivery quality scales with the taker's `Technique`:

| Taker `Technique` | Header conversion chance |
|---|---|
| 100 | +16% |
| 85 | +10% |
| 60 | Neutral (average pro) |
| 40 | -8% |
| 20 | -16% |

So a high-`Technique` midfielder is your ideal corner taker even if their `Heading` is poor. Their delivery makes your aerial striker (high `Heading`, ideally tagged **Target Man**) more dangerous. A specialist with all three of `Technique`, `Composure`, and a separate aerial finisher with high `Heading` is the strongest set-piece setup.

---

## Player roles

Optional tags you can attach to a player to reshape their contribution.

| Role | What it does | Best for |
|---|---|---|
| **Captain** | Two effects, both small (≈ ±2.4% at the extremes) and both scaled by *captain quality* — `(ControlRating + Composure) / 2`, or `(GoalkeeperRating + Composure) / 2` for keepers: (1) a team-wide multiplier on control + defense, and (2) a self-multiplier on the captain's own play. A high-quality captain lifts both; a low-quality captain drags both. **Tagging a poor leader hurts you on both axes** — armband isn't a free boost. | A composed, intelligent senior — one per team |
| **Target Man** | +100% selection weight on corners + crosses (selection only — the score still comes from `Heading` via the chance formula, so tagging a non-aerial striker is a wasted slot) | Aerial striker — they'll get the ball when it goes in the air |
| **Playmaker** | Becomes the focal point of that position group's contribution to team control — their score is weighted 2× vs the other players in the same group when computing the group's mean. **Tag a strong controller and team control rises; tag a weak one and it drops.** Not a free boost. | Your strongest `ControlRating` player in the position |
| **Ball Winner** | Same focal-point mechanic as Playmaker but on the defensive side — the Ball Winner's defense score is weighted 2× within their position group when computing team defense. **Tag your destroyer and the team's defense lifts; tag a weak defender and it drops.** Not a free boost. Note: a Ball Winner who's the only player in their position group has no effect (no group mean to drag). | High-`Tackling` / `DefenseRating` midfielder or defender |

You can stack roles across multiple players — a Captain + Playmaker + Ball Winner + Target Man lineup is legal.

---

## Season 19 onwards — tactics and roles

Everything above describes the engine seasons 1 to 18 were played on. From season 19 a match is played as attacks between named players: a pass forward, then a take-on, a run in behind, a cross, a long ball or a shot from distance, each one a contest between two of them. Tactics and roles change the odds of particular contests instead of a team score, so what a setting is worth depends on who you have and who you are playing. None of them is better on its own, which is why there are no percentages below. (The in-product help, `TacticsInfoModal.tsx` in lost-pigs-ui, carries the same text.)

### Press

How hard your team closes the ball down. It trades time for space.

| Press | Their player on the ball | Room behind you | Your legs |
|---|---|---|---|
| Low | Has time: passes, crosses and long shots come off more, and they see more of the ball | None: dribbles and runs in behind are harder | No cost |
| Medium (default) | Baseline | Baseline | No cost |
| High | No time: passes, crosses and long shots come off less, and they see less of the ball | Plenty: dribbles, runs in behind and flick-ons are easier | Your own attacks fade from the 60th minute, and more from the 75th |

A press is only as good as the running in it. How much time a high press takes away depends on the `WorkRate` of the player closing the ball down, and how much your own attack fades depends on the average `WorkRate` of your outfield players: a side that can run all day barely notices, one that can't is finished by the 75th minute. `WorkRate` does nothing for a side that doesn't press high.

Pressing high also gives away more fouls when you win the ball (more free kicks and penalties against you), raises your own injury risk, and invites the long ball, which goes over a press. Sitting off does the reverse of all three.

- Press high with high-`WorkRate` players against a side that wants time on the ball: passers, crossers, long-range shooters.
- Don't press quick dribblers or forwards who run in behind: you are handing them the room.
- Sit off a side that runs at you. Don't sit off good crossers with a big forward to aim at.
- Being pressed? A long ball to a strong Target Man skips the press.

### Tempo

How quickly your team plays when it has the ball.

| Tempo | Your dribbles and runs in behind | Your passes, crosses and long shots | Attacks in the match |
|---|---|---|---|
| Slow | Harder: the defence is set | Better: fewer passes go astray and the cross is one worth heading | Fewer, for both sides |
| Normal (default) | Baseline | Baseline | Baseline |
| Fast | Easier: you catch them before they are set | Looser: more passes go astray and crosses are harder to win | More, for both sides |

- Play fast with quick dribblers (`Dribbling`, `Pace`) and forwards who run in behind (`Pace`, `Positioning`).
- Play slow with good passers and crossers (`Passing`, `Crossing`), a forward who wins headers, or midfielders who shoot from distance.
- Your team already leans towards what its players are good at. Tempo is how hard you lean.

### Line Height

How far up the pitch your defence stands.

| Line | Their passing | Balls in behind you |
|---|---|---|
| Deep | Easier, and they see a little more of the ball | Tried less often, and they lose the race more |
| Normal (default) | Baseline | Baseline |
| High | Harder, and they see a little less of the ball | Tried more often, and they win the race more |

A ball in behind is a race between their forward (`Pace`, `Positioning`) and your defender (`Pace`, `Marking`). If he loses it, whoever is covering has to make a tackle (`Tackling`): your other defender, or with one defender and two midfielders the midfielder who tackles best. With nobody covering, the forward is through on your keeper. A flick-on from a long ball is run on to the same way.

- Play high with quick defenders, or against slow forwards and sides that want to pass through you.
- Play deep against quick forwards, or when your defenders are slow.
- Unlike earlier seasons, your defenders are judged on the same attributes at any height. What changes is how often they are asked to win a race.

### Set-piece taker

Name one player to take your free kicks, penalties and corners. Leave it blank and your best taker steps up for each free kick and each penalty, and corners are taken by one of your better crossers.

| Chance | What the taker does | Attributes that matter |
|---|---|---|
| Free kick | Takes the shot | `Technique`, then `ShotPower` |
| Penalty | Takes the shot | `Composure`, then `Finishing` |
| Corner | Delivers it: someone else goes up for it | `Crossing` |

A good corner makes the header easier to win: the delivery helps your player in the air against the defender marking him. Who goes up for it follows `Heading` and `Strength`, and a Target Man is found twice as often. The taker can't head his own corner.

- Naming a taker is not automatically an upgrade: he takes all three. Name one if the same player is your best, or near it, at free kicks and penalties and can cross.
- Don't name your best header. Left blank he may still take the odd corner if he also crosses well; naming someone else keeps him in the box for all of them.
- A goal down from the 82nd minute, your keeper comes up for corners.

### Player roles

| Role | What it does | Best for |
|---|---|---|
| **Captain** | A small boost to the whole team and to his own play, both scaled by his quality: `(ControlRating + Composure) / 2` (`GoalkeeperRating` instead for keepers). Above 60 he lifts both; below it he drags both down. | A composed, intelligent senior — one per team |
| **Target Man** | The one your team looks for in the air: aimed at twice as often with crosses, corners and long balls from the back. It only decides who is aimed at. Winning it takes `Heading` and `Strength`, and a long ball is mostly `Strength`. | A strong forward who is good in the air, most of all in a team with no midfield, which goes long more than any other |
| **Playmaker** | Plays nearly twice as many of your passes forward as the players beside him. That helps if he is your best passer and costs you if he isn't. Does little for a midfielder on his own in midfield: he is playing most of them anyway. | Your best `Passing` midfielder when you have two (or defender, when you have no midfield) |
| **Ball Winner** | Goes to the ball. When his line is outnumbered (one defender against two forwards, say) he makes up half of what it is short. The price is fouls: he gives one away about a third more often when he wins a tackle. | A high-`Tackling` defender or midfielder in a line that is outnumbered. Good tacklers foul less to begin with; a poor tackler with this tag is a liability |

---

## Building a team — three concrete examples

### "Counter-attacking" team

Style: Pyramid formation, deep line, slow tempo, low press.

Recruit:
- Goalkeeper with high `GoalkeeperRating` and decent `SpeedRating` (saves through-balls).
- Two defenders with high `DefenseRating` and `Tackling` (positional, not fast — deep line means they don't need raw pace).
- One midfielder with high `Technique` and `ControlRating` (long passes for counters).
- Striker with high `SpeedRating`, `Finishing`, and `Composure` (1-on-1s and breakaways are your bread and butter; clinical conversion).

Why it works: low press + deep line = solid defense, lots of misses by the opponent. Slow tempo = you don't waste your few chances. High-`Technique` long-range and high-`SpeedRating` 1-on-1 breakaways are your goal sources.

### "Possession" team

Style: Diamond formation, normal line, normal tempo, medium press.

Recruit:
- Two well-rounded midfielders, one tagged as **Playmaker** (high `ControlRating`).
- Defenders with balanced `DefenseRating` / `Tackling` / `SpeedRating` — Diamond defends well.
- Striker with high `AttackRating` and `Finishing` for open play.

Why it works: the Diamond's possession bonus (3%) compounds over the match — you'll see 60-65% of chances. Most of your goals come from open play, so finishing matters more than specialist attributes.

### "Aerial bombardment" team

Style: Box formation (or Y), high tempo, high line, target-man specialist.

Recruit:
- Striker with **maxed `Heading`** tagged as **Target Man** (gets ~2× selection on corners and crosses).
- Set-piece taker with high `Technique` (delivers great corners).
- Fast defenders with high `SpeedRating` (high line means they need to chase).
- Captain on your most experienced player.

Why it works: high tempo creates more chances; many of those will be crosses and corners, where your aerial specialist dominates. The high line is risky but creates more counter-attack opportunities. Heading + Technique combo is a goal machine.

---

## Common pitfalls

**Don't recruit on `OverallRating` alone.** A player with `OverallRating` 85 split across `AttackRating` 90, `SpeedRating` 80, no `Heading` is a different player from one with `AttackRating` 80, `SpeedRating` 75, `Heading` 95. The first is your open-play striker; the second is your corner specialist. They suit different lineups.

**Don't pair tactics with the wrong builds.** A high line with slow defenders is a goal-conceding machine. A high press with low-`WorkRate` midfielders fizzles by the 60th minute. Match your tactics to the players you have.

**Don't ignore the goalkeeper.** Saves are a huge fraction of your defensive total. A 90-rated keeper vs a 75-rated keeper is the difference between conceding 1 and conceding 2 in most matches.

**Don't forget injury risk.** Box formation + high press + injury-prone players is a recipe for a bench full of players you can't field. Spread the load.

**Don't expect a single tactic to fix a weak squad.** The skill curve makes top players genuinely better; tactics shift the trade-offs but don't manufacture quality. A perfectly-tactically-set Wales still beats England maybe 1-2 games in 10.

---

## Verifying a match

Every match is reproducible. The engine takes a seed (derived from an Algorand block hash for verifiability) and produces the same events every time. If you want to re-verify a historical match, the `algorand` package can re-derive the seed from the block round, and feeding it back through `RunGameWithSeed` produces the identical event sequence. There's no hidden randomness, no server-side bias.
