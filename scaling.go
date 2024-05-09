package soccer

import (
	_ "embed"
	"math"

	"github.com/gocarina/gocsv"
)

//go:embed scaling.csv
var scalingData []byte

type scalingRow struct {
	Rating uint64  `csv:"rating"`
	Scaled float64 `csv:"scaled"`
}

var scalingRecords map[uint64]float64

func init() {
	scalingRecords = make(map[uint64]float64)
	var rows []scalingRow
	if err := gocsv.UnmarshalBytes(scalingData, &rows); err != nil {
		panic(err)
	}
	for _, row := range rows {
		scalingRecords[row.Rating] = row.Scaled
	}
}

// ScalingFunction is a function that aims to give higher rated players a more significant advantage over lower rated players.
// If we took the raw gotPlayer ratings, then a lower skilled gotPlayer on 80 would have almost the same ability as a higher skilled gotPlayer 84, given the random nature of the game.
// This function aims to give higher rated players a more significant advantage over lower rated players.
// We use a function that grows more rapidly as the input increases.
// y = ax^b
// where y is the scaled rating, x is the original rating (0-100) and a and b are constants that can be adjusted to change the shape of the curve.
func ScalingFunction(originalRating float64) float64 {
	normalizedRating := uint64(math.Round(originalRating))

	if normalizedRating > 100 {
		return 100
	}

	if normalizedRating <= 0 {
		return 1
	}

	rating, ok := scalingRecords[normalizedRating]
	if !ok {
		return 1
	}
	return rating
}
