package allocation_test

import (
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Build a small varied candidate pool for tests.
func makeCandidates() []allocation.Candidate {
	var out []allocation.Candidate
	add := func(id string, pos soccer.PlayerPosition, overall, aggression int) {
		out = append(out, allocation.Candidate{
			ID:   id,
			Name: id,
			Attributes: soccer.PlayerAttributes{
				PrimaryPosition:  pos,
				OverallRating:    overall,
				AggressionRating: aggression,
			},
		})
	}
	// 3 of each (position, level) so allocation has options.
	for i := 0; i < 3; i++ {
		add("gk-leg"+string(rune('a'+i)), soccer.PlayerPositionGoalkeeper, 90, 60)
		add("gk-wc"+string(rune('a'+i)), soccer.PlayerPositionGoalkeeper, 82, 60)
		add("gk-pro"+string(rune('a'+i)), soccer.PlayerPositionGoalkeeper, 75, 60)
		add("gk-sp"+string(rune('a'+i)), soccer.PlayerPositionGoalkeeper, 60, 60)
		add("gk-am"+string(rune('a'+i)), soccer.PlayerPositionGoalkeeper, 50, 60)

		add("def-leg"+string(rune('a'+i)), soccer.PlayerPositionDefense, 90, 60)
		add("def-wc"+string(rune('a'+i)), soccer.PlayerPositionDefense, 82, 60)
		add("def-pro"+string(rune('a'+i)), soccer.PlayerPositionDefense, 75, 60)
		add("def-sp"+string(rune('a'+i)), soccer.PlayerPositionDefense, 60, 60)
		add("def-am"+string(rune('a'+i)), soccer.PlayerPositionDefense, 50, 60)

		add("mid-leg"+string(rune('a'+i)), soccer.PlayerPositionMidfield, 90, 60)
		add("mid-wc"+string(rune('a'+i)), soccer.PlayerPositionMidfield, 82, 60)
		add("mid-pro"+string(rune('a'+i)), soccer.PlayerPositionMidfield, 75, 60)
		add("mid-sp"+string(rune('a'+i)), soccer.PlayerPositionMidfield, 60, 60)
		add("mid-am"+string(rune('a'+i)), soccer.PlayerPositionMidfield, 50, 60)

		add("atk-leg"+string(rune('a'+i)), soccer.PlayerPositionAttack, 90, 60)
		add("atk-wc"+string(rune('a'+i)), soccer.PlayerPositionAttack, 82, 60)
		add("atk-pro"+string(rune('a'+i)), soccer.PlayerPositionAttack, 75, 60)
		add("atk-sp"+string(rune('a'+i)), soccer.PlayerPositionAttack, 60, 60)
		add("atk-am"+string(rune('a'+i)), soccer.PlayerPositionAttack, 50, 60)
	}
	// A few aggressive players (high aggression, mid overall) to populate
	// the Aggressive pool.
	for i := 0; i < 5; i++ {
		add("agg"+string(rune('a'+i)), soccer.PlayerPositionDefense, 84, 90)
	}
	return out
}

// The headline contract: same seed + same inputs ⇒ identical assignments.
// This is the reason the engine seeds from an Algorand block hash — the
// allocation can be re-verified later without trusting the operator.
func TestAllocate_Deterministic(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := []allocation.Asset{
		{ID: "1", Name: "one", Tier: allocation.AssetTierS},
		{ID: "2", Name: "two", Tier: allocation.AssetTierA},
		{ID: "3", Name: "three", Tier: allocation.AssetTierB},
		{ID: "4", Name: "four", Tier: allocation.AssetTierC},
		{ID: "5", Name: "five", Tier: allocation.AssetTierAggressive},
	}

	for _, seed := range []int64{1, 42, 99} {
		first, err := allocation.Allocate(rand.New(rand.NewSource(seed)), pool, assets)
		require.NoError(t, err)
		second, err := allocation.Allocate(rand.New(rand.NewSource(seed)), pool, assets)
		require.NoError(t, err)
		assert.Equal(t, first, second, "seed %d allocations diverged", seed)
	}
}

// One Assignment per Asset, in order.
func TestAllocate_OneAssignmentPerAsset(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := []allocation.Asset{
		{ID: "a", Tier: allocation.AssetTierS},
		{ID: "b", Tier: allocation.AssetTierA},
		{ID: "c", Tier: allocation.AssetTierB},
	}

	got, err := allocation.Allocate(rand.New(rand.NewSource(1)), pool, assets)
	require.NoError(t, err)
	require.Len(t, got, 3)
	for i, a := range got {
		assert.Equal(t, assets[i].ID, a.Asset.ID, "assignment %d out of order", i)
	}
}

// Aggressive-tier assets must always pull from the aggressive pool — they
// never get a "regular" player. This is the rules entry point that exists
// solely because v1 wanted aggression as a separate dimension.
func TestAllocate_AggressiveTierUsesAggressivePool(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := make([]allocation.Asset, 50)
	for i := range assets {
		assets[i] = allocation.Asset{ID: "x", Tier: allocation.AssetTierAggressive}
	}

	got, err := allocation.Allocate(rand.New(rand.NewSource(1)), pool, assets)
	require.NoError(t, err)
	for i, a := range got {
		assert.GreaterOrEqual(t, a.Player.Attributes.AggressionRating, 80,
			"assignment %d player %s aggression %d below 80", i, a.Player.ID, a.Player.Attributes.AggressionRating)
		assert.LessOrEqual(t, a.Player.Attributes.OverallRating, 86,
			"assignment %d player %s overall %d above 86", i, a.Player.ID, a.Player.Attributes.OverallRating)
	}
}

// Tier S assets must never land an Amateur. The tier→level distribution
// table is the entire point of the allocation system.
func TestAllocate_TierSNeverGetsLowLevels(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := make([]allocation.Asset, 200)
	for i := range assets {
		assets[i] = allocation.Asset{ID: "x", Tier: allocation.AssetTierS}
	}
	got, err := allocation.Allocate(rand.New(rand.NewSource(7)), pool, assets)
	require.NoError(t, err)
	for _, a := range got {
		assert.GreaterOrEqual(t, a.Player.Attributes.OverallRating, 70,
			"Tier S assigned a player with overall %d (must be ≥70)", a.Player.Attributes.OverallRating)
	}
}

// Position rolls should produce all 4 positions across many assets — if
// one position never appeared, the weighting is broken.
func TestAllocate_AllPositionsRolled(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := make([]allocation.Asset, 200)
	for i := range assets {
		assets[i] = allocation.Asset{ID: "x", Tier: allocation.AssetTierS}
	}
	got, err := allocation.Allocate(rand.New(rand.NewSource(3)), pool, assets)
	require.NoError(t, err)

	seen := map[soccer.PlayerPosition]bool{}
	for _, a := range got {
		seen[a.Position] = true
	}
	assert.True(t, seen[soccer.PlayerPositionGoalkeeper])
	assert.True(t, seen[soccer.PlayerPositionDefense])
	assert.True(t, seen[soccer.PlayerPositionMidfield])
	assert.True(t, seen[soccer.PlayerPositionAttack])
}

// Empty pool → ErrEmptyPool. Don't silently substitute.
func TestAllocate_EmptyPoolReturnsError(t *testing.T) {
	empty := allocation.NewPool(nil, allocation.DefaultRules())
	assets := []allocation.Asset{{ID: "1", Tier: allocation.AssetTierS}}
	_, err := allocation.Allocate(rand.New(rand.NewSource(1)), empty, assets)
	assert.ErrorIs(t, err, allocation.ErrEmptyPool)
}

func TestAllocate_NilSourceReturnsError(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	_, err := allocation.Allocate(nil, pool, nil)
	require.Error(t, err)
}
