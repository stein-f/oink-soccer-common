package soccer

type TeamType string

const (
	TeamTypeHome TeamType = "Home"
	TeamTypeAway TeamType = "Away"
)

type PlayerPosition string

const (
	PlayerPositionGoalkeeper PlayerPosition = "Goalkeeper"
	PlayerPositionDefense    PlayerPosition = "Defense"
	PlayerPositionMidfield   PlayerPosition = "Midfield"
	PlayerPositionAttack     PlayerPosition = "Attack"
	PlayerPositionAny        PlayerPosition = "Any"
)

type PlayerLevel string

const (
	PlayerLevelLegendary        PlayerLevel = "Legendary"
	PlayerLevelWorldClass       PlayerLevel = "World class"
	PlayerLevelProfessional     PlayerLevel = "Professional"
	PlayerLevelSemiProfessional PlayerLevel = "Semi Professional"
	PlayerLevelAmateur          PlayerLevel = "Amateur"
)

type FormationType string

const (
	FormationTypePyramid FormationType = "The Pyramid"
	FormationTypeDiamond FormationType = "The Diamond"
	FormationTypeY       FormationType = "The Y"
	FormationTypeBox     FormationType = "The Box"
)

type GameEventType string

const (
	GameEventTypeGoal GameEventType = "Goal"
	GameEventTypeMiss GameEventType = "Miss"
)

type BoostType string

const (
	BoostTypeTeam     BoostType = "Team Boost"
	BoostTypePlayer   BoostType = "Player Boost"
	BoostTypePosition BoostType = "Position Boost"
)

type GameOutcomeType string

const (
	GameOutcomeTypeWon   GameOutcomeType = "Won"
	GameOutcomeTypeLost  GameOutcomeType = "Lost"
	GameOutcomeTypeDrawn GameOutcomeType = "Drawn"
)

type ChanceType string

// ChanceType values are emitted on GameEvent in v2 so downstream UIs can
// render richer commentary. They were declared in v1 but never populated.
const (
	ChanceTypeCorner         ChanceType = "Corner"
	ChanceTypeCross          ChanceType = "Cross"
	ChanceTypeOpenPlay       ChanceType = "Open Play"
	ChanceTypeGoalKeeperShot ChanceType = "Goalkeeper Shot"
	ChanceTypeLongRange      ChanceType = "Long Range"
	ChanceTypeFreeKick       ChanceType = "Free Kick"
	ChanceTypePenalty        ChanceType = "Penalty"
)

type InjurySeverity string

const (
	InjurySeverityLow  InjurySeverity = "Low Severity"
	InjurySeverityMid  InjurySeverity = "Mid Severity"
	InjurySeverityHigh InjurySeverity = "High Severity"
)

type PlayerTag string

const (
	PlayerTagInjuryProne PlayerTag = "Injury Prone"
)
