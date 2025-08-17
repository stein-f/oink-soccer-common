package allocation

import (
	"math"
	"math/rand"
	"os"
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
	SoFIFAID          string `csv:"sofifa_id"`
	PlayerURL         string `csv:"player_url"`
	ShortName         string `csv:"short_name"`
	LongName          string `csv:"long_name"`
	ClubPosition      string `csv:"club_position"`
	PlayerPositions   string `csv:"player_positions"`
	SkillMoves        string `csv:"skill_moves"`
	PlayerTags        string `csv:"player_tags"`
	PlayerTraits      string `csv:"player_traits"`
	Shooting          int    `csv:"shooting"`
	Passing           int    `csv:"passing"`
	Defending         int    `csv:"defending"`
	Goalkeeping       int    `csv:"goalkeeping_handling"`
	GoalkeepingDiving int    `csv:"goalkeeping_diving"`
	Mentality         int    `csv:"mentality_composure"`
	Overall           int    `csv:"overall"`

	// speed
	Pace         int `csv:"pace"`
	SprintSpeed  int `csv:"movement_sprint_speed"`
	Acceleration int `csv:"movement_acceleration"`

	// physical
	MentalityAggression int    `csv:"mentality_aggression"`
	WorkRate            string `csv:"work_rate"`
	Stamina             int    `csv:"power_stamina"`
	Strength            int    `csv:"power_strength"`
}

func (r *Record) CalculateOverallPhysicalRating() int {
	physicality := r.CalculatePhysicalityRating()
	speed := r.CalculateSpeed()
	position := r.GetPosition()
	if position == soccer.PlayerPositionGoalkeeper {
		speed = r.GoalkeepingDiving
		physicality = r.GoalkeepingDiving
	}
	return int(math.Ceil((float64(physicality)+float64(speed))/2) * 1.06)
}

func (r *Record) CalculatePhysicalityRating() int {
	physicalRating := float64(r.MentalityAggression)*0.34 +
		float64(r.Stamina)*0.33 +
		float64(r.Strength)*0.33

	result := int(physicalRating + 0.5)
	if result > 100 {
		result = 100
	}
	if result < 55 {
		result = 55
	}

	return result
}

func (r *Record) CalculateSpeed() int {
	speed := float64(r.SprintSpeed)*0.3 +
		float64(r.Acceleration)*0.4 +
		float64(r.Pace)*0.3
	result := int(speed + 0.5)
	if result > 100 {
		result = 100
	}
	if result < 55 {
		result = 55
	}
	return result
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
		SpeedRating:      normalizeRating(randSource, r.CalculateSpeed()),
		PhysicalRating:   normalizeRating(randSource, r.CalculateOverallPhysicalRating()),
		GoalkeeperRating: normalizeRating(randSource, r.Goalkeeping),
		DefenseRating:    normalizeRating(randSource, r.Defending),
		ControlRating:    normalizeRating(randSource, r.Passing),
		AttackRating:     normalizeRating(randSource, r.Shooting),
		AggressionRating: normalizeRating(randSource, r.MentalityAggression),
		Position:         r.GetPosition(),
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

func findPlayerLevel(overallRating int) soccer.PlayerLevel {
	for level, rng := range PlayerLevelBands {
		if overallRating >= rng[0] && overallRating <= rng[1] {
			return level
		}
	}
	return soccer.PlayerLevelAmateur
}

func normalizeRating(r *rand.Rand, rating int) int {
	if rating == 0 {
		minVal := 5
		maxVal := 30
		return r.Intn(maxVal-minVal+1) + minVal
	}
	return rating
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
