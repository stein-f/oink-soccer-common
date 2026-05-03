package soccer

type GameEvent struct {
	Type   GameEventType `json:"type"`
	Event  any           `json:"event"` // GoalEvent | MissEvent
	Minute int           `json:"minute"`
	// ChanceType is new in v2. Empty string for events that pre-date the field.
	ChanceType ChanceType `json:"chance_type,omitempty"`
}

func (g GameEvent) IsGoal() bool {
	return g.Type == GameEventTypeGoal
}

func (g GameEvent) GetGoalEvent() GoalEvent {
	return g.Event.(GoalEvent)
}

func (g GameEvent) GetMissEvent() MissEvent {
	return g.Event.(MissEvent)
}

type GoalEvent struct {
	PlayerID string   `json:"player_id"`
	TeamType TeamType `json:"team_type"`
}

type MissEvent struct {
	PlayerID string   `json:"player_id"`
	TeamType TeamType `json:"team_type"`
}

type GameStats struct {
	HomeTeamStats TeamStats `json:"home_team_stats"`
	AwayTeamStats TeamStats `json:"away_team_stats"`
}

type TeamStats struct {
	TeamType TeamType `json:"team_type"`
	Shots    int      `json:"shots"`
	Goals    int      `json:"goals"`
}

// CreateGameStats aggregates a slice of GameEvent into per-team shot/goal totals.
func CreateGameStats(events []GameEvent) GameStats {
	home := TeamStats{TeamType: TeamTypeHome}
	away := TeamStats{TeamType: TeamTypeAway}
	for _, e := range events {
		var team TeamType
		switch e.Type {
		case GameEventTypeGoal:
			team = e.GetGoalEvent().TeamType
		case GameEventTypeMiss:
			team = e.GetMissEvent().TeamType
		default:
			continue
		}
		switch team {
		case TeamTypeHome:
			home.Shots++
			if e.IsGoal() {
				home.Goals++
			}
		case TeamTypeAway:
			away.Shots++
			if e.IsGoal() {
				away.Goals++
			}
		}
	}
	return GameStats{HomeTeamStats: home, AwayTeamStats: away}
}
