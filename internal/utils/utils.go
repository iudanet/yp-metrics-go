package utils

import (
	"math/rand/v2"
)

func GetRandomNumber() float64 {
	r := rand.New(rand.NewPCG(1, 2))
	return r.Float64()
}
