package soccer

import (
	"math/rand"

	"github.com/mroth/weightedrand"
)

var chanceTypeWeights = map[ChanceType]uint{
	ChanceTypeCorner:         3,
	ChanceTypeCross:          4,
	ChanceTypeOpenPlay:       7,
	ChanceTypeGoalKeeperShot: 1,
	ChanceTypeLongRange:      4,
	ChanceTypeFreeKick:       3,
	ChanceTypePenalty:        2,
}

// deterministic order for building choices (avoids map iteration randomness)
var chanceTypeOrder = []ChanceType{
	ChanceTypeCorner,
	ChanceTypeCross,
	ChanceTypeOpenPlay,
	ChanceTypeGoalKeeperShot,
	ChanceTypeLongRange,
	ChanceTypeFreeKick,
	ChanceTypePenalty,
}

func DetermineChanceType(previousChanceType *ChanceType, randSource *rand.Rand) (ChanceType, error) {
	var choices []weightedrand.Choice
	for _, chanceType := range chanceTypeOrder {
		if weight, ok := chanceTypeWeights[chanceType]; ok {
			choices = append(choices, weightedrand.Choice{
				Item:   chanceType,
				Weight: weight,
			})
		}
	}

	chooser, err := weightedrand.NewChooser(choices...)
	if err != nil {
		return "", err
	}

	chanceType := chooser.PickSource(randSource).(ChanceType)

	if previousChanceType != nil && chanceType == *previousChanceType {
		return pickChanceTypeDifferentToPrevious(*previousChanceType, randSource, choices, chooser)
	}

	return chanceType, nil
}

func pickChanceTypeDifferentToPrevious(previousChanceType ChanceType, randSource *rand.Rand, choices []weightedrand.Choice, chooser *weightedrand.Chooser) (ChanceType, error) {
	choicesWithoutPrevious := []weightedrand.Choice{}
	for _, choice := range choices {
		if choice.Item != previousChanceType {
			choicesWithoutPrevious = append(choicesWithoutPrevious, choice)
		}
	}

	chooser, err := weightedrand.NewChooser(choicesWithoutPrevious...)
	if err != nil {
		return "", err
	}

	return chooser.PickSource(randSource).(ChanceType), nil
}
