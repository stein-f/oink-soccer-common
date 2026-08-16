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
	// Specialists: pass their position's signature-stat filter at both
	// drawable levels (World Class + Professional).
	addSpecialist := func(id string, pos soccer.PlayerPosition, overall int, set func(*soccer.PlayerAttributes)) {
		attrs := soccer.PlayerAttributes{PrimaryPosition: pos, OverallRating: overall, AggressionRating: 55}
		set(&attrs)
		out = append(out, allocation.Candidate{ID: id, Name: id, Attributes: attrs})
	}
	for i := 0; i < 3; i++ {
		s := string(rune('a' + i))
		addSpecialist("spec-atk-wc"+s, soccer.PlayerPositionAttack, 82, func(a *soccer.PlayerAttributes) { a.Finishing = 85 })
		addSpecialist("spec-atk-pro"+s, soccer.PlayerPositionAttack, 74, func(a *soccer.PlayerAttributes) { a.Finishing = 80 })
		addSpecialist("spec-mid-wc"+s, soccer.PlayerPositionMidfield, 81, func(a *soccer.PlayerAttributes) { a.Technique = 84 })
		addSpecialist("spec-mid-pro"+s, soccer.PlayerPositionMidfield, 73, func(a *soccer.PlayerAttributes) { a.Technique = 79 })
		addSpecialist("spec-def-wc"+s, soccer.PlayerPositionDefense, 83, func(a *soccer.PlayerAttributes) { a.Tackling = 86 })
		addSpecialist("spec-def-pro"+s, soccer.PlayerPositionDefense, 75, func(a *soccer.PlayerAttributes) { a.Tackling = 81 })
	}
	return out
}

// makeLegends returns curated legend cards — PlayerLevel Legendary marks
// them for the exactly-once legend lottery.
func makeLegends(n int) []allocation.Candidate {
	positions := []soccer.PlayerPosition{
		soccer.PlayerPositionAttack,
		soccer.PlayerPositionMidfield,
		soccer.PlayerPositionDefense,
		soccer.PlayerPositionGoalkeeper,
	}
	var out []allocation.Candidate
	for i := 0; i < n; i++ {
		out = append(out, allocation.Candidate{
			ID:   "L000" + string(rune('1'+i)),
			Name: "Legend " + string(rune('1'+i)),
			Attributes: soccer.PlayerAttributes{
				PrimaryPosition: positions[i%len(positions)],
				OverallRating:   90 + i%5,
				PlayerLevel:     soccer.PlayerLevelLegendary,
			},
		})
	}
	return out
}

// The headline contract: same seed + same inputs ⇒ identical assignments.
// This is the reason the engine seeds from an Algorand block hash — the
// allocation can be re-verified later without trusting the operator.
func TestAllocate_Deterministic(t *testing.T) {
	candidates := append(makeCandidates(), makeLegends(3)...)
	pool := allocation.NewPool(candidates, allocation.DefaultRules())
	assets := []allocation.Asset{
		{ID: "1", Name: "one", Tier: allocation.AssetTierS},
		{ID: "2", Name: "two", Tier: allocation.AssetTierA},
		{ID: "3", Name: "three", Tier: allocation.AssetTierB},
		{ID: "4", Name: "four", Tier: allocation.AssetTierC},
		{ID: "5", Name: "five", Tier: allocation.AssetTierAggressive},
		{ID: "6", Name: "six", Tier: allocation.AssetTierSpecialist},
		{ID: "7", Name: "seven", Tier: allocation.AssetTierSpecialist},
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

// Specialist-tier assets draw position-respecting specialists: the player
// fills the rolled position and clears that position's signature-stat
// filter. The World Class band's 86 ceiling doubles as the overall cap.
func TestAllocate_SpecialistTierRespectsPositionFilters(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := make([]allocation.Asset, 300)
	for i := range assets {
		assets[i] = allocation.Asset{ID: "x", Tier: allocation.AssetTierSpecialist}
	}
	got, err := allocation.Allocate(rand.New(rand.NewSource(11)), pool, assets)
	require.NoError(t, err)

	for i, a := range got {
		attrs := a.Player.Attributes
		assert.Equal(t, a.Position, attrs.PrimaryPosition,
			"assignment %d: specialist pick must respect the rolled position", i)
		assert.LessOrEqual(t, attrs.OverallRating, 86,
			"assignment %d player %s overall %d above the 86 cap", i, a.Player.ID, attrs.OverallRating)
		switch a.Position {
		case soccer.PlayerPositionAttack:
			assert.GreaterOrEqual(t, attrs.EffectiveFinishing(), 78,
				"assignment %d: attacker %s below finishing floor", i, a.Player.ID)
		case soccer.PlayerPositionMidfield:
			assert.GreaterOrEqual(t, attrs.EffectiveTechnique(), 78,
				"assignment %d: midfielder %s below technique floor", i, a.Player.ID)
		case soccer.PlayerPositionDefense:
			assert.GreaterOrEqual(t, attrs.EffectiveTackling(), 78,
				"assignment %d: defender %s below tackling floor", i, a.Player.ID)
		}
	}
}

// The published ratio: schnoz World Class odds are Tier S's 49% ÷ 10 =
// 4.9%. Monte Carlo over one big allocation; generous tolerance to keep
// the test stable across Go versions.
func TestAllocate_SpecialistWorldClassShareMatchesRatio(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := make([]allocation.Asset, 20000)
	for i := range assets {
		assets[i] = allocation.Asset{ID: "x", Tier: allocation.AssetTierSpecialist}
	}
	got, err := allocation.Allocate(rand.New(rand.NewSource(13)), pool, assets)
	require.NoError(t, err)

	var wc int
	for _, a := range got {
		if a.Player.Attributes.OverallRating >= 80 {
			wc++
		}
	}
	share := float64(wc) / float64(len(got))
	assert.InDelta(t, 0.049, share, 0.015,
		"specialist World Class share %f should be ~4.9%%", share)
}

// Legend cards are seeded exactly once each, and only Tier S / Tier A /
// Tier Specialist assets can host them.
func TestAllocate_LegendsSeededExactlyOnceOnEligibleTiers(t *testing.T) {
	candidates := append(makeCandidates(), makeLegends(3)...)
	pool := allocation.NewPool(candidates, allocation.DefaultRules())

	var assets []allocation.Asset
	addAssets := func(n int, tier allocation.AssetTier) {
		for i := 0; i < n; i++ {
			assets = append(assets, allocation.Asset{ID: string(tier), Tier: tier})
		}
	}
	addAssets(50, allocation.AssetTierS)
	addAssets(50, allocation.AssetTierA)
	addAssets(100, allocation.AssetTierSpecialist)
	addAssets(50, allocation.AssetTierB)
	addAssets(50, allocation.AssetTierC)

	got, err := allocation.Allocate(rand.New(rand.NewSource(17)), pool, assets)
	require.NoError(t, err)

	hosts := map[string]allocation.AssetTier{}
	for _, a := range got {
		if a.Player.Attributes.PlayerLevel == soccer.PlayerLevelLegendary {
			_, dup := hosts[a.Player.ID]
			assert.False(t, dup, "legend %s allocated more than once", a.Player.ID)
			hosts[a.Player.ID] = a.Asset.Tier
		}
	}
	require.Len(t, hosts, 3, "every legend card must be seeded exactly once")
	for id, tier := range hosts {
		assert.Contains(t,
			[]allocation.AssetTier{allocation.AssetTierS, allocation.AssetTierA, allocation.AssetTierSpecialist},
			tier, "legend %s hosted on ineligible tier %s", id, tier)
	}
}

// The legend lottery is weighted 10:1 — with equal counts of Tier S and
// Tier Specialist assets, the pig share of hosts converges on
// 10/11 ≈ 90.9%.
func TestAllocate_LegendHostOddsFavorPigsTenToOne(t *testing.T) {
	candidates := append(makeCandidates(), makeLegends(1)...)
	pool := allocation.NewPool(candidates, allocation.DefaultRules())

	var assets []allocation.Asset
	for i := 0; i < 100; i++ {
		assets = append(assets, allocation.Asset{ID: "s", Tier: allocation.AssetTierS})
	}
	for i := 0; i < 100; i++ {
		assets = append(assets, allocation.Asset{ID: "z", Tier: allocation.AssetTierSpecialist})
	}

	const runs = 3000
	var sHosts int
	for seed := int64(0); seed < runs; seed++ {
		got, err := allocation.Allocate(rand.New(rand.NewSource(seed)), pool, assets)
		require.NoError(t, err)
		for _, a := range got {
			if a.Player.Attributes.PlayerLevel == soccer.PlayerLevelLegendary {
				if a.Asset.Tier == allocation.AssetTierS {
					sHosts++
				}
			}
		}
	}
	share := float64(sHosts) / float64(runs)
	assert.InDelta(t, 10.0/11.0, share, 0.02,
		"Tier S legend-host share %f should be ~10/11", share)
}

// Real players whose overall reaches the legendary band (87+) but who are
// not curated legend cards are unreachable: they index only under the
// Legendary level, which no tier distribution draws.
func TestAllocate_RealPlayersAboveCapNeverDrawn(t *testing.T) {
	pool := allocation.NewPool(makeCandidates(), allocation.DefaultRules())
	assets := make([]allocation.Asset, 500)
	for i := range assets {
		assets[i] = allocation.Asset{ID: "x", Tier: allocation.AssetTierS}
	}
	got, err := allocation.Allocate(rand.New(rand.NewSource(23)), pool, assets)
	require.NoError(t, err)
	for _, a := range got {
		assert.LessOrEqual(t, a.Player.Attributes.OverallRating, 86,
			"player %s (overall %d) should be unreachable — 87+ is legend-card territory",
			a.Player.ID, a.Player.Attributes.OverallRating)
	}
}
