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

## Verify a game

The random number generator used in the engine is seeded with a round from the blockchain. This means that the outcome of a game can be verified by running the engine with the same seed.
Running the simulation with the same seed will produce the same outcome.

Edit the game key in `cmd/verify/main.go` to verify a the game of your choice. The game key is the last path segment of the game's highlights URL. For example, the highlights URL `https://www.thelostpigs.com/oink-soccer/player/TC0yLTYtMS05` has a game key of `TC0yLTYtMS05`.

Then run the following command:

```shell
go run cmd/verify/main.go
```

which will print the result to the console.

```text
Round: 35323350, Block hash: TMUTUFAKGCDT4VHG2QJCIRR26ATBIWDOKDGDLDEQTGLAY34ZKDTA
Team1 FC 2 - 2 Team2 FC
```

## Run player allocation

The player allocation algorithm is used to allocate players to assets. We use the block hash of a round from the Algorand blockchain to seed the random number generator. This means that the player allocation can be verified by running the engine with the same seed (block hash) to produce repeatable results. It is a tamper proof way to allocate players to teams that can be verified by anyone, ensuring that the allocation is fair, open, verifiable and transparent.

```shell
go run cmd/allocation/runner/main.go
```

This will generate a csv file with the allocations in `cmd/allocation/s3/out/assigned_players.csv`. You can search the file using grep as follows:

```shell
grep Salah cmd/allocation/s3/out/assigned_players.csv
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
