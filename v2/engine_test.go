package soccer_test

import (
	"errors"
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// strongLineup is the v2 test fixture used by engine-level tests. Mirrors
// v1's StrongTeam from testdata/fixtures.go so behaviour can be compared
// against the v1 baseline snapshots.
func strongLineup(formation soccer.FormationType) soccer.GameLineup {
	return soccer.GameLineup{
		Team: soccer.Team{ID: "team-strong", Formation: formation},
		Players: []soccer.SelectedPlayer{
			player("1", soccer.PlayerPositionGoalkeeper, 88, 33, 21, 37, 74),
			player("2", soccer.PlayerPositionDefense, 14, 90, 81, 37, 80),
			player("3", soccer.PlayerPositionMidfield, 14, 55, 85, 91, 80),
			player("4", soccer.PlayerPositionMidfield, 11, 75, 81, 71, 81),
			player("5", soccer.PlayerPositionAttack, 14, 22, 85, 93, 80),
		},
	}
}

func player(id string, pos soccer.PlayerPosition, gk, def, ctrl, atk, speed int) soccer.SelectedPlayer {
	return soccer.SelectedPlayer{
		ID: id, Name: id,
		Attributes: soccer.PlayerAttributes{
			GoalkeeperRating: gk, DefenseRating: def, ControlRating: ctrl, AttackRating: atk, SpeedRating: speed,
			PrimaryPosition: pos, Positions: []soccer.PlayerPosition{pos},
		},
		SelectedPosition: pos,
	}
}

func TestRunGameWithSeed_RejectsNilSource(t *testing.T) {
	_, _, err := soccer.RunGameWithSeed(nil, soccer.GameLineup{}, soccer.GameLineup{})
	assert.True(t, errors.Is(err, soccer.ErrNilRandSource))
}

// Determinism is the contract the engine exists to provide — same seed and
// same lineups must always yield the same events and injuries.
func TestRunGameWithSeed_IsDeterministic(t *testing.T) {
	home := strongLineup(soccer.FormationTypeDiamond)
	away := strongLineup(soccer.FormationTypeDiamond)

	for _, seed := range []int64{1, 42, 99, 12345} {
		events1, inj1, err1 := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
		require.NoError(t, err1)
		events2, inj2, err2 := soccer.RunGameWithSeed(rand.New(rand.NewSource(seed)), home, away)
		require.NoError(t, err2)

		assert.Equal(t, events1, events2, "seed %d events differ between runs", seed)
		assert.Equal(t, inj1, inj2, "seed %d injuries differ between runs", seed)
	}
}

// Stronger team should win more often than not against a weaker team. This
// is a smoke test, not a balance test — Phase 4's full balance harness is
// in match_balance_test.go.
func TestRunGameWithSeed_StrongerTeamWinsMoreOften(t *testing.T) {
	strong := strongLineup(soccer.FormationTypeDiamond)
	weak := soccer.GameLineup{
		Team: soccer.Team{ID: "team-weak", Formation: soccer.FormationTypeDiamond},
		Players: []soccer.SelectedPlayer{
			player("a", soccer.PlayerPositionGoalkeeper, 50, 20, 20, 20, 50),
			player("b", soccer.PlayerPositionDefense, 20, 50, 50, 20, 50),
			player("c", soccer.PlayerPositionMidfield, 20, 50, 50, 50, 50),
			player("d", soccer.PlayerPositionMidfield, 20, 50, 50, 50, 50),
			player("e", soccer.PlayerPositionAttack, 20, 20, 50, 50, 50),
		},
	}

	const trials = 200
	var strongWins, weakWins int
	for i := 0; i < trials; i++ {
		events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(int64(i))), strong, weak)
		require.NoError(t, err)
		stats := soccer.CreateGameStats(events)
		if stats.HomeTeamStats.Goals > stats.AwayTeamStats.Goals {
			strongWins++
		} else if stats.HomeTeamStats.Goals < stats.AwayTeamStats.Goals {
			weakWins++
		}
	}
	assert.Greater(t, strongWins, weakWins*3, "strong team must dominate weak team — got %d strong, %d weak", strongWins, weakWins)
}

// ChanceType must be populated on every event in v2. Empty would mean the
// engine isn't tagging events properly.
func TestRunGameWithSeed_AllEventsHaveChanceType(t *testing.T) {
	home := strongLineup(soccer.FormationTypeDiamond)
	away := strongLineup(soccer.FormationTypeY)
	events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(7)), home, away)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	for i, e := range events {
		assert.NotEmpty(t, e.ChanceType, "event %d has empty chance type", i)
	}
}

// Events must come back in chronological order so consumers (UI commentary,
// highlight reels) can render them without re-sorting.
func TestRunGameWithSeed_EventsAreChronological(t *testing.T) {
	home := strongLineup(soccer.FormationTypeDiamond)
	away := strongLineup(soccer.FormationTypeBox)
	events, _, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(3)), home, away)
	require.NoError(t, err)
	for i := 1; i < len(events); i++ {
		assert.LessOrEqual(t, events[i-1].Minute, events[i].Minute, "event %d is out of order", i)
	}
}

// CreateGameStats has no engine dependency — pure aggregator.
func TestCreateGameStats_AggregatesShotsAndGoals(t *testing.T) {
	events := []soccer.GameEvent{
		{Type: soccer.GameEventTypeGoal, Event: soccer.GoalEvent{TeamType: soccer.TeamTypeHome, PlayerID: "1"}, Minute: 10},
		{Type: soccer.GameEventTypeMiss, Event: soccer.MissEvent{TeamType: soccer.TeamTypeHome, PlayerID: "1"}, Minute: 20},
		{Type: soccer.GameEventTypeGoal, Event: soccer.GoalEvent{TeamType: soccer.TeamTypeAway, PlayerID: "2"}, Minute: 30},
		{Type: soccer.GameEventTypeMiss, Event: soccer.MissEvent{TeamType: soccer.TeamTypeAway, PlayerID: "2"}, Minute: 40},
		{Type: soccer.GameEventTypeMiss, Event: soccer.MissEvent{TeamType: soccer.TeamTypeAway, PlayerID: "2"}, Minute: 50},
	}

	stats := soccer.CreateGameStats(events)

	assert.Equal(t, soccer.TeamStats{TeamType: soccer.TeamTypeHome, Shots: 2, Goals: 1}, stats.HomeTeamStats)
	assert.Equal(t, soccer.TeamStats{TeamType: soccer.TeamTypeAway, Shots: 3, Goals: 1}, stats.AwayTeamStats)
}
