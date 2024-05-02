package soccer

type TeamType string

const (
	TeamTypeHome TeamType = "Home"
	TeamTypeAway TeamType = "Away"
)

func TeamTypeValues() []TeamType {
	return []TeamType{
		TeamTypeHome,
		TeamTypeAway,
	}
}

type PlayerLevel string

const (
	PlayerLevelLegendary        PlayerLevel = "Legendary"
	PlayerLevelWorldClass       PlayerLevel = "World class"
	PlayerLevelProfessional     PlayerLevel = "Professional"
	PlayerLevelSemiProfessional PlayerLevel = "Semi Professional"
	PlayerLevelAmateur          PlayerLevel = "Amateur"
)

type PlayerPosition string

const (
	PlayerPositionGoalkeeper PlayerPosition = "Goalkeeper"
	PlayerPositionDefense    PlayerPosition = "Defense"
	PlayerPositionMidfield   PlayerPosition = "Midfield"
	PlayerPositionAttack     PlayerPosition = "Attack"
	PlayerPositionAny        PlayerPosition = "Any"
)

type FormationType string

const (
	FormationTypePyramid FormationType = "The Pyramid"
	FormationTypeDiamond FormationType = "The Diamond"
	FormationTypeY       FormationType = "The Y"
)

func FormationTypeValues() []FormationType {
	return []FormationType{
		FormationTypePyramid,
		FormationTypeDiamond,
		FormationTypeY,
	}
}

type GameEventType string

const (
	GameEventTypeGoal GameEventType = "Goal"
	GameEventTypeMiss GameEventType = "Miss"
)

func GameEventTypeValues() []GameEventType {
	return []GameEventType{
		GameEventTypeGoal,
		GameEventTypeMiss,
	}
}

type BoostType string

const (
	BoostTypeTeam     BoostType = "Team Boost"
	BoostTypePlayer   BoostType = "Player Boost"
	BoostTypePosition BoostType = "Position Boost" // deprecated, use team and player boosts instead
)

func BoostTypeValues() []BoostType {
	return []BoostType{
		BoostTypeTeam,
		BoostTypePosition,
	}
}

type GameOutcomeType string

const (
	GameOutcomeTypeWon   GameOutcomeType = "Won"
	GameOutcomeTypeLost  GameOutcomeType = "Lost"
	GameOutcomeTypeDrawn GameOutcomeType = "Drawn"
)

func GameOutcomeTypeValues() []GameOutcomeType {
	return []GameOutcomeType{
		GameOutcomeTypeWon,
		GameOutcomeTypeLost,
		GameOutcomeTypeDrawn,
	}
}

type InjurySeverity string

const (
	InjurySeverityLow  InjurySeverity = "Low Severity"
	InjurySeverityMid  InjurySeverity = "Mid Severity"
	InjurySeverityHigh InjurySeverity = "High Severity"
)
