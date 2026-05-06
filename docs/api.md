# v2 Public API

This document is the contract v2 ships. Anything listed here must keep working without breaking changes within the v2 series; anything *not* listed is implementation detail that may change.

## Module path

```go
import soccer "github.com/stein-f/oink-soccer-common/v2"
```

v1 (`github.com/stein-f/oink-soccer-common`) and v2 can coexist in the same project — they're separate Go modules. Migrate consumers one by one.

## Entry point

```go
func RunGameWithSeed(rand *rand.Rand, home, away GameLineup) ([]GameEvent, Injuries, error)
```

Simulates a deterministic match. Same `(seed, home, away)` always returns the same output. Returns `ErrNilRandSource` if `rand` is nil.

## Types

### Lineups

```go
type Team struct {
    ID         string
    CustomName string
    Formation  FormationType
    Tactics    Tactics  // optional, zero value = neutral
}

type GameLineup struct {
    Team       Team
    Players    []SelectedPlayer
    ItemBoosts []Boost
}

type SelectedPlayer struct {
    ID               string
    Name             string
    Attributes       PlayerAttributes
    SelectedPosition PlayerPosition
    Injury           *InjuryEvent
    Role             PlayerRole       // optional, zero value = no role
}

type PlayerAttributes struct {
    GoalkeeperRating  int
    DefenseRating     int
    SpeedRating       int
    ControlRating     int
    AttackRating      int
    AggressionRating  int
    OverallRating     int
    PlayerLevel       PlayerLevel
    PrimaryPosition   PlayerPosition
    Positions         []PlayerPosition
    Tag               []string
    BasedOnPlayer     string
    BasedOnPlayerURL  string

    // Optional. Backfill from SpeedRating when zero.
    WorkRate int
}
```

### Events + stats

```go
type GameEvent struct {
    Type       GameEventType   // Goal | Miss
    Event      any             // GoalEvent | MissEvent
    Minute     int
    ChanceType ChanceType      // new in v2 — populated on every event
}

type GoalEvent struct { PlayerID string; TeamType TeamType }
type MissEvent struct { PlayerID string; TeamType TeamType }

type GameStats struct {
    HomeTeamStats TeamStats
    AwayTeamStats TeamStats
}

type TeamStats struct {
    TeamType TeamType
    Shots    int
    Goals    int
}

func CreateGameStats(events []GameEvent) GameStats
```

### Injuries

```go
type Injury struct {
    Severity       InjurySeverity
    MinDays        int
    MaxDays        int
    Name           string
    StatsReduction float64
    Description    string
    Weight         uint
}

type InjuryEvent struct {
    TeamID       string
    PlayerID     string
    Expires      time.Time   // zero in v2 — call ResolveInjuryExpiry
    DurationDays int         // new in v2 — rolled deterministically
    Injury       Injury
}

type Injuries struct {
    HomeTeamInjuries []InjuryEvent
    AwayTeamInjuries []InjuryEvent
}

func GetAllInjuries() []Injury
func ResolveInjuryExpiry(now time.Time, e InjuryEvent) time.Time
```

### Tactics + roles

All optional. Zero values are neutral; existing v1-style lineups produce sensible v2 games without touching these fields.

```go
type Tactics struct {
    Press         PressLevel    // "" | low | medium | high
    Tempo         TempoLevel    // "" | slow | normal | fast
    LineHeight    LineHeight    // "" | deep | normal | high
    SetPieceTaker string        // PlayerID — takes FK + Penalty directly; delivers Corners (Technique scales conversion, finisher still picked by Heading)
}

type PlayerRole string  // "" | captain | target_man | playmaker | ball_winner
```

### Boosts + formations

```go
const DRDecayPerApplication = 0.97  // alias of internal/tuning.BoostDecay

type Boost struct {
    BoostType     BoostType
    BoostPosition PlayerPosition
    MinBoost      float64
    MaxBoost      float64
    Note          string
    Applications  int
}

type FormationConfig struct {
    FormationType   FormationType
    Slots           map[uint64]PlayerPosition
    Profile         FormationProfile  // new in v2 — engine reads this

    // Deprecated — derived from Profile for backward JSON compatibility.
    DefenseModifier float64
    ControlModifier float64
    AttackModifier  float64
}

type FormationProfile = tuning.FormationProfile  // re-exported

var (
    ThePyramidFormation FormationConfig
    TheDiamondFormation FormationConfig
    TheYFormation       FormationConfig
    TheBoxFormation     FormationConfig
)
```

## Enums

```go
TeamType            TeamTypeHome | TeamTypeAway
PlayerPosition      PlayerPositionGoalkeeper | …Defense | …Midfield | …Attack | …Any
PlayerLevel         PlayerLevelLegendary | …WorldClass | …Professional | …SemiProfessional | …Amateur
FormationType       FormationTypePyramid | FormationTypeDiamond | FormationTypeY | FormationTypeBox
GameEventType       GameEventTypeGoal | GameEventTypeMiss
BoostType           BoostTypeTeam | BoostTypePlayer | BoostTypePosition
GameOutcomeType     GameOutcomeTypeWon | GameOutcomeTypeLost | GameOutcomeTypeDrawn
ChanceType          OpenPlay | Cross | Corner | LongRange | FreeKick | Penalty | GoalKeeperShot
InjurySeverity      Low | Mid | High
PlayerTag           InjuryProne
PressLevel          None | Low | Medium | High
TempoLevel          None | Slow | Normal | Fast
LineHeight          None | Deep | Normal | High
PlayerRole          None | Captain | TargetMan | Playmaker | BallWinner
```

## Subpackages

### `v2/algorand`

```go
client := algorand.NewClient(httpClient)
seed, err := client.FetchBlockSeed(ctx, round)
// seed.Source is a *rand.Rand; pass to RunGameWithSeed.
```

Pure helper for deriving a deterministic `*rand.Rand` from an Algorand block hash. Returns errors instead of panicking; takes a context.

### `v2/allocation`

```go
pool := allocation.NewPool(candidates, allocation.DefaultRules())
assignments, err := allocation.Allocate(rand, pool, assets)
```

Player-to-NFT allocation, used once per season. Same determinism contract as the engine.

## Removed from v1

These were unused by `lost-pigs` and have been dropped from v2:

- `RunGame` (no-seed variant)
- `DetermineTeamChances`, `DetermineChanceType`, `GetInjuries`, `GetTeamBoost`
- `CalculateTeamControlScore`, `CalculateTeamDefenseScore`, `CalculateTeamAverageAggression`
- `GetRandomMinutes`, `ApplyInjury`, `ScalingFunction`, `DiminishingMultiplier`
- `CreateRandomSourceFromAlgorandBlockHash` (replaced by `algorand.Client.FetchBlockSeed`)

If a downstream consumer needs any of these, file an issue — most can be reintroduced as thin wrappers around v2 internals.

## Migration from v1

Most v1 call sites just need the import path swapped:

```diff
- soccer "github.com/stein-f/oink-soccer-common"
+ soccer "github.com/stein-f/oink-soccer-common/v2"
```

Then handle the two intentional behavioural changes:

1. **Injury expiry**: v1 set `InjuryEvent.Expires` using `time.Now()`. v2 leaves it zero and exposes `DurationDays`. If you persisted `Expires` directly, swap to:
   ```go
   event.Expires = soccer.ResolveInjuryExpiry(time.Now(), event)
   ```
2. **ChanceType**: previously empty on every event, now populated. Treat empty as "Open Play" if you need a fallback for archived data.

That's it — no other public-API changes.
