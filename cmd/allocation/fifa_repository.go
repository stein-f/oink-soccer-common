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
