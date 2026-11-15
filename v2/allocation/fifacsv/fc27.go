package fifacsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/stein-f/oink-soccer-common/v2/allocation"
)

// fc27MenGender is EA's division label for the men's game. The women's
// division is excluded: the game has only ever drawn from men's football.
const fc27MenGender = "Men's Football"

// fc27SameColumns are the detailed attributes whose names match in both
// schemas.
var fc27SameColumns = []string{
	"preferred_foot", "weak_foot", "skill_moves",
	"attacking_crossing", "attacking_finishing", "attacking_heading_accuracy",
	"attacking_short_passing", "attacking_volleys",
	"skill_dribbling", "skill_curve", "skill_fk_accuracy", "skill_long_passing", "skill_ball_control",
	"movement_acceleration", "movement_sprint_speed", "movement_agility", "movement_reactions", "movement_balance",
	"power_shot_power", "power_jumping", "power_stamina", "power_strength", "power_long_shots",
	"mentality_aggression", "mentality_interceptions", "mentality_positioning",
	"mentality_vision", "mentality_penalties", "mentality_composure",
	"defending_standing_tackle", "defending_sliding_tackle",
	"goalkeeping_diving", "goalkeeping_handling", "goalkeeping_kicking",
	"goalkeeping_positioning", "goalkeeping_reflexes",
}

// fc27Renamed pairs {sofifa column, FC 27 column} for fields that only
// changed name. Ordered slices, not maps: repo convention keeps every
// iteration order fixed.
var fc27Renamed = [][2]string{
	{"sofifa_id", "player_id"},
	{"overall", "overall_rating"},
	{"dob", "birthdate"},
	{"club_name", "club"},
	{"league_name", "league"},
	{"nationality_name", "nationality"},
	{"player_traits", "playstyles"},
	{"defending_marking_awareness", "defending_awareness"},
}

// fc27OutfieldFacets pairs {sofifa column, FC 27 column} for the card facets.
var fc27OutfieldFacets = [][2]string{
	{"pace", "pace"},
	{"shooting", "shooting"},
	{"passing", "passing"},
	{"dribbling", "dribbling"},
	{"defending", "defending"},
	{"physic", "physicality"},
}

// ConvertStats summarises a ConvertFC27 run.
type ConvertStats struct {
	Players      int // men's players written
	Goalkeepers  int
	SkippedWomen int
	Legends      int // curated legend cards carried over from the legacy dataset
}

// ConvertFC27 rewrites EA's FC 27 ratings export (Kaggle
// mikedpad/ea-sports-fc27-player-ratings, players.csv) into the sofifa
// FIFA 22 column layout that LoadCandidates and the lost-pigs importer read,
// so neither loader changes. legacy is the current FIFA 22 dataset: it
// supplies the header and the curated L-prefixed legend cards, which are
// appended verbatim.
//
// Goalkeepers: FC 27 stores a keeper's DIV/HAN/KIC/REF/SPD/POS card facets
// in the outfield facet columns (pace…physicality). sofifa left those blank
// for keepers and both loaders depend on it (a zero fills from a low random
// band; zero pace falls back to overall), so they are blanked here. Copying
// them would hand every keeper ~72 attack and ~74 speed.
func ConvertFC27(fc27, legacy io.Reader, out io.Writer) (ConvertStats, error) {
	var stats ConvertStats

	header, legends, err := readLegacy(legacy)
	if err != nil {
		return stats, err
	}
	stats.Legends = len(legends)
	outIdx := make(map[string]int, len(header))
	for i, h := range header {
		outIdx[h] = i
	}

	in := csv.NewReader(fc27)
	inHeader, err := in.Read()
	if err != nil {
		return stats, fmt.Errorf("fc27: read header: %w", err)
	}
	inIdx := make(map[string]int, len(inHeader))
	for i, h := range inHeader {
		inIdx[strings.TrimPrefix(h, "\ufeff")] = i
	}
	required := []string{"gender", "position", "alternate_positions", "common_name", "first_name", "last_name"}
	required = append(required, fc27SameColumns...)
	for _, pair := range fc27Renamed {
		required = append(required, pair[1])
	}
	for _, pair := range fc27OutfieldFacets {
		required = append(required, pair[1])
	}
	for _, col := range required {
		if _, ok := inIdx[col]; !ok {
			return stats, fmt.Errorf("fc27: missing column %q", col)
		}
	}

	type row struct {
		id, overall int
		fields      []string
	}
	var rows []row
	seen := map[string]bool{}
	for {
		rec, err := in.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return stats, fmt.Errorf("fc27: %w", err)
		}
		get := func(col string) string { return strings.TrimSpace(rec[inIdx[col]]) }

		if get("gender") != fc27MenGender {
			stats.SkippedWomen++
			continue
		}

		id := get("player_id")
		numericID, err := strconv.Atoi(id)
		if err != nil {
			return stats, fmt.Errorf("fc27: non-numeric player_id %q", id)
		}
		if seen[id] {
			return stats, fmt.Errorf("fc27: duplicate player_id %s", id)
		}
		seen[id] = true
		overall, err := strconv.Atoi(get("overall_rating"))
		if err != nil {
			return stats, fmt.Errorf("fc27: player %s: overall_rating %q", id, get("overall_rating"))
		}
		primary := get("position")
		if mapPosition(primary) == "" {
			return stats, fmt.Errorf("fc27: player %s: unknown position %q", id, primary)
		}

		fields := make([]string, len(header))
		set := func(col, value string) {
			if i, ok := outIdx[col]; ok {
				fields[i] = value
			}
		}
		for _, pair := range fc27Renamed {
			set(pair[0], get(pair[1]))
		}
		for _, col := range fc27SameColumns {
			set(col, get(col))
		}

		first, last := get("first_name"), get("last_name")
		set("short_name", shortName(get("common_name"), first, last))
		set("long_name", strings.TrimSpace(first+" "+last))
		set("player_url", "https://sofifa.com/player/"+id)
		positions := append([]string{primary}, strings.Fields(get("alternate_positions"))...)
		set("player_positions", strings.Join(positions, ", "))

		if primary == "GK" {
			stats.Goalkeepers++
			set("goalkeeping_speed", get("defending")) // a keeper's SPD facet
		} else {
			for _, pair := range fc27OutfieldFacets {
				set(pair[0], get(pair[1]))
			}
		}
		rows = append(rows, row{id: numericID, overall: overall, fields: fields})
	}
	stats.Players = len(rows)

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].overall != rows[j].overall {
			return rows[i].overall > rows[j].overall
		}
		return rows[i].id < rows[j].id
	})

	// CRLF matches the FIFA 22 dataset, so carried-over legend rows stay
	// byte-identical.
	w := csv.NewWriter(out)
	w.UseCRLF = true
	if err := w.Write(header); err != nil {
		return stats, err
	}
	for _, r := range rows {
		if err := w.Write(r.fields); err != nil {
			return stats, err
		}
	}
	if err := w.WriteAll(legends); err != nil {
		return stats, err
	}
	return stats, w.Error()
}

// readLegacy returns the sofifa header and the legend-card rows of the
// current dataset.
func readLegacy(legacy io.Reader) ([]string, [][]string, error) {
	r := csv.NewReader(legacy)
	header, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("legacy: read header: %w", err)
	}
	if len(header) == 0 || strings.TrimPrefix(header[0], "\ufeff") != "sofifa_id" {
		return nil, nil, fmt.Errorf("legacy: first column must be sofifa_id")
	}
	var legends [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("legacy: %w", err)
		}
		if strings.HasPrefix(rec[0], allocation.LegendCardPrefix) {
			legends = append(legends, rec)
		}
	}
	if len(legends) == 0 {
		return nil, nil, fmt.Errorf("legacy: no %s-prefixed legend cards found", allocation.LegendCardPrefix)
	}
	return header, legends, nil
}

// shortName mirrors sofifa's short_name: EA's single-name form when it has
// one ("Vinícius Jr."), otherwise first initial and surname ("K. Mbappé").
func shortName(common, first, last string) string {
	if common != "" {
		return common
	}
	if first == "" {
		return last
	}
	initial, _ := utf8.DecodeRuneInString(first)
	return string(initial) + ". " + last
}
