package tuning_test

import (
	"testing"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
	"github.com/stretchr/testify/assert"
)

// Tuning constants are the dials of the engine; these tests don't pin exact
// values (those will drift as we tune balance) but they pin invariants that
// must hold for the engine to behave sensibly.

func TestPositionWeightsSumToOne(t *testing.T) {
	cases := map[string]tuning.PositionWeights{
		"control": tuning.ControlPositionWeights,
		"defense": tuning.DefensePositionWeights,
	}
	for name, w := range cases {
		t.Run(name, func(t *testing.T) {
			sum := w.Goalkeeper + w.Defense + w.Midfield + w.Attack
			assert.InDelta(t, 1.0, sum, 1e-9, "%s position weights must sum to 1.0, got %v", name, sum)
		})
	}
}

func TestStatsReductionOrdering(t *testing.T) {
	// Higher-severity injuries must reduce stats more (lower multiplier).
	assert.Less(t, tuning.StatsReductionHigh, tuning.StatsReductionMed)
	assert.Less(t, tuning.StatsReductionMed, tuning.StatsReductionLow)
	assert.Less(t, tuning.StatsReductionLow, 1.0)
	assert.Greater(t, tuning.StatsReductionHigh, 0.0)
}

func TestBoostDecayBounds(t *testing.T) {
	assert.Greater(t, tuning.BoostDecay, 0.0)
	assert.Less(t, tuning.BoostDecay, 1.0)
	assert.Greater(t, tuning.BoostMinMultiplier, 0.0)
	assert.Less(t, tuning.BoostMinMultiplier, 1.0)
}

func TestChanceRangesValid(t *testing.T) {
	// Every chance range must have Min ≤ Max and Min ≥ 1.
	for key, r := range tuning.FormationChanceRanges {
		assert.LessOrEqual(t, r.Min, r.Max, "range %q has Min > Max", key)
		assert.GreaterOrEqual(t, r.Min, 1, "range %q has Min < 1", key)
	}
	assert.LessOrEqual(t, tuning.FallbackChanceRange.Min, tuning.FallbackChanceRange.Max)
}

func TestSkillCurve_ShapeAndBounds(t *testing.T) {
	// Curve must be monotonically increasing, bounded to [floor, 100], and
	// strictly convex (the whole point — amplifies skill differential).
	assert.Equal(t, 100.0, tuning.SkillCurve(100))
	assert.Equal(t, tuning.SkillCurveFloor, tuning.SkillCurve(0))
	assert.Equal(t, tuning.SkillCurveFloor, tuning.SkillCurve(-5)) // negative input ⇒ floor

	// Monotonic.
	prev := tuning.SkillCurve(0)
	for x := 1.0; x <= 100; x++ {
		v := tuning.SkillCurve(x)
		assert.GreaterOrEqual(t, v, prev, "curve must be monotonic at x=%v", x)
		prev = v
	}

	// Convex: a +5 raw bump near the top must scale to a larger absolute
	// gain than the same bump near the middle.
	gainTop := tuning.SkillCurve(95) - tuning.SkillCurve(90)
	gainMid := tuning.SkillCurve(55) - tuning.SkillCurve(50)
	assert.Greater(t, gainTop, gainMid, "curve must be convex (upper bumps > lower bumps)")
}

func TestEventMinuteBucketsCoverMatch(t *testing.T) {
	// Buckets must form a contiguous span from minute 1 to at least minute 90.
	if len(tuning.EventMinuteBuckets) == 0 {
		t.Fatal("no event minute buckets defined")
	}
	first := tuning.EventMinuteBuckets[0]
	last := tuning.EventMinuteBuckets[len(tuning.EventMinuteBuckets)-1]
	assert.Equal(t, 1, first.MinMinute, "first bucket must start at minute 1")
	assert.GreaterOrEqual(t, last.MaxMinute, 90, "last bucket must cover full time")

	for i, b := range tuning.EventMinuteBuckets {
		assert.Less(t, b.MinMinute, b.MaxMinute, "bucket %d Min < Max", i)
		assert.Greater(t, b.Weight, uint(0), "bucket %d must have non-zero weight", i)
		if i > 0 {
			prev := tuning.EventMinuteBuckets[i-1]
			assert.Equal(t, prev.MaxMinute+1, b.MinMinute, "bucket %d not contiguous with bucket %d", i, i-1)
		}
	}
}
