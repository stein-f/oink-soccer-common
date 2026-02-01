package allocation

import (
	"math/rand"
	"sort"

	"github.com/mroth/weightedrand"
	"github.com/pkg/errors"
	soccer "github.com/stein-f/oink-soccer-common"
)

const (
	aggressionThreshold         = 80
	aggressivePlayersUpperBound = 86
)

type PlayerProfile struct {
	Asset      EligibleAsset `json:"asset"`
	FifaPlayer FifaPlayer    `json:"fifa_player"`
}

var PlayerLevelBands = map[soccer.PlayerLevel][]int{
	soccer.PlayerLevelLegendary:        {87, 100},
	soccer.PlayerLevelWorldClass:       {80, 86},
	soccer.PlayerLevelProfessional:     {70, 79},
	soccer.PlayerLevelSemiProfessional: {55, 69},
	soccer.PlayerLevelAmateur:          {0, 54},
}

type PlayersLookup struct {
	Goalkeepers       map[soccer.PlayerLevel][]FifaPlayer
	Defenders         map[soccer.PlayerLevel][]FifaPlayer
	Midfielders       map[soccer.PlayerLevel][]FifaPlayer
	Attackers         map[soccer.PlayerLevel][]FifaPlayer
	AggressivePlayers []FifaPlayer
	Rand              *rand.Rand
}

func (p *PlayersLookup) AddPlayers(profiles []FifaPlayer) {
	for _, profile := range profiles {
		p.AddPlayer(profile)
	}
}

func (p *PlayersLookup) AddPlayer(profile FifaPlayer) {
	overallRating := profile.PlayerAttributes.GetOverallRating()
	tiers := getPlayerLevels(overallRating)
	for _, tier := range tiers {
		switch profile.PlayerAttributes.PrimaryPosition {
		case soccer.PlayerPositionGoalkeeper:
			p.Goalkeepers[tier] = append(p.Goalkeepers[tier], profile)
		case soccer.PlayerPositionDefense:
			p.Defenders[tier] = append(p.Defenders[tier], profile)
		case soccer.PlayerPositionMidfield:
			p.Midfielders[tier] = append(p.Midfielders[tier], profile)
		case soccer.PlayerPositionAttack:
			p.Attackers[tier] = append(p.Attackers[tier], profile)
		}
	}
	if profile.PlayerAttributes.AggressionRating >= aggressionThreshold &&
		profile.PlayerAttributes.GetOverallRating() <= aggressivePlayersUpperBound {
		p.AggressivePlayers = append(p.AggressivePlayers, profile)
	}
}

// Normalize sorts all internal player slices to a deterministic order so selections
// from these slices are reproducible. Sorting key: OverallRating (desc), then PlayerID (asc).
func (p *PlayersLookup) Normalize() {
	sortPlayerSlice := func(s []FifaPlayer) {
		sort.Slice(s, func(i, j int) bool {
			a := s[i]
			b := s[j]
			if a.PlayerAttributes.OverallRating != b.PlayerAttributes.OverallRating {
				return a.PlayerAttributes.OverallRating > b.PlayerAttributes.OverallRating
			}
			return a.PlayerID < b.PlayerID
		})
	}

	// iterate and sort deterministically by sorting the map keys first
	var gkKeys []string
	for k := range p.Goalkeepers {
		gkKeys = append(gkKeys, string(k))
	}
	sort.Strings(gkKeys)
	for _, ks := range gkKeys {
		lvl := soccer.PlayerLevel(ks)
		sortPlayerSlice(p.Goalkeepers[lvl])
	}

	var defKeys []string
	for k := range p.Defenders {
		defKeys = append(defKeys, string(k))
	}
	sort.Strings(defKeys)
	for _, ks := range defKeys {
		lvl := soccer.PlayerLevel(ks)
		sortPlayerSlice(p.Defenders[lvl])
	}

	var midKeys []string
	for k := range p.Midfielders {
		midKeys = append(midKeys, string(k))
	}
	sort.Strings(midKeys)
	for _, ks := range midKeys {
		lvl := soccer.PlayerLevel(ks)
		sortPlayerSlice(p.Midfielders[lvl])
	}

	var attKeys []string
	for k := range p.Attackers {
		attKeys = append(attKeys, string(k))
	}
	sort.Strings(attKeys)
	for _, ks := range attKeys {
		lvl := soccer.PlayerLevel(ks)
		sortPlayerSlice(p.Attackers[lvl])
	}

	// also sort aggressive players list for determinism
	sortPlayerSlice(p.AggressivePlayers)
}

func getPlayerLevels(overallRating int) []soccer.PlayerLevel {
	// Extract keys and sort deterministically so behaviour is reproducible
	var levels []soccer.PlayerLevel
	var keys []string
	for k := range PlayerLevelBands {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	for _, ks := range keys {
		tier := soccer.PlayerLevel(ks)
		rnge := PlayerLevelBands[tier]
		if overallRating >= rnge[0] && overallRating <= rnge[1] {
			levels = append(levels, tier)
		}
	}
	return levels
}

func (p *PlayersLookup) GetRandomPlayer(position soccer.PlayerPosition, asset EligibleAsset) (FifaPlayer, error) {
	if asset.EligibleAssetTier == EligibleAssetTierAggressive {
		return randElementInSlice(p.Rand, p.AggressivePlayers), nil
	}

	levelProbabilities := tierToPlayerLevelProbability[asset.EligibleAssetTier]

	// Build deterministic weighted choices from the map of level probabilities
	levelChoicesRaw := soccer.BuildChoicesFromMapNumberKeys(levelProbabilities)
	// Convert items in the raw choices back to soccer.PlayerLevel type
	levelChoices := make([]weightedrand.Choice, 0, len(levelChoicesRaw))
	for _, c := range levelChoicesRaw {
		// c.Item was stored as the original key type (soccer.PlayerLevel)
		levelChoices = append(levelChoices, weightedrand.Choice{Item: c.Item.(soccer.PlayerLevel), Weight: c.Weight})
	}

	chooser, err := weightedrand.NewChooser(levelChoices...)
	if err != nil {
		return FifaPlayer{}, errors.Wrap(err, "failed to get player position")
	}
	playerLevel := chooser.PickSource(p.Rand).(soccer.PlayerLevel)
	switch position {
	case soccer.PlayerPositionGoalkeeper:
		return randElementInSlice(p.Rand, p.Goalkeepers[playerLevel]), nil
	case soccer.PlayerPositionDefense:
		return randElementInSlice(p.Rand, p.Defenders[playerLevel]), nil
	case soccer.PlayerPositionMidfield:
		return randElementInSlice(p.Rand, p.Midfielders[playerLevel]), nil
	case soccer.PlayerPositionAttack:
		return randElementInSlice(p.Rand, p.Attackers[playerLevel]), nil
	}
	return FifaPlayer{}, errors.New("invalid player position")
}

var tierToPlayerLevelProbability = map[EligibleAssetTier]map[soccer.PlayerLevel]int{
	EligibleAssetTierS: {
		soccer.PlayerLevelLegendary:        3,
		soccer.PlayerLevelWorldClass:       45,
		soccer.PlayerLevelProfessional:     52,
		soccer.PlayerLevelSemiProfessional: 0,
		soccer.PlayerLevelAmateur:          0,
	},
	EligibleAssetTierA: {
		soccer.PlayerLevelLegendary:        3,
		soccer.PlayerLevelWorldClass:       37,
		soccer.PlayerLevelProfessional:     60,
		soccer.PlayerLevelSemiProfessional: 0,
		soccer.PlayerLevelAmateur:          0,
	},
	EligibleAssetTierB: {
		soccer.PlayerLevelLegendary:        0,
		soccer.PlayerLevelWorldClass:       8,
		soccer.PlayerLevelProfessional:     42,
		soccer.PlayerLevelSemiProfessional: 40,
		soccer.PlayerLevelAmateur:          10,
	},
	EligibleAssetTierC: {
		soccer.PlayerLevelLegendary:        0,
		soccer.PlayerLevelWorldClass:       4,
		soccer.PlayerLevelProfessional:     30,
		soccer.PlayerLevelSemiProfessional: 46,
		soccer.PlayerLevelAmateur:          20,
	},
}

func NewPlayersLookup(randSource *rand.Rand, players []FifaPlayer) *PlayersLookup {
	lookup := &PlayersLookup{
		Goalkeepers: make(map[soccer.PlayerLevel][]FifaPlayer),
		Defenders:   make(map[soccer.PlayerLevel][]FifaPlayer),
		Midfielders: make(map[soccer.PlayerLevel][]FifaPlayer),
		Attackers:   make(map[soccer.PlayerLevel][]FifaPlayer),
		Rand:        randSource,
	}
	lookup.AddPlayers(players)
	// ensure deterministic ordering within buckets
	lookup.Normalize()
	return lookup
}

func BuildPlayersLookup(randSource *rand.Rand, repository FifaPlayerRepository) (*PlayersLookup, error) {
	players, err := repository.GetAllPlayers()
	if err != nil {
		return &PlayersLookup{}, err
	}
	return NewPlayersLookup(randSource, players), nil
}

type EligibleAssetTier string

const (
	EligibleAssetTierS          EligibleAssetTier = "Tier S"
	EligibleAssetTierA          EligibleAssetTier = "Tier A"
	EligibleAssetTierB          EligibleAssetTier = "Tier B"
	EligibleAssetTierC          EligibleAssetTier = "Tier C"
	EligibleAssetTierAggressive EligibleAssetTier = "Tier Aggressive"
)

type EligibleAssetOrigin string

const (
	EligibleAssetOriginAlgorand EligibleAssetOrigin = "Algorand"
)

type EligibleAsset struct {
	PlayerID          string            `json:"player_id"`
	Name              string            `json:"name"`
	EligibleAssetTier EligibleAssetTier `json:"tier"`
}

func (e EligibleAsset) GetOrigin() EligibleAssetOrigin {
	return EligibleAssetOriginAlgorand
}

func randElementInSlice(r *rand.Rand, slice []FifaPlayer) FifaPlayer {
	if len(slice) == 0 {
		return FifaPlayer{}
	}
	return slice[r.Intn(len(slice))]
}

func GetRandomPosition(randSource *rand.Rand) (soccer.PlayerPosition, error) {
	playerChoices := []weightedrand.Choice{
		{Item: soccer.PlayerPositionGoalkeeper, Weight: 15},
		{Item: soccer.PlayerPositionDefense, Weight: 20},
		{Item: soccer.PlayerPositionMidfield, Weight: 20},
		{Item: soccer.PlayerPositionAttack, Weight: 20},
	}
	chooser, err := weightedrand.NewChooser(playerChoices...)
	if err != nil {
		return "", errors.Wrap(err, "failed to get player position")
	}
	return chooser.PickSource(randSource).(soccer.PlayerPosition), nil
}
