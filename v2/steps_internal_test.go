package soccer

import (
	"math/rand"
	"testing"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stepLineup(formation FormationType, prefix string) GameLineup {
	var players []SelectedPlayer
	for slot := uint64(1); slot <= 5; slot++ {
		pos := formationConfigFor(formation, map[FormationType]map[uint64]PlayerPosition{
			FormationTypePyramid: slotsPyramid, FormationTypeDiamond: slotsDiamond,
			FormationTypeY: slotsY, FormationTypeBox: slotsBox,
		}[formation]).Slots[slot]
		players = append(players, SelectedPlayer{
			ID: prefix + string(rune('0'+slot)), SelectedPosition: pos,
			Attributes: PlayerAttributes{
				GoalkeeperRating: 70, DefenseRating: 70, ControlRating: 70, AttackRating: 70, SpeedRating: 70,
				PrimaryPosition: pos, Positions: []PlayerPosition{pos},
			},
		})
	}
	return GameLineup{Team: Team{ID: prefix, Formation: formation}, Players: players}
}

func stepSides(t *testing.T, home, away FormationType) (*stepMatch, *stepSide, *stepSide) {
	t.Helper()
	r := rand.New(rand.NewSource(1))
	h, _ := newStepSide(r, TeamTypeHome, stepLineup(home, "h"), &TeamTotals{})
	a, _ := newStepSide(r, TeamTypeAway, stepLineup(away, "a"), &TeamTotals{})
	return &stepMatch{r: r}, h, a
}

func TestSteps_LanesFollowLineupOrder(t *testing.T) {
	_, box, diamond := stepSides(t, FormationTypeBox, FormationTypeDiamond)

	// Two in a line take a side each, in line-up order; one holds the centre.
	require.Len(t, box.lines[PlayerPositionDefense], 2)
	assert.Equal(t, -1.0, box.lines[PlayerPositionDefense][0].lane)
	assert.Equal(t, 1.0, box.lines[PlayerPositionDefense][1].lane)
	assert.Equal(t, "h2", box.lines[PlayerPositionDefense][0].ID)
	assert.Empty(t, box.lines[PlayerPositionMidfield], "the Box fields no midfield")
	require.Len(t, diamond.lines[PlayerPositionDefense], 1)
	assert.Equal(t, 0.0, diamond.lines[PlayerPositionDefense][0].lane)
	assert.NotNil(t, box.keeper)
	assert.Len(t, box.outfield, 4)
}

func TestSteps_AWingerFacesTheDefenderOnHisSide(t *testing.T) {
	m, box, other := stepSides(t, FormationTypeBox, FormationTypeBox)
	left, right := box.lines[PlayerPositionAttack][0], box.lines[PlayerPositionAttack][1]

	// The attacking left wing is the defending side's right.
	opp, cover := m.facing(left, box, other, PlayerPositionDefense, tackleScore)
	assert.Equal(t, other.lines[PlayerPositionDefense][1].ID, opp.ID)
	assert.Equal(t, 1.0, cover, "two against two: nobody spare, nobody stretched")
	opp, _ = m.facing(right, box, other, PlayerPositionDefense, tackleScore)
	assert.Equal(t, other.lines[PlayerPositionDefense][0].ID, opp.ID)
}

// TestSteps_NumbersDecideCover is the whole of what a formation does in the
// step model: every pairing of shapes, for the two places an attack is met.
func TestSteps_NumbersDecideCover(t *testing.T) {
	midfield := map[FormationType]int{FormationTypePyramid: 1, FormationTypeDiamond: 2, FormationTypeY: 1, FormationTypeBox: 0}
	defence := map[FormationType]int{FormationTypePyramid: 2, FormationTypeDiamond: 1, FormationTypeY: 1, FormationTypeBox: 2}
	attack := map[FormationType]int{FormationTypePyramid: 1, FormationTypeDiamond: 1, FormationTypeY: 2, FormationTypeBox: 2}
	want := func(defenders, attackers int) float64 {
		switch {
		case defenders == 0:
			return tuning.StepsLineCover
		case defenders > attackers:
			return tuning.StepsSupport
		case defenders < attackers:
			return tuning.StepsStretched
		}
		return 1
	}
	formations := []FormationType{FormationTypePyramid, FormationTypeDiamond, FormationTypeY, FormationTypeBox}
	for _, af := range formations {
		for _, df := range formations {
			m, att, def := stepSides(t, af, df)

			// The forwards against the defence.
			for _, fwd := range att.lines[PlayerPositionAttack] {
				opp, cover := m.facing(fwd, att, def, PlayerPositionDefense, tackleScore)
				require.NotNil(t, opp, "%s v %s", af, df)
				assert.Equal(t, PlayerPositionDefense, opp.line)
				assert.Equal(t, want(defence[df], attack[af]), cover, "%s forwards v %s defence", af, df)
			}

			// The passer against the midfield. A side with no midfield
			// passes from the back; one with none to defend with sends a
			// forward back, out of his place.
			passers := att.lines[PlayerPositionMidfield]
			if len(passers) == 0 {
				passers = att.lines[PlayerPositionDefense]
			}
			for _, passer := range passers {
				opp, cover := m.facing(passer, att, def, PlayerPositionMidfield, interceptScore)
				require.NotNil(t, opp, "%s v %s", af, df)
				assert.Equal(t, want(midfield[df], len(passers)), cover, "%s passers v %s midfield", af, df)
				if midfield[df] == 0 {
					assert.NotEqual(t, PlayerPositionMidfield, opp.line)
				}
			}
		}
	}
}

func TestSteps_ABallWinnerIsNotStretched(t *testing.T) {
	m, y, diamond := stepSides(t, FormationTypeY, FormationTypeDiamond)
	fwd := y.lines[PlayerPositionAttack][0]
	_, cover := m.facing(fwd, y, diamond, PlayerPositionDefense, tackleScore)
	assert.Equal(t, tuning.StepsStretched, cover, "one defender against two forwards")

	diamond.lines[PlayerPositionDefense][0].Role = PlayerRoleBallWinner
	_, cover = m.facing(fwd, y, diamond, PlayerPositionDefense, tackleScore)
	assert.Equal(t, 1.0, cover)
}

func TestSteps_ARatingEdgeIsWorthSomethingButNotEverything(t *testing.T) {
	// The plan's target: 20 points better wins a contest by 15-25 points
	// more than a coin toss, and nobody is ever a certainty.
	edge := oddsToP(stepOdds(90, 70, 1)) - 0.5
	assert.InDelta(t, 0.20, edge, 0.05)
	assert.InDelta(t, 0.5, oddsToP(stepOdds(80, 80, 1)), 1e-9)
	assert.Less(t, oddsToP(stepOdds(95, 70, 1)), 0.85)
	assert.Greater(t, oddsToP(stepOdds(60, 90, 1)), 0.15)
}

func TestSteps_TeamBoostsAreSoftened(t *testing.T) {
	// A +5% item is in every contest of an attack and of defending one, so
	// it reaches the odds softened.
	lineup := stepLineup(FormationTypeDiamond, "h")
	lineup.ItemBoosts = []Boost{{BoostType: BoostTypeTeam, MinBoost: 1.05, MaxBoost: 1.05}}
	side, _ := newStepSide(rand.New(rand.NewSource(1)), TeamTypeAway, lineup, &TeamTotals{})
	mod := side.mod(side.outfield[0])
	assert.Greater(t, mod, 1.0)
	assert.Less(t, mod, 1.05)
}
