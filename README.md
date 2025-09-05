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


```text
playerAttackScore = (attackRating * 5 + physicalRating) / 4
```
