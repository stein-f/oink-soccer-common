package main

import (
	"fmt"
	"math"
)

func PerUseDecay(targetEnd float64, uses int) float64 {
	if uses <= 0 {
		return 1.0
	}
	return math.Pow(targetEnd, 1.0/float64(uses))
}

func main() {
	fmt.Println("To reach 0.35 in 25 uses:", PerUseDecay(0.35, 25))
	fmt.Println("To reach 0.50 in 25 uses:", PerUseDecay(0.50, 25))
	fmt.Println("To reach 0.60 in 25 uses:", PerUseDecay(0.60, 25))
}
