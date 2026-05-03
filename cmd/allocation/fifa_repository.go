package allocation

import (
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/gocarina/gocsv"
	soccer "github.com/stein-f/oink-soccer-common"
)

type FifaPlayerRepository interface {
	GetAllPlayers() ([]FifaPlayer, error)
}

type FifaPlayersRepository struct {
	RandSource *rand.Rand
}

func (f FifaPlayersRepository) GetAllPlayers() ([]FifaPlayer, error) {
	file, err := os.ReadFile("cmd/allocation/fifa_players_22.csv")
	if err != nil {
		return nil, err
	}
	var records []Record
	err = gocsv.UnmarshalBytes(file, &records)
	if err != nil {
		return nil, err
	}
	var players []FifaPlayer
	for _, record := range records {
		players = append(players, record.ToDomain(f.RandSource))
	}
	return players, nil
}

type Record struct {
	SoFIFAID            string `csv:"sofifa_id"`
	PlayerURL           string `csv:"player_url"`
	ShortName           string `csv:"short_name"`
	LongName            string `csv:"long_name"`
	ClubPosition        string `csv:"club_position"`
	PlayerPositions     string `csv:"player_positions"`
	SkillMoves          string `csv:"skill_moves"`
	WorkRate            string `csv:"work_rate"`
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

	// Specialist columns — feed v2's per-chance attributes (Heading, Composure,
	// Technique, Finishing, Tackling). Each is consumed by ToDomain and lost
	// here for v1 (which ignores the resulting fields on PlayerAttributes).
	AttackingFinishing       int `csv:"attacking_finishing"`
	AttackingHeadingAccuracy int `csv:"attacking_heading_accuracy"`
	AttackingCrossing        int `csv:"attacking_crossing"`
	PowerJumping             int `csv:"power_jumping"`
	PowerStamina             int `csv:"power_stamina"`
	PowerLongShots           int `csv:"power_long_shots"`
	SkillCurve               int `csv:"skill_curve"`
	SkillFKAccuracy          int `csv:"skill_fk_accuracy"`
	MovementAcceleration     int `csv:"movement_acceleration"`
	MovementSprintSpeed      int `csv:"movement_sprint_speed"`
	DefendingStandingTackle  int `csv:"defending_standing_tackle"`
	DefendingSlidingTackle   int `csv:"defending_sliding_tackle"`
	MentalityInterceptions   int `csv:"mentality_interceptions"`
}

type FifaPlayer struct {
	PlayerID         string
	PlayerAttributes soccer.PlayerAttributes
}

func (r *Record) ToDomain(randSource *rand.Rand) FifaPlayer {
	var tags []string
	if r.PlayerTags != "" {
		tokens := strings.Split(r.PlayerTags, ",")
		for _, token := range tokens {
			withoutSpaces := strings.TrimSpace(token)
			withoutHash := strings.ReplaceAll(withoutSpaces, "#", "")
			tags = append(tags, withoutHash)
		}
	}
	if r.PlayerTraits != "" {
		tokens := strings.Split(r.PlayerTraits, ",")
		for _, token := range tokens {
			withoutSpaces := strings.TrimSpace(token)
			withoutHash := strings.ReplaceAll(withoutSpaces, "#", "")
			tags = append(tags, withoutHash)
		}
	}
	attributes := soccer.PlayerAttributes{
		SpeedRating:      normalizePace(r.Pace, r.Overall),
		GoalkeeperRating: normalizeRating(randSource, r.Goalkeeping),
		DefenseRating:    normalizeRating(randSource, r.Defending),
		ControlRating:    normalizeRating(randSource, r.Passing),
		AttackRating:     normalizeRating(randSource, r.Shooting),
		AggressionRating: normalizeRating(randSource, r.MentalityAggression),
		PrimaryPosition:  r.GetPosition(),
		Positions:        r.GetPositions(),
		Tag:              tags,
		BasedOnPlayer:    r.ShortName,
		BasedOnPlayerURL: r.PlayerURL,

		// v2 specialist attributes. Stored on the JSON output so v2's engine
		// can resolve corners by Heading, penalties by Composure, etc. v1's
		// engine sees these as inert.
		Pace:      r.Pace,
		Recovery:  averagePositive(r.PowerStamina, r.MentalityInterceptions),
		WorkRate:  r.PowerStamina,
		Finishing: r.AttackingFinishing,
		Heading:   averagePositive(r.AttackingHeadingAccuracy, r.PowerJumping),
		Technique: averagePositive(r.SkillCurve, r.SkillFKAccuracy, r.PowerLongShots),
		Composure: r.Mentality, // mentality_composure
		Tackling:  averagePositive(r.DefendingStandingTackle, r.DefendingSlidingTackle, r.MentalityInterceptions),
	}
	overallRating := attributes.GetOverallRating()
	attributes.OverallRating = overallRating
	attributes.PlayerLevel = findPlayerLevel(overallRating)
	return FifaPlayer{
		PlayerID:         r.SoFIFAID,
		PlayerAttributes: attributes,
	}
}

func (r *Record) GetPosition() soccer.PlayerPosition {
	position := getPosition(r.ClubPosition)
	if position != "" {
		return position
	}

	playerPositions := strings.Split(r.PlayerPositions, ",")
	return getPosition(playerPositions[0])
}

func (r *Record) GetPositions() []soccer.PlayerPosition {
	positions := make(map[soccer.PlayerPosition]struct{})
	positions[getPosition(r.ClubPosition)] = struct{}{}
	for _, position := range strings.Split(r.PlayerPositions, ",") {
		positions[getPosition(position)] = struct{}{}
	}
	result := make([]soccer.PlayerPosition, 0, len(positions))
	for position := range positions {
		result = append(result, position)
	}

	if len(result) == 0 {
		// for backwards compatability fall back to primary position
		return []soccer.PlayerPosition{r.GetPosition()}
	}

	// Sort deterministically by string value so callers see stable ordering
	sort.Slice(result, func(i, j int) bool {
		return string(result[i]) < string(result[j])
	})

	return result
}

func findPlayerLevel(overallRating int) soccer.PlayerLevel {
	// Iterate PlayerLevelBands deterministically by sorting keys
	var keys []string
	for k := range PlayerLevelBands {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	for _, ks := range keys {
		level := soccer.PlayerLevel(ks)
		rng := PlayerLevelBands[level]
		if overallRating >= rng[0] && overallRating <= rng[1] {
			return level
		}
	}
	return soccer.PlayerLevelAmateur
}

func normalizeRating(r *rand.Rand, rating int) int {
	if rating == 0 {
		// return random rating between 5 and 50
		min := 5
		max := 30
		return r.Intn(max-min+1) + min
	}
	return rating
}

func normalizePace(pace int, overall int) int {
	if pace == 0 {
		return overall
	}
	return pace
}

// averagePositive averages the supplied values, ignoring zeros (FIFA leaves
// many specialist columns at 0 for players outside the relevant role — e.g.
// goalkeepers have no `attacking_heading_accuracy`). Returns 0 if every input
// is 0, which lets the v2 Effective* accessors fall back to their composite.
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

func getPosition(position string) soccer.PlayerPosition {
	switch position {
	case "GK":
		return soccer.PlayerPositionGoalkeeper
	case "CB", "LCB", "RCB":
		return soccer.PlayerPositionDefense
	case "LB", "LWB":
		return soccer.PlayerPositionDefense
	case "RB", "RWB":
		return soccer.PlayerPositionDefense
	case "CDM", "LDM", "RDM":
		return soccer.PlayerPositionDefense
	case "CM", "LCM", "RCM":
		return soccer.PlayerPositionMidfield
	case "CAM", "LAM", "RAM":
		return soccer.PlayerPositionMidfield
	case "LM", "LW", "RM", "RW":
		return soccer.PlayerPositionMidfield
	case "CF", "LF", "RF", "ST", "LS", "RS":
		return soccer.PlayerPositionAttack
	default:
		return ""
	}
}
