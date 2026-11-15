package fifacsv_test

import (
	"bufio"
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
	"github.com/stein-f/oink-soccer-common/v2/allocation/fifacsv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot is the checkout root holding cmd/allocation, relative to this
// package directory (v2/allocation/fifacsv).
const repoRoot = "../../.."

const fc27Header = "player_id,common_name,first_name,last_name,overall_rating,position,alternate_positions,club,league,nationality,gender,preferred_foot,skill_moves,weak_foot,height_cm,weight_kg,birthdate,playstyles,pace,shooting,passing,dribbling,defending,physicality,attacking_crossing,attacking_finishing,attacking_heading_accuracy,attacking_short_passing,attacking_volleys,skill_dribbling,skill_curve,skill_fk_accuracy,skill_long_passing,skill_ball_control,movement_acceleration,movement_sprint_speed,movement_agility,movement_reactions,movement_balance,power_shot_power,power_jumping,power_stamina,power_strength,power_long_shots,mentality_aggression,mentality_interceptions,mentality_positioning,mentality_vision,mentality_penalties,mentality_composure,defending_awareness,defending_standing_tackle,defending_sliding_tackle,goalkeeping_diving,goalkeeping_handling,goalkeeping_kicking,goalkeeping_positioning,goalkeeping_reflexes,edition,snapshot_date"

// Rows lifted from the 2026-09-12 snapshot (Courtois: keeper facets in the
// outfield columns; Putellas: women's division).
const fc27Rows = `231747,,Kylian,Mbappé,91,ST,LW,Real Madrid,LALIGA EA SPORTS,France,Men's Football,Right,5,4,,,1998-12-20,"Finesse Shot, Quick Step+",96,91,80,92,29,76,74,95,78,87,87,92,80,70,74,93,97,96,93,92,82,91,90,83,77,86,61,19,91,83,86,88,15,34,24,13,5,7,11,6,fc27,2026-09-12
192119,,Thibaut,Courtois,90,GK,,Real Madrid,LALIGA EA SPORTS,Belgium,Men's Football,Left,1,3,,,1992-05-11,"Far Throw, Rush Out+",87,89,78,90,46,90,14,14,13,33,12,13,19,20,35,23,42,52,63,88,45,59,68,38,70,17,23,15,13,60,27,66,20,18,16,87,89,78,90,90,fc27,2026-09-12
227203,Alexia Putellas,Alexia,Putellas Segura,91,CM,CAM ST,London City,Barclays WSL,Spain,Women's Football,Left,5,5,,,1994-02-04,Finesse Shot,80,90,91,92,72,75,88,90,74,94,87,91,89,83,90,94,80,80,87,92,92,89,82,80,74,90,70,78,91,93,89,92,60,82,64,15,17,11,15,10,fc27,2026-09-12
`

// legacyFixture returns the real FIFA 22 header plus one legend-card row.
func legacyFixture(t *testing.T) string {
	t.Helper()
	f, err := os.Open(filepath.Join(repoRoot, fifacsv.FIFA22Dataset))
	require.NoError(t, err)
	defer f.Close()
	header, err := bufio.NewReader(f).ReadString('\n')
	require.NoError(t, err)
	header = strings.TrimRight(header, "\r\n")

	cols := strings.Split(header, ",")
	legend := make([]string, len(cols))
	for i, c := range cols {
		switch c {
		case "sofifa_id":
			legend[i] = "L0099"
		case "short_name", "long_name":
			legend[i] = "Test Legend"
		case "player_positions", "club_position":
			legend[i] = "ST"
		case "overall":
			legend[i] = "95"
		}
	}
	return header + "\r\n" + strings.Join(legend, ",") + "\r\n"
}

func convert(t *testing.T, fc27 string) (map[string]allocation.Candidate, fifacsv.ConvertStats) {
	t.Helper()
	var out bytes.Buffer
	stats, err := fifacsv.ConvertFC27(strings.NewReader(fc27), strings.NewReader(legacyFixture(t)), &out)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "players.csv")
	require.NoError(t, os.WriteFile(path, out.Bytes(), 0o644))
	cands, err := fifacsv.LoadCandidates(path, rand.New(rand.NewSource(1)))
	require.NoError(t, err)
	byID := map[string]allocation.Candidate{}
	for _, c := range cands {
		byID[c.ID] = c
	}
	return byID, stats
}

func TestConvertFC27(t *testing.T) {
	byID, stats := convert(t, fc27Header+"\n"+fc27Rows)

	assert.Equal(t, fifacsv.ConvertStats{Players: 2, Goalkeepers: 1, SkippedWomen: 1, Legends: 1}, stats)
	assert.Len(t, byID, 3)
	assert.NotContains(t, byID, "227203", "women's division is excluded")

	mbappe := byID["231747"].Attributes
	assert.Equal(t, "K. Mbappé", byID["231747"].Name)
	assert.Equal(t, 91, mbappe.OverallRating)
	assert.Equal(t, soccer.PlayerLevelWorldClass, mbappe.PlayerLevel)
	assert.Equal(t, soccer.PlayerPositionAttack, mbappe.PrimaryPosition)
	assert.ElementsMatch(t, []soccer.PlayerPosition{soccer.PlayerPositionAttack, soccer.PlayerPositionMidfield}, mbappe.Positions)
	assert.Equal(t, 96, mbappe.SpeedRating)
	assert.Equal(t, 91, mbappe.AttackRating)
	assert.Equal(t, 80, mbappe.ControlRating)
	assert.Equal(t, 29, mbappe.DefenseRating)
	assert.Equal(t, 95, mbappe.Finishing)
	assert.Equal(t, "https://sofifa.com/player/231747", mbappe.BasedOnPlayerURL)

	// The keeper's DIV/HAN/KIC/REF/SPD facets must not land in the outfield
	// ratings: they load exactly as a FIFA 22 keeper did.
	courtois := byID["192119"].Attributes
	assert.Equal(t, soccer.PlayerPositionGoalkeeper, courtois.PrimaryPosition)
	assert.Equal(t, 89, courtois.GoalkeeperRating, "goalkeeping_handling")
	assert.Equal(t, courtois.OverallRating, courtois.SpeedRating, "blank pace falls back to overall")
	for name, v := range map[string]int{"attack": courtois.AttackRating, "defense": courtois.DefenseRating, "control": courtois.ControlRating} {
		assert.True(t, v >= 5 && v <= 30, "%s %d should be the 5-30 filler", name, v)
	}

	legend := byID["L0099"].Attributes
	assert.Equal(t, soccer.PlayerLevelLegendary, legend.PlayerLevel)
	assert.Equal(t, 95, legend.OverallRating)
}

func TestConvertFC27_Rejects(t *testing.T) {
	mbappe := strings.SplitN(fc27Rows, "\n", 2)[0]
	cases := map[string]string{
		"duplicate player_id": fc27Header + "\n" + mbappe + "\n" + mbappe + "\n",
		"unknown position":    fc27Header + "\n" + strings.Replace(mbappe, ",ST,LW,", ",XX,LW,", 1) + "\n",
		"missing column":      strings.Replace(fc27Header, "playstyles", "play_styles", 1) + "\n" + mbappe + "\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := fifacsv.ConvertFC27(strings.NewReader(in), strings.NewReader(legacyFixture(t)), &bytes.Buffer{})
			assert.Error(t, err)
		})
	}
}

// TestFC27Dataset guards the committed fifa_players_fc27.csv.
func TestFC27Dataset(t *testing.T) {
	cands, err := fifacsv.LoadCandidates(filepath.Join(repoRoot, fifacsv.FC27Dataset), rand.New(rand.NewSource(1)))
	require.NoError(t, err)

	seen := map[string]bool{}
	legends := 0
	for _, c := range cands {
		require.False(t, seen[c.ID], "duplicate id %s", c.ID)
		seen[c.ID] = true
		a := c.Attributes
		require.NotEmpty(t, a.PrimaryPosition, "player %s has no position", c.ID)
		if strings.HasPrefix(c.ID, allocation.LegendCardPrefix) {
			legends++
			continue
		}
		if a.PrimaryPosition == soccer.PlayerPositionGoalkeeper {
			require.Equal(t, a.OverallRating, a.SpeedRating, "keeper %s carries FC 27 facets", c.ID)
		}
	}
	assert.Equal(t, 41, legends, "every curated legend card is carried over")
	assert.Greater(t, len(cands)-legends, 15000)
}

func TestDatasetPath(t *testing.T) {
	assert.Equal(t, fifacsv.FIFA22Dataset, fifacsv.DatasetPath(1))
	assert.Equal(t, fifacsv.FIFA22Dataset, fifacsv.DatasetPath(fifacsv.FC27FirstSeason-1))
	assert.Equal(t, fifacsv.FC27Dataset, fifacsv.DatasetPath(fifacsv.FC27FirstSeason))
	assert.Equal(t, fifacsv.FC27Dataset, fifacsv.DatasetPath(fifacsv.FC27FirstSeason+1))
	assert.Equal(t, 19, fifacsv.FC27FirstSeason, "lost-pigs domain.FifaDatasetKey must move with this")
}
