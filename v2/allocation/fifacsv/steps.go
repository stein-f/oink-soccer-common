package fifacsv

import soccer "github.com/stein-f/oink-soccer-common/v2"

// StepSource holds the dataset columns behind the player-on-player attributes
// the step engine compares (soccer.PlayerAttributes: Dribbling, Passing, ...).
//
// Two importers read the dataset: LoadCandidates here, and the lost-pigs
// importer that writes each season's player records. Both fill a StepSource
// from their own row and call Apply, so the derivation lives in one place and
// a player's stored attributes match the ones the engine was calibrated on.
//
// Field names follow the sofifa columns with their group prefix dropped
// (skill_dribbling → SkillDribbling, movement_agility → Agility).
type StepSource struct {
	Dribbling, SkillDribbling, Agility, Balance int
	Passing, ShortPassing, Vision, LongPassing  int
	Crossing, Curve                             int
	Acceleration, SprintSpeed                   int
	Strength, Jumping                           int
	Interceptions, MarkingAwareness             int
	AttackPositioning                           int
	ShotPower, LongShots                        int
	GKReflexes, GKDiving, GKHandling            int
	GKSpeed, GKPositioning                      int
}

// Apply writes the player-on-player attributes onto attrs. A column the
// dataset leaves empty is skipped in its average; an attribute with no data
// at all stays zero, and the engine's Effective* accessor falls back to the
// core rating. The dataset blanks the composite columns (dribbling, passing)
// for goalkeepers, so theirs come from the detailed columns alone.
//
// Apply makes no random draws: the same row always gives the same attributes.
func (s StepSource) Apply(attrs *soccer.PlayerAttributes) {
	attrs.Dribbling = averagePositive(s.Dribbling, s.SkillDribbling, s.Agility, s.Balance)
	attrs.Passing = averagePositive(s.Passing, s.ShortPassing, s.Vision, s.LongPassing)
	attrs.Crossing = averagePositive(s.Crossing, s.Curve)
	attrs.Pace = averagePositive(s.Acceleration, s.SprintSpeed)
	attrs.Strength = averagePositive(s.Strength, s.Jumping)
	attrs.Interceptions = s.Interceptions
	attrs.Positioning = s.AttackPositioning
	attrs.Marking = s.MarkingAwareness
	attrs.ShotPower = averagePositive(s.ShotPower, s.LongShots)
	attrs.Reflexes = averagePositive(s.GKReflexes, s.GKDiving)
	attrs.Handling = s.GKHandling
	attrs.OneOnOne = oneOnOne(s.GKSpeed, s.GKPositioning)
}

// oneOnOne weights positioning twice as heavily as speed. A keeper's speed
// column sits about 30 points below his other skills (median 32 against 63
// for positioning in the FC 27 dataset), so an even average would put the
// attribute on a lower scale than Reflexes and Handling and let a quick,
// poorly positioned keeper outrank a much better one.
func oneOnOne(speed, positioning int) int {
	switch {
	case speed > 0 && positioning > 0:
		return (speed + 2*positioning) / 3
	case positioning > 0:
		return positioning
	}
	return speed
}
