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
	BoostTypePosition BoostType = "Position Boost"
)

func BoostTypeValues() []BoostType {
	return []BoostType{
		BoostTypeTeam,
		BoostTypePosition,
	}
}
