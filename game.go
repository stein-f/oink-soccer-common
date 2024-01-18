package soccer

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/mroth/weightedrand"
)

const (
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

func (p SelectedPlayer) GetControlScore() float64 {
	if p.IsOutOfPosition() {
		return float64(p.Attributes.GetControlScore()) * OutOfPositionScaleFactor
	}
	return p.Attributes.GetControlScore()
}

func (p SelectedPlayer) GetAttackScore() float64 {
	if p.IsOutOfPosition() {
		return float64(p.Attributes.GetAttackScore()) * OutOfPositionScaleFactor
	}
	return p.Attributes.GetAttackScore()
}

func (p SelectedPlayer) GetDefenseScore() float64 {
	if p.IsOutOfPosition() {
		return float64(p.Attributes.GetDefenseScore()) * OutOfPositionScaleFactor
	}
	return p.Attributes.GetDefenseScore()
}

type GameLineup struct {
	Team       Team             `json:"team"`
	Players    []SelectedPlayer `json:"players"`
	ItemBoosts []Boost          `json:"item_boosts"`
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

func RunGameWithSeed(randSource *rand.Rand, homeTeam GameLineup, awayTeam GameLineup) ([]GameEvent, error) {
	events := []GameEvent{}

	teamChances, err := DetermineTeamChances(randSource, homeTeam, awayTeam)
	if err != nil {
		return nil, err
	}

	eventMinutes, err := GetRandomMinutes(randSource, len(teamChances))
	if err != nil {
		return nil, err
	}

	for i, teamChance := range teamChances {
		minuteOfEvent := eventMinutes[i]
		event, err := runTeamChance(randSource, teamChance, homeTeam, awayTeam, minuteOfEvent)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func RunGame(homeTeam GameLineup, awayTeam GameLineup) ([]GameEvent, error) {
	source := rand.NewSource(time.Now().UnixNano())
	randSource := rand.New(source)
	return RunGameWithSeed(randSource, homeTeam, awayTeam)
}

func getTeamItemBoost(lineup GameLineup) float64 {
	for _, boost := range lineup.ItemBoosts {
		if boost.BoostType == BoostTypeTeam {
			return boost.GetBoost()
		}
	}
	return 1
}

func getPositionItemBoost(boosts []Boost, position PlayerPosition) float64 {
	for _, boost := range boosts {
		if boost.BoostType == BoostTypePosition && boost.BoostPosition == position {
			return boost.GetBoost()
		}
	}
	return 1
}

func runTeamChance(randSource *rand.Rand, attackingTeamType TeamType, homeTeamLineup GameLineup, awayTeamLineup GameLineup, minuteOfEvent int) (GameEvent, error) {
	attackingTeamLineup := homeTeamLineup
	defensiveTeamLineup := awayTeamLineup

	// determine formation boosts - applied to the overall team scores rather than at individual level
	attackFormationBoost := getAttackFormationBoost(homeTeamLineup)
	defenseFormationBoost := getDefenseFormationBoost(awayTeamLineup)

	if attackingTeamType == TeamTypeAway {
		attackingTeamLineup = awayTeamLineup
		defensiveTeamLineup = homeTeamLineup

		attackFormationBoost = getAttackFormationBoost(awayTeamLineup)
		defenseFormationBoost = getDefenseFormationBoost(homeTeamLineup)
	}

	positionOfAttackPlayer, err := determinePositionOfAttackingTeamChance(randSource, attackingTeamLineup)
	if err != nil {
		return GameEvent{}, err
	}

	attackPlayer, err := getRandomPlayerByPosition(randSource, positionOfAttackPlayer, attackingTeamLineup.Players)
	if err != nil {
		return GameEvent{}, err
	}

	teamDefenseScore := CalculateTeamDefenseScore(defensiveTeamLineup)
	scaledDefenseScore := applyBoost(defenseFormationBoost, ScalingFunction(teamDefenseScore))

	attackingPlayerAttackScore := attackPlayer.GetAttackScore()
	scaledAttackScore := applyBoost(attackFormationBoost, ScalingFunction(attackingPlayerAttackScore))

	attackingTeamBoost := getTeamItemBoost(attackingTeamLineup)
	scaledAttackScore = applyBoost(attackingTeamBoost, scaledAttackScore)

	goalChanceChoices := []weightedrand.Choice{
		{Item: true, Weight: uint(scaledAttackScore)},
		{Item: false, Weight: uint(scaledDefenseScore)},
	}
	resultChooser, err := weightedrand.NewChooser(goalChanceChoices...)
	if err != nil {
		return GameEvent{}, fmt.Errorf("failed to create result chooser. %w", err)
	}

	isGoal := resultChooser.PickSource(randSource).(bool)
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

func getAttackFormationBoost(lineup GameLineup) float64 {
	formationConfig := getFormationConfig(lineup.Team.Formation)
	return formationConfig.AttackModifier
}

func getDefenseFormationBoost(lineup GameLineup) float64 {
	formationConfig := getFormationConfig(lineup.Team.Formation)
	return formationConfig.DefenseModifier
}

func applyBoost(boost float64, score float64) float64 {
	return score * boost
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

// determines the position of the gotPlayer that will have the chance to score a goal.
// weighted rand based on gotPlayer position: 60% attack, 30% midfield, 10% defense
func determinePositionOfAttackingTeamChance(randSource *rand.Rand, attackingTeamLineup GameLineup) (PlayerPosition, error) {
	playerChoices := []weightedrand.Choice{}
	for i := range attackingTeamLineup.Players {
		var weight uint
		switch attackingTeamLineup.Players[i].SelectedPosition {
		case PlayerPositionGoalkeeper:
			weight = uint(2)
		case PlayerPositionDefense:
			weight = uint(10)
		case PlayerPositionAny:
			weight = uint(10)
		case PlayerPositionMidfield:
			weight = uint(20)
		case PlayerPositionAttack:
			weight = uint(70)
		default:
			weight = uint(1)
		}
		playerChoices = append(playerChoices, weightedrand.Choice{
			Item:   attackingTeamLineup.Players[i],
			Weight: weight,
		})
	}
	chooser, err := weightedrand.NewChooser(playerChoices...)
	if err != nil {
		return "", fmt.Errorf("failed to create gotPlayer chooser. %w", err)
	}
	return chooser.PickSource(randSource).(SelectedPlayer).SelectedPosition, nil
}

func getRandomPlayerByPosition(randSource *rand.Rand, position PlayerPosition, players []SelectedPlayer) (SelectedPlayer, error) {
	var playersByPosition []SelectedPlayer
	for _, player := range players {
		if player.SelectedPosition == position {
			playersByPosition = append(playersByPosition, player)
		}
	}
	if len(playersByPosition) == 0 {
		return players[randSource.Intn(len(players))], nil
	}
	return playersByPosition[randSource.Intn(len(playersByPosition))], nil
}

// DetermineTeamChances determines the chances of each team to score a goal. It is based on the control score of each team.
func DetermineTeamChances(randSource *rand.Rand, homeTeamPlayers GameLineup, awayTeamPlayers GameLineup) ([]TeamType, error) {
	eventCount, err := getEventCount(randSource)
	if err != nil {
		return nil, err
	}

	homeTeamControlScore := 100 * ScalingFunction(CalculateTeamControlScore(homeTeamPlayers))
	awayTeamControlScore := 100 * ScalingFunction(CalculateTeamControlScore(awayTeamPlayers))

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
		teamType := chooser.PickSource(randSource).(TeamType)
		teamChances = append(teamChances, teamType)
	}

	return teamChances, nil
}

func getFormationControlBoost(lineup GameLineup) float64 {
	formationConfig := getFormationConfig(lineup.Team.Formation)
	return formationConfig.ControlModifier
}

func getEventCount(randSource *rand.Rand) (int, error) {
	chooser, err := weightedrand.NewChooser(eventCountWeights...)
	if err != nil {
		return 0, fmt.Errorf("failed to create event count chooser. %w", err)
	}
	return chooser.PickSource(randSource).(int), nil
}

func GetRandomMinutes(randSource *rand.Rand, count int) ([]int, error) {
	var minutes []int
	for i := 0; i < count; i++ {
		minute, err := getEventMinute(randSource)
		if err != nil {
			return nil, err
		}
		minutes = append(minutes, minute)
	}
	sort.Slice(minutes, func(i, j int) bool {
		return minutes[i] < minutes[j]
	})
	return minutes, nil
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
	{Item: 3, Weight: 1},
	{Item: 4, Weight: 2},
	{Item: 5, Weight: 3},
	{Item: 6, Weight: 5},
	{Item: 7, Weight: 5},
	{Item: 8, Weight: 3},
	{Item: 9, Weight: 2},
	{Item: 10, Weight: 1},
}

type eventMinuteRange struct {
	MinMinute int
	MaxMinute int
}

var eventMinuteWeights = []weightedrand.Choice{
	{Item: eventMinuteRange{MinMinute: 1, MaxMinute: 15}, Weight: 99},
	{Item: eventMinuteRange{MinMinute: 16, MaxMinute: 30}, Weight: 158},
	{Item: eventMinuteRange{MinMinute: 31, MaxMinute: 45}, Weight: 142},
	{Item: eventMinuteRange{MinMinute: 46, MaxMinute: 60}, Weight: 178},
	{Item: eventMinuteRange{MinMinute: 61, MaxMinute: 75}, Weight: 168},
	{Item: eventMinuteRange{MinMinute: 76, MaxMinute: 98}, Weight: 254},
}

func getEventMinute(randSource *rand.Rand) (int, error) {
	// validate max and min aren't the same, which would cause a panic in rand.Intn
	for _, eventMinuteWeight := range eventMinuteWeights {
		eventMinuteRange := eventMinuteWeight.Item.(eventMinuteRange)
		if eventMinuteRange.MinMinute == eventMinuteRange.MaxMinute {
			return 0, fmt.Errorf("event minute range is invalid. min: %d, max: %d", eventMinuteRange.MinMinute, eventMinuteRange.MaxMinute)
		}
	}

	chooser, err := weightedrand.NewChooser(eventMinuteWeights...)
	if err != nil {
		return 0, fmt.Errorf("failed to create event count chooser. %w", err)
	}
	eventRange := chooser.PickSource(randSource).(eventMinuteRange)
	return rand.Intn(eventRange.MaxMinute-eventRange.MinMinute+1) + eventRange.MinMinute, nil
}
