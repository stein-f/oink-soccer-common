package soccer

import "math"

// ScalingFunction is a Gompertz function that takes a rating and returns a biased rating.
// For example, a rating of 100 will much more likely to score than a rating of 75.
// See scaling_test.go for a table of inputs and outputs.
func ScalingFunction(input int) int {
	// Scale the input to a range of 0 to 1
	scaledRating := float64(input) / 100

	// Calculate a probability using the Gompertz function
	probability := 1 - math.Exp(-math.Exp(4.5*scaledRating-4.5))

	// Ensure the probability is within the range of 0 to 1
	probability = math.Max(0, math.Min(1, probability))

	return int(probability * 100)
}

// NormalizeRating ensures the rating value is between 0-100
// To normalize, divide the sum of the weighted ratings by the product of the total number of players and the maximum possible score.
func NormalizeRating(sumOfRatings int, maxTotalRating int) int {
	if maxTotalRating == 0 {
		// Avoid division by zero
		return 0
	}

	normalizedRating := (float64(sumOfRatings) / float64(maxTotalRating)) * 100.0

	if normalizedRating < 0 {
		return 0
	} else if normalizedRating > 100 {
		return 100
	}

	return int(normalizedRating)
}
