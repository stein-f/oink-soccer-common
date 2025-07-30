package main

import (
	"fmt"
	"github.com/mroth/weightedrand"
	soccer "github.com/stein-f/oink-soccer-common"
	"math/rand"
)

func main() {
	// Test different aggression levels to see their impact on injury rates
	testAggressionLevels := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	// Use a fixed seed for reproducible results
	randSource := rand.New(rand.NewSource(42))

	// Define the injury weights (same as in the soccer package)
	injuryWeightsDefaults := []weightedrand.Choice{
		{Item: false, Weight: 30},
		{Item: true, Weight: 1},
	}

	injuryWeightsInjuryPronePlayers := []weightedrand.Choice{
		{Item: false, Weight: 15},
		{Item: true, Weight: 1},
	}

	fmt.Println("Testing injury rates at different aggression levels")
	fmt.Println("==================================================")
	fmt.Println("Aggression | Injuries (per 10,000) | Relative to Baseline")
	fmt.Println("--------------------------------------------------")

	// Run 10,000 iterations for each aggression level
	iterations := 10_000
	baselineInjuries := 0

	for _, aggression := range testAggressionLevels {
		// Reset the random source for fair comparison
		randSource = rand.New(rand.NewSource(42))

		injuryCount := 0
		for i := 0; i < iterations; i++ {
			_, isInjured := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, aggression, randSource)
			if isInjured {
				injuryCount++
			}
		}

		// Save baseline (aggression 0) for relative comparison
		if aggression == 0 {
			baselineInjuries = injuryCount
		}

		relativeToBaseline := float64(injuryCount) / float64(baselineInjuries)

		fmt.Printf("%9d | %22d | %20.2f\n", aggression, injuryCount, relativeToBaseline)
	}

	// Specifically compare aggression 80 vs 90
	fmt.Println("\nFocusing on the difference between aggression 80 and 90:")

	// Reset the random sources
	randSource80 := rand.New(rand.NewSource(42))
	randSource90 := rand.New(rand.NewSource(42))

	injuries80 := 0
	injuries90 := 0

	for i := 0; i < iterations; i++ {
		_, isInjured80 := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 80, randSource80)
		if isInjured80 {
			injuries80++
		}

		_, isInjured90 := soccer.ApplyInjury(injuryWeightsDefaults, injuryWeightsInjuryPronePlayers, false, 90, randSource90)
		if isInjured90 {
			injuries90++
		}
	}

	percentIncrease := (float64(injuries90) - float64(injuries80)) / float64(injuries80) * 100

	fmt.Printf("Aggression 80: %d injuries\n", injuries80)
	fmt.Printf("Aggression 90: %d injuries\n", injuries90)
	fmt.Printf("Difference: %d injuries\n", injuries90-injuries80)
	fmt.Printf("Percent increase: %.2f%%\n", percentIncrease)
}
