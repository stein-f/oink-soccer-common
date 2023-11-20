# oink-soccer-common

The purpose of this repository is to publish the algorithm used to determine the outcome of an oink soccer match.

## Run the example

You must have Go installed to run the example. https://go.dev/doc/install

```shell
go run cmd/simulate/main.go
```

The following will output to the console.

```text
Games played: 10000
StrongTeam wins: 9021
StrongTeam chances/game: 4.442100
WeakTeam wins: 284
WeakTeam chances/game: 2.078500
Draws: 695
Goals/game: 3.928200
Attacker goals: 25642 (65.276717%)
Midfielder goals: 12903 (32.847106%)
Defender goals: 631 (1.606334%)
Goalkeeper goals: 106 (0.269844%)
```

## algorithm

1. Choose the number of events (goal|miss) in the game. It is a weighted random number between 1 and 12.
2. For each event, determine which team is the attacking and defensive team. It is a weighted random choice based on the team's `control score`.
3. Determine which attacking player will have the team chance. It is a weighted random choice based on the player's position and control score. Attackers are more likely to get the chance than midfielders, who are more likely to get the chance than defenders.
4. Determine the event outcome. It is a weighted random choice based on the player's `attack score`. The higher the attack score, the more likely the player is to score a goal. This is offset by defending team's overall `defense score`.

## Control Score

The player control score is a weighted sum of the `control` and `speed` attributes:

```text
playerControlScore = (controlRating * 3 + speedRating) / 4
```

The team control score is the sum of the player control scores, weighted by position as follows:

- Goalkeeper: 5%
- Defense: 15%
- Midfield: 65%
- Attack: 15%

The average score is taken for a position where there are multiple players in that position.

## Defense Score

The team defense score is the sum of the player defense scores, weighted by position as follows:

- Goalkeeper: 35%
- Defense: 40%
- Midfield: 20%
- Attack: 5%

The average score is taken for a position where there are multiple players in that position. The individual defense score of a player is a function of the defense and speed attributes as follows:

```text
playerDefenseScore = (defenseRating * 3 + speedRating) / 4
```

## Attack Score

Attack score works slightly differently to defense and control scores (which are weighted averages of the overall team capabilities). A random player is chosen for the scoring chance, which is a weighted random choice based on the player position. Attackers are more likely to get the chance than midfielders, who are more likely to get the chance than defenders.

The selected player's attack score is then used to determine the event outcome. The score is a weighted sum of the `attack` and `speed` attributes:

```text
playerAttackScore = (attackRating * 3 + speedRating) / 4
```

## Formations

Formations apply boosts and penalties to the team's attack and defense scores as follows.

### The Pyramid (2-1-1)

Defensive formation

- 10% defense boost
- 10% attack penalty

```
  5
  4
2   3
  1
```

### The Diamond (1-2-1)

Balanced formation

- no boosts/penalties

```
  5
3   4
  2
  1
```

### The Y (1-1-2)

Attacking formation

- 10% defense penalty
- 10% attack boost

```
4   5
  3
  2
  1
```
