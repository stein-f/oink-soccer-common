package soccer_test

import (
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTakePenaltyWithSeed_Deterministic(t *testing.T) {
	taker := player("taker", soccer.PlayerPositionAttack, 10, 20, 60, 85, 70)
	keeper := player("keeper", soccer.PlayerPositionGoalkeeper, 85, 30, 20, 15, 70)

	a := soccer.TakePenaltyWithSeed(rand.New(rand.NewSource(42)), taker, keeper, soccer.TeamTypeHome)
	b := soccer.TakePenaltyWithSeed(rand.New(rand.NewSource(42)), taker, keeper, soccer.TeamTypeHome)

	assert.Equal(t, a, b, "same seed must give the same outcome")
	assert.Equal(t, "taker", a.TakerID)
	assert.Equal(t, "keeper", a.KeeperID)
	assert.Equal(t, soccer.TeamTypeHome, a.TeamType)
	assert.Contains(t, []soccer.PenaltyDirection{
		soccer.PenaltyDirectionLeft, soccer.PenaltyDirectionMid, soccer.PenaltyDirectionRight,
	}, a.Direction)
	assert.Contains(t, []soccer.PenaltyResult{soccer.PenaltyResultScored, soccer.PenaltyResultMissed}, a.Result)
}

func TestTakePenaltyWithSeed_GoodTakersConvertMoreThanWeakOnes(t *testing.T) {
	eliteTaker := player("elite", soccer.PlayerPositionAttack, 10, 20, 90, 95, 80)
	eliteTaker.Attributes.Composure = 95
	weakTaker := player("weak", soccer.PlayerPositionAttack, 10, 20, 30, 35, 50)
	weakTaker.Attributes.Composure = 30
	keeper := player("keeper", soccer.PlayerPositionGoalkeeper, 80, 30, 20, 15, 70)

	const n = 4000
	eliteGoals, weakGoals := 0, 0
	for i := 0; i < n; i++ {
		r := rand.New(rand.NewSource(int64(i)))
		if soccer.TakePenaltyWithSeed(r, eliteTaker, keeper, soccer.TeamTypeHome).IsGoal() {
			eliteGoals++
		}
		r = rand.New(rand.NewSource(int64(i)))
		if soccer.TakePenaltyWithSeed(r, weakTaker, keeper, soccer.TeamTypeHome).IsGoal() {
			weakGoals++
		}
	}

	assert.Greater(t, eliteGoals, weakGoals, "an elite taker should convert more than a weak one")
	// Penalties convert at high rates: a clutch finisher should be well above half.
	assert.Greater(t, eliteGoals, n/2)
}

func TestRunShootoutWithSeed(t *testing.T) {
	home := strongLineup(soccer.FormationTypeDiamond)
	home.Team.ID = "home"
	away := strongLineup(soccer.FormationTypeDiamond)
	away.Team.ID = "away"

	t.Run("rejects nil rand source", func(t *testing.T) {
		_, err := soccer.RunShootoutWithSeed(nil, home, away)
		assert.ErrorIs(t, err, soccer.ErrNilRandSource)
	})

	t.Run("rejects empty lineups", func(t *testing.T) {
		_, err := soccer.RunShootoutWithSeed(rand.New(rand.NewSource(1)), soccer.GameLineup{}, away)
		assert.ErrorIs(t, err, soccer.ErrNoPenaltyTakers)
	})

	t.Run("is deterministic and produces a decisive, consistent result", func(t *testing.T) {
		a, err := soccer.RunShootoutWithSeed(rand.New(rand.NewSource(7)), home, away)
		require.NoError(t, err)
		b, err := soccer.RunShootoutWithSeed(rand.New(rand.NewSource(7)), home, away)
		require.NoError(t, err)

		assert.Equal(t, a, b, "same seed must give the same shootout")

		// There must be a winner and it must match the scoreline.
		assert.NotEqual(t, a.HomeScore, a.AwayScore, "shootout must be decisive")
		if a.HomeScore > a.AwayScore {
			assert.Equal(t, soccer.TeamTypeHome, a.Winner)
		} else {
			assert.Equal(t, soccer.TeamTypeAway, a.Winner)
		}

		// At least the 10 regulation kicks were taken, alternating home/away.
		assert.GreaterOrEqual(t, len(a.Kicks), 2*5)
		for i, kick := range a.Kicks {
			wantTeam := soccer.TeamTypeHome
			if i%2 == 1 {
				wantTeam = soccer.TeamTypeAway
			}
			assert.Equal(t, wantTeam, kick.TeamType, "kick %d wrong team", i)
		}

		// Scores equal the number of scored kicks per team.
		var home, awayScored int
		for _, kick := range a.Kicks {
			if kick.IsGoal() {
				if kick.TeamType == soccer.TeamTypeHome {
					home++
				} else {
					awayScored++
				}
			}
		}
		assert.Equal(t, home, a.HomeScore)
		assert.Equal(t, awayScored, a.AwayScore)
	})

	t.Run("regulation always completes all five kicks each", func(t *testing.T) {
		// Across many seeds, the kick count is at least 10 and even (paired).
		for seed := int64(0); seed < 200; seed++ {
			res, err := soccer.RunShootoutWithSeed(rand.New(rand.NewSource(seed)), home, away)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(res.Kicks), 10)
			assert.Equal(t, 0, len(res.Kicks)%2, "kicks must be paired")
		}
	})
}
