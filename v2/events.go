package soccer

type GameEvent struct {
	Type   GameEventType `json:"type"`
	Event  any           `json:"event"` // GoalEvent | MissEvent
	Minute int           `json:"minute"`
	// ChanceType is new in v2. Empty string for events that pre-date the field.
	ChanceType ChanceType `json:"chance_type,omitempty"`
	// Steps is how the event came about, contest by contest, in order. Only
	// ModelSteps (season 19+) fills it.
	Steps []Step `json:"steps,omitempty"`
}

// StepKind names one contest inside an attack.
type StepKind string

const (
	StepKindPass   StepKind = "pass"    // a ball forward, against whoever tries to cut it out
	StepKindTakeOn StepKind = "take_on" // a dribble at a defender
	StepKindRun    StepKind = "run"     // a race in behind the last defender
	StepKindCross  StepKind = "cross"   // a ball into the box, against the man closing it down
	StepKindAerial StepKind = "aerial"  // two players going for the same ball in the air
	StepKindFoul   StepKind = "foul"    // the defender won it unfairly
	// StepKindRebound is a loose ball off the keeper or the woodwork, and
	// who got to it. It opens the chance that follows.
	StepKindRebound StepKind = "rebound"
	StepKindShot    StepKind = "shot" // the shot, against a defender trying to block it
	StepKindSave    StepKind = "save" // the shot on target, against the keeper
)

// Step is one contest between two players. ActorID is the player on the
// attacking side of it (the keeper's save is the one exception: the actor is
// still the shooter, and Won means he scored). OpponentID is empty when
// nobody contested it.
//
// Detail narrows the kind:
//   - pass:  short | long | through
//   - shot:  blocked | wide | over | post | bar | on_target
//   - save:  goal | held | parried
//   - foul:  free_kick | penalty
//   - take_on: solo (a run from deep, past a midfielder first) | cut_in (a
//     wide player coming inside to shoot himself)
//   - rebound: save | post | bar (what it came back off)
type Step struct {
	Kind       StepKind `json:"kind"`
	ActorID    string   `json:"actor_id"`
	OpponentID string   `json:"opponent_id,omitempty"`
	Won        bool     `json:"won"`
	Detail     string   `json:"detail,omitempty"`
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
	// AssistID is the team-mate whose pass or cross made the goal. Only
	// ModelSteps fills it; empty for a goal nobody set up.
	AssistID string `json:"assist_id,omitempty"`
	// Detail is what kind of goal it was, when it was one to remember:
	// worldie (from distance, struck with real power), solo (a run from deep
	// past two or more), rebound (first to a loose ball) or keeper (the
	// goalkeeper, up for a late corner). Only ModelSteps fills it.
	Detail string `json:"detail,omitempty"`
}

type MissEvent struct {
	PlayerID string   `json:"player_id"`
	TeamType TeamType `json:"team_type"`
	// Detail is how the chance ended: blocked, wide, over, post, bar, held
	// or parried. StopperID is the defender who blocked it or the keeper who
	// saved it. Only ModelSteps fills them.
	Detail    string `json:"detail,omitempty"`
	StopperID string `json:"stopper_id,omitempty"`
}

// StopEvent is an attack that ended before a shot: a tackle, or a forward
// caught by the last defender. PlayerID is the defender who made the stop
// and TeamType his side; AgainstID is the attacker he stopped. Only
// ModelSteps emits it, and only for a few notable stops a match (the rest
// are counted in MatchOutcome.Totals).
type StopEvent struct {
	PlayerID  string   `json:"player_id"`
	TeamType  TeamType `json:"team_type"`
	AgainstID string   `json:"against_id"`
	Detail    string   `json:"detail"` // tackle | caught
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
