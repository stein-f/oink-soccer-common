package soccer

import (
	"fmt"
	"github.com/mroth/weightedrand"
	"math/rand"
	"sort"
)

const (
	MinGameMinute            = 1
	MaxGameMinute            = 98
	OutOfPositionScaleFactor = 0.85
)

type SelectedPlayer struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Attributes       PlayerAttributes `json:"attributes"`
	SelectedPosition PlayerPosition   `json:"position"`
}

func (p SelectedPlayer) IsOutOfPosition() bool {
	if p.Attributes.Position == PlayerPositionAny {
		return false
	}
	return p.SelectedPosition != p.Attributes.Position
}

func (p SelectedPlayer) GetControlScore() int {
	if p.IsOutOfPosition() {
		return int(float64(p.Attributes.GetControlScore()) * OutOfPositionScaleFactor)
	}
	return p.Attributes.GetControlScore()
}

func (p SelectedPlayer) GetAttackScore() int {
	if p.IsOutOfPosition() {
		return int(float64(p.Attributes.GetAttackScore()) * OutOfPositionScaleFactor)
	}
	return p.Attributes.GetAttackScore()
}

func (p SelectedPlayer) GetDefenseScore() int {
	if p.IsOutOfPosition() {
		return int(float64(p.Attributes.GetDefenseScore()) * OutOfPositionScaleFactor)
	}
	return p.Attributes.GetDefenseScore()
}

type GameLineup struct {
	Team    Team             `json:"team"`
	Players []SelectedPlayer `json:"players"`
}

func (l GameLineup) FindPlayer(id string) (SelectedPlayer, bool) {
	for _, player := range l.Players {
		if player.ID == id {
			return player, true
		}
	}
	return SelectedPlayer{}, false
}

type GameEvent struct {
	Type   GameEventType `json:"type"`
	Event  any           `json:"event"` // GoalEvent, MissEvent
	Minute int           `json:"minute"`
}

func (g GameEvent) GetTypeType() TeamType {
	if g.IsGoal() {
		return g.GetGoalEvent().TeamType
	}
	return g.GetMissEvent().TeamType
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

func RunGame(homeTeam GameLineup, awayTeam GameLineup) ([]GameEvent, error) {
	events := []GameEvent{}
	teamChances, err := DetermineTeamChances(homeTeam.Players, awayTeam.Players)
	if err != nil {
		return nil, err
	}
	eventMinutes := getRandomMinutes(len(teamChances))
	for i, teamChance := range teamChances {
		minuteOfEvent := eventMinutes[i]
		event, err := runTeamChance(teamChance, homeTeam, awayTeam, minuteOfEvent)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func runTeamChance(attackingTeamType TeamType, homeTeamLineup GameLineup, awayTeamLineup GameLineup, minuteOfEvent int) (GameEvent, error) {
	attackingTeamLineup := homeTeamLineup
	attackBoost := getAttackBoost(homeTeamLineup)

	defensiveTeamLineup := awayTeamLineup
	defenseBoost := getDefenseBoost(awayTeamLineup)

	if attackingTeamType == TeamTypeAway {
		attackingTeamLineup = awayTeamLineup
		attackBoost = getAttackBoost(awayTeamLineup)

		defensiveTeamLineup = homeTeamLineup
		defenseBoost = getDefenseBoost(homeTeamLineup)
	}

	positionOfAttackPlayer, err := determinePositionOfAttackingTeamChance(attackingTeamLineup)
	if err != nil {
		return GameEvent{}, err
	}

	attackPlayer, err := getRandomPlayerByPosition(positionOfAttackPlayer, attackingTeamLineup.Players)
	if err != nil {
		return GameEvent{}, err
	}

	scaledDefenseScore := ScalingFunction(applyBoost(defenseBoost, CalculateTeamDefenseScore(defensiveTeamLineup.Players)))
	scaledAttackScore := ScalingFunction(applyBoost(attackBoost, attackPlayer.GetAttackScore()))

	goalChanceChoices := []weightedrand.Choice{
		{Item: true, Weight: uint(scaledAttackScore)},
		{Item: false, Weight: uint(scaledDefenseScore)},
	}
	resultChooser, err := weightedrand.NewChooser(goalChanceChoices...)
	if err != nil {
		return GameEvent{}, fmt.Errorf("failed to create result chooser. %w", err)
	}

	isGoal := resultChooser.Pick().(bool)
	if isGoal {
		return GameEvent{
			Type: GameEventTypeGoal,
			Event: GoalEvent{
				PlayerID: attackPlayer.ID,
				TeamType: attackingTeamType,
			},
			Minute: minuteOfEvent,
		}, nil
	}

	return GameEvent{
		Type: GameEventTypeMiss,
		Event: MissEvent{
			PlayerID: attackPlayer.ID,
			TeamType: attackingTeamType,
		},
		Minute: minuteOfEvent,
	}, nil
}

func getAttackBoost(lineup GameLineup) float64 {
	formationConfig := getFormationConfig(lineup.Team.Formation)
	return formationConfig.AttackModifier
}

func getDefenseBoost(lineup GameLineup) float64 {
	formationConfig := getFormationConfig(lineup.Team.Formation)
	return formationConfig.DefenseModifier
}

func applyBoost(boost float64, score int) int {
	return int(float64(score) * boost)
}

func getFormationConfig(formationType FormationType) FormationConfig {
	switch formationType {
	case FormationTypePyramid:
		return ThePyramidFormation
	case FormationTypeY:
		return TheYFormation
	default:
		return TheDiamondFormation
	}
}

// determines the position of the player that will have the chance to score a goal.
// weighted rand based on player position: 60% attack, 30% midfield, 10% defense
func determinePositionOfAttackingTeamChance(attackingTeamLineup GameLineup) (PlayerPosition, error) {
	playerChoices := []weightedrand.Choice{}
	for i := range attackingTeamLineup.Players {
		var weight uint
		switch attackingTeamLineup.Players[i].SelectedPosition {
		case PlayerPositionDefense:
			weight = uint(10)
		case PlayerPositionMidfield:
			weight = uint(30)
		default:
			weight = uint(60)
		}
		playerChoices = append(playerChoices, weightedrand.Choice{
			Item:   attackingTeamLineup.Players[i],
			Weight: weight,
		})
	}
	chooser, err := weightedrand.NewChooser(playerChoices...)
	if err != nil {
		return "", fmt.Errorf("failed to create player chooser. %w", err)
	}
	return chooser.Pick().(SelectedPlayer).SelectedPosition, nil
}

func getRandomPlayerByPosition(position PlayerPosition, players []SelectedPlayer) (SelectedPlayer, error) {
	var playersByPosition []SelectedPlayer
	for _, player := range players {
		if player.SelectedPosition == position {
			playersByPosition = append(playersByPosition, player)
		}
	}
	return playersByPosition[rand.Intn(len(playersByPosition))], nil
}

// DetermineTeamChances determines the chances of each team to score a goal. It is based on the control score of each team.
func DetermineTeamChances(
	homeTeamPlayers []SelectedPlayer,
	awayTeamPlayers []SelectedPlayer,
) ([]TeamType, error) {
	eventCount, err := getEventCount()
	if err != nil {
		return nil, err
	}

	homeTeamControlScore := ScalingFunction(CalculateTeamControlScore(homeTeamPlayers))
	awayTeamControlScore := ScalingFunction(CalculateTeamControlScore(awayTeamPlayers))

	choices := []weightedrand.Choice{
		{Item: TeamTypeHome, Weight: uint(homeTeamControlScore)},
		{Item: TeamTypeAway, Weight: uint(awayTeamControlScore)},
	}
	chooser, err := weightedrand.NewChooser(choices...)
	if err != nil {
		return nil, fmt.Errorf("failed to create team chances chooser. %w", err)
	}

	var teamChances []TeamType
	for i := 0; i < eventCount; i++ {
		teamType := chooser.Pick().(TeamType)
		teamChances = append(teamChances, teamType)
	}

	return teamChances, nil
}

func getEventCount() (int, error) {
	chooser, err := weightedrand.NewChooser(eventCountWeights...)
	if err != nil {
		return 0, fmt.Errorf("failed to create event count chooser. %w", err)
	}
	return chooser.Pick().(int), nil
}

func getRandomMinutes(count int) []int {
	var minutes []int
	for i := 0; i < count; i++ {
		minutes = append(minutes, rand.Intn(MaxGameMinute-MinGameMinute+1)+MinGameMinute)
	}
	sort.Slice(minutes, func(i, j int) bool {
		return minutes[i] < minutes[j]
	})
	return minutes
}

func CreateGameStats(events []GameEvent) GameStats {
	homeTeamStats := TeamStats{TeamType: TeamTypeHome}
	awayTeamStats := TeamStats{TeamType: TeamTypeAway}

	for _, event := range events {
		switch event.Type {
		case GameEventTypeGoal:
			goalEvent := event.Event.(GoalEvent)
			if goalEvent.TeamType == TeamTypeHome {
				homeTeamStats.Goals++
				homeTeamStats.Shots++
			} else {
				awayTeamStats.Goals++
				awayTeamStats.Shots++
			}
		case GameEventTypeMiss:
			missEvent := event.Event.(MissEvent)
			if missEvent.TeamType == TeamTypeHome {
				homeTeamStats.Shots++
			} else {
				awayTeamStats.Shots++
			}
		}
	}

	return GameStats{
		HomeTeamStats: homeTeamStats,
		AwayTeamStats: awayTeamStats,
	}
}

var eventCountWeights = []weightedrand.Choice{
	{Item: 3, Weight: 2},
	{Item: 4, Weight: 3},
	{Item: 5, Weight: 5},
	{Item: 6, Weight: 8},
	{Item: 7, Weight: 8},
	{Item: 8, Weight: 5},
	{Item: 9, Weight: 3},
	{Item: 10, Weight: 2},
	{Item: 11, Weight: 1},
	{Item: 12, Weight: 1},
}
