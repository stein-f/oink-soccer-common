package soccer

import "github.com/stein-f/oink-soccer-common/v2/internal/tuning"

// FormationConfig describes a tactical shape — its name, slot layout, and
// the trade-offs it makes against neutral. lost-pigs serializes this as JSON
// for the formation picker UI.
//
// The legacy DefenseModifier / ControlModifier / AttackModifier fields are
// kept for backward JSON compatibility but are derived from Profile (see
// formationConfigFor). New code should read Profile directly.
type FormationConfig struct {
	FormationType FormationType
	Slots         map[uint64]PlayerPosition
	Profile       FormationProfile

	// Deprecated: derived from Profile for backward compatibility. The v2
	// engine ignores these fields. Read Profile.* instead.
	DefenseModifier float64
	ControlModifier float64
	AttackModifier  float64
}

// FormationProfile is the trade-off matrix for a formation. See
// internal/tuning.FormationProfile for the full doc.
type FormationProfile = tuning.FormationProfile

var (
	ThePyramidFormation = formationConfigFor(FormationTypePyramid, slotsPyramid)
	TheDiamondFormation = formationConfigFor(FormationTypeDiamond, slotsDiamond)
	TheYFormation       = formationConfigFor(FormationTypeY, slotsY)
	TheBoxFormation     = formationConfigFor(FormationTypeBox, slotsBox)
)

var (
	slotsPyramid = map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionDefense,
		4: PlayerPositionMidfield,
		5: PlayerPositionAttack,
	}
	slotsDiamond = map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionMidfield,
		4: PlayerPositionMidfield,
		5: PlayerPositionAttack,
	}
	slotsY = map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionMidfield,
		4: PlayerPositionAttack,
		5: PlayerPositionAttack,
	}
	slotsBox = map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionDefense,
		4: PlayerPositionAttack,
		5: PlayerPositionAttack,
	}
)

func formationConfigFor(t FormationType, slots map[uint64]PlayerPosition) FormationConfig {
	profile := tuning.LookupFormationProfile(string(t))
	return FormationConfig{
		FormationType:   t,
		Slots:           slots,
		Profile:         profile,
		DefenseModifier: profile.DefSolidity,
		ControlModifier: profile.Possession,
		AttackModifier:  profile.ChanceCreation * profile.ChanceQuality,
	}
}

// formationProfileFor returns the trade-off profile for the formation
// stored on a lineup. Used by the engine. Falls back to NeutralProfile.
func formationProfileFor(t FormationType) FormationProfile {
	return tuning.LookupFormationProfile(string(t))
}
