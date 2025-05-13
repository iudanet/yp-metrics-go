package utils

import (
	"math"
	"math/rand/v2"
	"time"
)

func GetRandomNumber() float64 {
	seed := uint64(time.Now().UnixNano())
	r := rand.New(rand.NewPCG(seed, 2))
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
