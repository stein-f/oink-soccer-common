package soccer

import "math"

const scalingConstant = float64(1.5e-08)

// ScalingFunction is a function that aims to give higher rated players a more significant advantage over lower rated players.
// If we took the raw gotPlayer ratings, then a lower skilled gotPlayer on 80 would have almost the same ability as a higher skilled gotPlayer 84, given the random nature of the game.
// This function aims to give higher rated players a more significant advantage over lower rated players.
// We use a function that grows more rapidly as the input increases.
// y = ax^b
// where y is the scaled rating, x is the original rating (0-100) and a and b are constants that can be adjusted to change the shape of the curve.
func ScalingFunction(originalRating float64) float64 {
	k := 0.1202        // Scaling factor to differentiate player impacts significantly
	baseRating := 55.0 // Mid-point of the rating scale

	// Calculate the impact score using an exponential function
	impactScore := math.Exp(k * (originalRating - baseRating))

	if impactScore > 100 {
		return 100
	}

	if impactScore < 5 {
		return 5
	}

	return impactScore
}
