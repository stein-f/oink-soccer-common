package fifacsv_test

import (
	"encoding/json"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
	"github.com/stein-f/oink-soccer-common/v2/allocation/fifacsv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepSource_Apply(t *testing.T) {
	src := fifacsv.StepSource{
		Dribbling: 92, SkillDribbling: 92, Agility: 93, Balance: 82, // 359 / 4
		Passing: 80, ShortPassing: 87, Vision: 83, LongPassing: 74, // 324 / 4
		Crossing: 74, Curve: 80,
		Acceleration: 97, SprintSpeed: 96,
		Strength: 77, Jumping: 90,
		Interceptions: 19, MarkingAwareness: 34,
		AttackPositioning: 91,
		ShotPower:         92, LongShots: 86,
		GKReflexes: 6, GKDiving: 13, GKHandling: 5, GKPositioning: 11, // no goalkeeping_speed
	}
	// Everything a player already carries is left as it is.
	before := soccer.PlayerAttributes{
		GoalkeeperRating: 5, DefenseRating: 29, SpeedRating: 96, ControlRating: 80, AttackRating: 91,
		AggressionRating: 61, OverallRating: 91, WorkRate: 91, Finishing: 95, Heading: 80,
		Technique: 78, Composure: 88, Tackling: 24, PrimaryPosition: soccer.PlayerPositionAttack,
	}
	got := before

	src.Apply(&got)

	want := before
	want.Dribbling, want.Passing, want.Crossing = 89, 81, 77
	want.Pace, want.Strength, want.Interceptions = 96, 83, 19
	want.Positioning, want.Marking, want.ShotPower = 91, 34, 89
	want.Reflexes, want.Handling, want.OneOnOne = 9, 5, 11
	assert.Equal(t, want, got)
}

func TestStepSource_Apply_Keeper(t *testing.T) {
	var attrs soccer.PlayerAttributes

	fifacsv.StepSource{GKReflexes: 90, GKDiving: 87, GKHandling: 89, GKSpeed: 46, GKPositioning: 90}.Apply(&attrs)

	assert.Equal(t, 88, attrs.Reflexes)
	assert.Equal(t, 89, attrs.Handling)
	assert.Equal(t, 75, attrs.OneOnOne, "positioning counts twice over speed: (46 + 2*90) / 3")
}

func TestStepSource_Apply_EmptyRowLeavesFallbacks(t *testing.T) {
	// Legend cards and hand-made rows carry no detailed columns. Their
	// attributes stay zero, so the engine falls back to the core ratings.
	attrs := soccer.PlayerAttributes{ControlRating: 88, DefenseRating: 40, AttackRating: 90, GoalkeeperRating: 12, SpeedRating: 85}

	fifacsv.StepSource{}.Apply(&attrs)

	assert.Zero(t, attrs.Dribbling)
	assert.Equal(t, 88, attrs.EffectiveDribbling())
	assert.Equal(t, 85, attrs.EffectivePace())
	assert.Equal(t, 12, attrs.EffectiveReflexes())
}

// TestFC27Dataset_StepAttributes guards the player-on-player attributes of
// the committed season-19 dataset: every real player has them, and nothing
// about them depends on the draw.
func TestFC27Dataset_StepAttributes(t *testing.T) {
	path := filepath.Join(repoRoot, fifacsv.FC27Dataset)
	cands, err := fifacsv.LoadCandidates(path, rand.New(rand.NewSource(1)))
	require.NoError(t, err)

	byID := map[string]soccer.PlayerAttributes{}
	for _, c := range cands {
		a := c.Attributes
		byID[c.ID] = a
		if strings.HasPrefix(c.ID, allocation.LegendCardPrefix) {
			continue
		}
		outfield := map[string]int{
			"dribbling": a.Dribbling, "passing": a.Passing, "crossing": a.Crossing, "pace": a.Pace,
			"strength": a.Strength, "interceptions": a.Interceptions, "positioning": a.Positioning,
			"marking": a.Marking, "shot power": a.ShotPower,
		}
		for name, v := range outfield {
			require.True(t, v >= 1 && v <= 99, "player %s %s = %d", c.ID, name, v)
		}
		require.Equal(t, a.GoalkeeperRating, a.Handling, "player %s: handling is the goalkeeper rating's own column", c.ID)
		if a.PrimaryPosition == soccer.PlayerPositionGoalkeeper {
			// A keeper's sub-skills sit around his rating; an outfielder's
			// are the dataset's token 5-20.
			require.True(t, a.Reflexes >= a.OverallRating-25 && a.OneOnOne >= 30,
				"keeper %s (overall %d) reflexes %d one-on-one %d", c.ID, a.OverallRating, a.Reflexes, a.OneOnOne)
		} else {
			require.True(t, a.Reflexes >= 1 && a.Reflexes <= 40 && a.OneOnOne >= 1, "outfielder %s reflexes %d one-on-one %d", c.ID, a.Reflexes, a.OneOnOne)
		}
	}

	mbappe, courtois := byID["231747"], byID["192119"]
	assert.Greater(t, mbappe.Pace, 90)
	assert.Greater(t, mbappe.Dribbling, 85)
	assert.Less(t, mbappe.Marking, 50)
	assert.Greater(t, courtois.Reflexes, 85)
	assert.Less(t, courtois.Dribbling, 50, "a keeper's dribbling comes from the detailed columns, not the keeper facets")

	// A different seed changes only the 5-30 filler for blank core ratings.
	other, err := fifacsv.LoadCandidates(path, rand.New(rand.NewSource(2)))
	require.NoError(t, err)
	for _, c := range other {
		a, b := byID[c.ID], c.Attributes
		a.GoalkeeperRating, a.DefenseRating, a.ControlRating, a.AttackRating, a.AggressionRating, a.Handling = 0, 0, 0, 0, 0, 0
		b.GoalkeeperRating, b.DefenseRating, b.ControlRating, b.AttackRating, b.AggressionRating, b.Handling = 0, 0, 0, 0, 0, 0
		require.Equal(t, a, b, "player %s", c.ID)
	}
}

func TestStepAttributes_AbsentFromOlderPlayersJSON(t *testing.T) {
	// A player stored before season 19 must serialise exactly as it did, so
	// archived line-ups and API responses for earlier seasons don't change.
	raw, err := json.Marshal(soccer.PlayerAttributes{ControlRating: 80, Finishing: 70})
	require.NoError(t, err)
	for _, key := range []string{"dribbling", "passing", "crossing", "pace", "strength", "interceptions", "positioning", "marking", "shot_power", "gk_reflexes", "gk_handling", "gk_one_on_one"} {
		assert.NotContains(t, string(raw), `"`+key+`"`)
	}
}
