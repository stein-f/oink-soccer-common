package soccer

import (
	"math/rand"
	"time"

	"github.com/mroth/weightedrand"
)

func GetAllInjuries() []Injury {
	return injuries
}

var injuryWeightsDefaults = []weightedrand.Choice{
	{Item: false, Weight: 40},
	{Item: true, Weight: 1},
}

var injuryWeightsInjuryPronePlayers = []weightedrand.Choice{
	{Item: false, Weight: 20},
	{Item: true, Weight: 1},
}

func ApplyInjury(chancesDefaults []weightedrand.Choice, chancesInjuryProne []weightedrand.Choice, isInjuryProne bool, randSource *rand.Rand) (Injury, bool) {
	choices := chancesDefaults
	if isInjuryProne {
		choices = chancesInjuryProne
	}
	chooser, err := weightedrand.NewChooser(choices...)
	if err != nil {
		return Injury{}, false
	}
	isInjured := chooser.PickSource(randSource).(bool)
	if !isInjured {
		return Injury{}, false
	}

	var injuryChoices []weightedrand.Choice
	for _, i := range GetAllInjuries() {
		injuryChoices = append(injuryChoices, weightedrand.Choice{Item: i, Weight: i.Weight})
	}
	injuryChooser, err := weightedrand.NewChooser(injuryChoices...)
	if err != nil {
		return Injury{}, false
	}
	injury := injuryChooser.PickSource(randSource).(Injury)
	return injury, true
}

var injuries = []Injury{
	// low severity
	{
		Severity:       InjurySeverityLow,
		StatsReduction: 0.95,
		MinDays:        1,
		MaxDays:        1,
		Name:           "Minor sprain",
		Description:    "Overstretched a ligament performing an unsuccessful tackle.",
		Weight:         100,
	},
	{
		Severity:       InjurySeverityLow,
		StatsReduction: 0.95,
		MinDays:        1,
		MaxDays:        1,
		Name:           "Squirrel Scare",
		Description:    "Spooked by a squirrel running onto the field, leading to a comical but unfortunate tumble.",
		Weight:         100,
	},
	{
		Severity:       InjurySeverityLow,
		StatsReduction: 0.95,
		MinDays:        1,
		MaxDays:        1,
		Name:           "Pie Burn",
		Description:    "Out for a game after trying to eat pie too quickly during half-time match and burning the roof of their mouth.",
		Weight:         100,
	}, {
		Severity:       InjurySeverityLow,
		StatsReduction: 0.95,
		MinDays:        1,
		MaxDays:        1,
		Name:           "Laugh Attack",
		Description:    "Couldn't stop laughing after a teammate's joke and ended up with a side stitch.",
		Weight:         100,
	}, {
		Severity:       InjurySeverityLow,
		StatsReduction: 0.95,
		MinDays:        1,
		MaxDays:        1,
		Name:           "Dance-Off Defeat",
		Description:    "Suffered a minor ego bruise and twisted ankle during an impromptu pre-match dance-off.",
		Weight:         100,
	},
	// medium severity
	{
		Severity:       InjurySeverityMid,
		StatsReduction: 0.9,
		MinDays:        2,
		MaxDays:        3,
		Name:           "Hamstring strain",
		Description:    "Minor hamstring tear after sprinting to catch up with a breakaway.",
		Weight:         25,
	},
	{
		Severity:       InjurySeverityMid,
		StatsReduction: 0.9,
		MinDays:        2,
		MaxDays:        3,
		Name:           "Concussion",
		Description:    "Head injury after a collision with a teammate during a header.",
		Weight:         25,
	},
	{
		Severity:       InjurySeverityMid,
		StatsReduction: 0.9,
		MinDays:        2,
		MaxDays:        3,
		Name:           "Mismatched Boots",
		Description:    "Wore two left boots to the game, resulting in blisters and confused running.",
		Weight:         25,
	},
	// high severity
	{
		Severity:       InjurySeverityHigh,
		StatsReduction: 0.85,
		MinDays:        3,
		MaxDays:        5,
		Name:           "Achilles Tendon Rupture",
		Description:    "Achilles tendon rupture after a sudden acceleration to chase down a ball.",
		Weight:         10,
	},
	{
		Severity:       InjurySeverityHigh,
		StatsReduction: 0.85,
		MinDays:        3,
		MaxDays:        5,
		Name:           "High-five fail",
		Description:    "Missed a high-five and accidentally poked themselves in the eye.",
		Weight:         5,
	},
}

type Injury struct {
	Severity       InjurySeverity `json:"severity"`
	MinDays        int            `json:"min_days"`
	MaxDays        int            `json:"max_days"`
	Name           string         `json:"name"`
	StatsReduction float64        `json:"stats_reduction"`
	Description    string         `json:"description"`
	Weight         uint           `json:"weight"`
}

type InjuryEvent struct {
	TeamID   string    `json:"team_id"`
	PlayerID string    `json:"player_id"`
	Expires  time.Time `json:"expires"`
	Injury   Injury    `json:"injury"`
}
