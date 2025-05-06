package utils

import (
	"math"
	"math/rand/v2"
)

func GetRandomNumber() float64 {
	r := rand.New(rand.NewPCG(1, 2))
	return r.Float64()
}

func Round(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)
	if frac >= 0.5 {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow
}
