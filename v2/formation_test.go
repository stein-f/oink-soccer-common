package soccer_test

import (
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stretchr/testify/assert"
)

// Every formation must publish a non-zero profile. A zero-valued profile
// would make the engine multiply everything by 0.
func TestEveryFormationPublishesAProfile(t *testing.T) {
	formations := []soccer.FormationConfig{
		soccer.ThePyramidFormation,
		soccer.TheDiamondFormation,
		soccer.TheYFormation,
		soccer.TheBoxFormation,
	}
	for _, f := range formations {
		t.Run(string(f.FormationType), func(t *testing.T) {
			assert.NotZero(t, f.Profile.Possession)
			assert.NotZero(t, f.Profile.ChanceCreation)
			assert.NotZero(t, f.Profile.ChanceQuality)
			assert.NotZero(t, f.Profile.DefSolidity)
			assert.NotZero(t, f.Profile.InjuryRisk)
		})
	}
}

// The legacy modifier fields exist purely for the lost-pigs frontend. They
// must be derived from Profile so they stay in lockstep — no formation can
// publish a control modifier of 1.05 (advertised) but a possession profile
// of 0.95 (actually used by the engine).
func TestLegacyModifiersAreDerivedFromProfile(t *testing.T) {
	formations := []soccer.FormationConfig{
		soccer.ThePyramidFormation,
		soccer.TheDiamondFormation,
		soccer.TheYFormation,
		soccer.TheBoxFormation,
	}
	for _, f := range formations {
		t.Run(string(f.FormationType), func(t *testing.T) {
			assert.Equal(t, f.Profile.DefSolidity, f.DefenseModifier)
			assert.Equal(t, f.Profile.Possession, f.ControlModifier)
			assert.Equal(t, f.Profile.ChanceCreation*f.Profile.ChanceQuality, f.AttackModifier)
		})
	}
}

// No formation should strictly dominate another — i.e. for every pair (A,B)
// there must be at least one axis where A is worse than B (treating
// InjuryRisk as "worse when higher", others as "better when higher").
func TestNoFormationStrictlyDominates(t *testing.T) {
	formations := map[string]soccer.FormationProfile{
		"Pyramid": soccer.ThePyramidFormation.Profile,
		"Diamond": soccer.TheDiamondFormation.Profile,
		"Y":       soccer.TheYFormation.Profile,
		"Box":     soccer.TheBoxFormation.Profile,
	}
	for nameA, a := range formations {
		for nameB, b := range formations {
			if nameA == nameB {
				continue
			}
			t.Run(nameA+"_vs_"+nameB, func(t *testing.T) {
				dominates := a.Possession >= b.Possession &&
					a.ChanceCreation >= b.ChanceCreation &&
					a.ChanceQuality >= b.ChanceQuality &&
					a.DefSolidity >= b.DefSolidity &&
					a.InjuryRisk <= b.InjuryRisk &&
					(a.Possession > b.Possession || a.ChanceCreation > b.ChanceCreation ||
						a.ChanceQuality > b.ChanceQuality || a.DefSolidity > b.DefSolidity ||
						a.InjuryRisk < b.InjuryRisk)
				assert.False(t, dominates, "%s strictly dominates %s — every axis at least equal and at least one strictly better", nameA, nameB)
			})
		}
	}
}
