// Package fifacsv loads the shared FIFA dataset (cmd/allocation/
// fifa_players_22.csv at the repo root) and maps each row to an allocation
// Candidate. Both the allocation CLI and the analysis commands import it so
// the mapping semantics exist in exactly one place.
package fifacsv

import (
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/gocarina/gocsv"
	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
)

// fifaRecord is one row of fifa_players_22.csv. The column set and the
// attribute mapping below mirror the v1 loader (cmd/allocation/fifa_repository.go)
// so both pipelines draw on the same source data; only the target type differs
// (v2 soccer.PlayerAttributes, which folds v1's Pace/Recovery into SpeedRating).
type fifaRecord struct {
	SoFIFAID            string `csv:"sofifa_id"`
	PlayerURL           string `csv:"player_url"`
	ShortName           string `csv:"short_name"`
	LongName            string `csv:"long_name"`
	ClubPosition        string `csv:"club_position"`
	PlayerPositions     string `csv:"player_positions"`
	WorkRateText        string `csv:"work_rate"`
	PlayerTags          string `csv:"player_tags"`
	PlayerTraits        string `csv:"player_traits"`
	Pace                int    `csv:"pace"`
	Shooting            int    `csv:"shooting"`
	Passing             int    `csv:"passing"`
	Defending           int    `csv:"defending"`
	Goalkeeping         int    `csv:"goalkeeping_handling"`
	Mentality           int    `csv:"mentality_composure"`
	MentalityAggression int    `csv:"mentality_aggression"`
	Overall             int    `csv:"overall"`

	// Specialist columns feed v2's per-chance attributes.
	AttackingFinishing       int `csv:"attacking_finishing"`
	AttackingHeadingAccuracy int `csv:"attacking_heading_accuracy"`
	PowerJumping             int `csv:"power_jumping"`
	PowerStamina             int `csv:"power_stamina"`
	PowerLongShots           int `csv:"power_long_shots"`
	SkillCurve               int `csv:"skill_curve"`
	SkillFKAccuracy          int `csv:"skill_fk_accuracy"`
	DefendingStandingTackle  int `csv:"defending_standing_tackle"`
	DefendingSlidingTackle   int `csv:"defending_sliding_tackle"`
	MentalityInterceptions   int `csv:"mentality_interceptions"`
}

// LoadCandidates reads the FIFA dataset CSV at path and maps each row to an
// allocation candidate. Missing ratings are filled deterministically from r,
// so the whole candidate set is reproducible from the same block-hash seed.
func LoadCandidates(path string, r *rand.Rand) ([]allocation.Candidate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []fifaRecord
	if err := gocsv.UnmarshalBytes(raw, &records); err != nil {
		return nil, err
	}
	candidates := make([]allocation.Candidate, 0, len(records))
	for _, rec := range records {
		candidates = append(candidates, allocation.Candidate{
			ID:         rec.SoFIFAID,
			Name:       rec.ShortName,
			Attributes: rec.toAttributes(r),
		})
	}
	return candidates, nil
}

func (rec fifaRecord) toAttributes(r *rand.Rand) soccer.PlayerAttributes {
	attrs := soccer.PlayerAttributes{
		SpeedRating:      normalizePace(rec.Pace, rec.Overall),
		GoalkeeperRating: normalizeRating(r, rec.Goalkeeping),
		DefenseRating:    normalizeRating(r, rec.Defending),
		ControlRating:    normalizeRating(r, rec.Passing),
		AttackRating:     normalizeRating(r, rec.Shooting),
		AggressionRating: normalizeRating(r, rec.MentalityAggression),
		PrimaryPosition:  rec.primaryPosition(),
		Positions:        rec.positions(),
		Tag:              rec.tags(),
		BasedOnPlayer:    rec.ShortName,
		BasedOnPlayerURL: rec.PlayerURL,

		// v2 specialist attributes (see PlayerAttributes doc for the mapping).
		WorkRate:  rec.PowerStamina,
		Finishing: rec.AttackingFinishing,
		Heading:   averagePositive(rec.AttackingHeadingAccuracy, rec.PowerJumping),
		Technique: averagePositive(rec.SkillCurve, rec.SkillFKAccuracy, rec.PowerLongShots),
		Composure: rec.Mentality,
		Tackling:  averagePositive(rec.DefendingStandingTackle, rec.DefendingSlidingTackle, rec.MentalityInterceptions),
	}
	// OverallRating is the FIFA `overall` column verbatim. v1 derived a
	// position-weighted overall here; since the 2026-06 OVR alignment the
	// game displays FIFA overall everywhere, so the allocation bands (and
	// the published tier odds) must be measured on the same number.
	attrs.OverallRating = rec.Overall
	attrs.PlayerLevel = playerLevelFor(rec.SoFIFAID, rec.Overall)
	return attrs
}

func (rec fifaRecord) tags() []string {
	var tags []string
	for _, src := range []string{rec.PlayerTags, rec.PlayerTraits} {
		if src == "" {
			continue
		}
		for _, token := range strings.Split(src, ",") {
			token = strings.ReplaceAll(strings.TrimSpace(token), "#", "")
			if token != "" {
				tags = append(tags, token)
			}
		}
	}
	return tags
}

func (rec fifaRecord) primaryPosition() soccer.PlayerPosition {
	if pos := mapPosition(rec.ClubPosition); pos != "" {
		return pos
	}
	return mapPosition(strings.Split(rec.PlayerPositions, ",")[0])
}

func (rec fifaRecord) positions() []soccer.PlayerPosition {
	seen := map[soccer.PlayerPosition]struct{}{}
	if pos := mapPosition(rec.ClubPosition); pos != "" {
		seen[pos] = struct{}{}
	}
	for _, p := range strings.Split(rec.PlayerPositions, ",") {
		if pos := mapPosition(p); pos != "" {
			seen[pos] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return []soccer.PlayerPosition{rec.primaryPosition()}
	}
	result := make([]soccer.PlayerPosition, 0, len(seen))
	for pos := range seen {
		result = append(result, pos)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// playerLevelFor buckets an overall rating into its level using the bands
// exported by the allocation package (single source of truth). Legendary is
// reserved for the curated legend cards (allocation.LegendCardPrefix ids) —
// they alone enter the legend lottery. Real players whose FIFA overall
// reaches the legendary band are capped to World Class, the top drawable
// level: "legend" means a historical player, not a rating, so a real 87+
// player is simply the best of the World Class pool. The pool must bucket
// on this level and not re-derive from the raw overall, or these players
// fall into the undrawable Legendary band and leave the game entirely.
func playerLevelFor(id string, overall int) soccer.PlayerLevel {
	if strings.HasPrefix(id, allocation.LegendCardPrefix) {
		return soccer.PlayerLevelLegendary
	}
	return allocation.LevelFor(overall)
}

// normalizeRating fills FIFA's many zeroed columns (e.g. goalkeeping for an
// outfielder) with a low random rating so unrelated stats don't read as 0.
func normalizeRating(r *rand.Rand, rating int) int {
	if rating == 0 {
		const min, max = 5, 30
		return r.Intn(max-min+1) + min
	}
	return rating
}

func normalizePace(pace, overall int) int {
	if pace == 0 {
		return overall
	}
	return pace
}

// averagePositive averages the supplied values, ignoring zeros. Returns 0 when
// every input is 0, which lets v2's Effective* accessors fall back to a composite.
func averagePositive(values ...int) int {
	var sum, n int
	for _, v := range values {
		if v > 0 {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / n
}

func mapPosition(position string) soccer.PlayerPosition {
	switch strings.TrimSpace(position) {
	case "GK":
		return soccer.PlayerPositionGoalkeeper
	case "CB", "LCB", "RCB", "LB", "LWB", "RB", "RWB", "CDM", "LDM", "RDM":
		return soccer.PlayerPositionDefense
	case "CM", "LCM", "RCM", "CAM", "LAM", "RAM", "LM", "LW", "RM", "RW":
		return soccer.PlayerPositionMidfield
	case "CF", "LF", "RF", "ST", "LS", "RS":
		return soccer.PlayerPositionAttack
	default:
		return ""
	}
}
