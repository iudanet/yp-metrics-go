package utils

import (
	"math/rand/v2"
	"time"
)

func GetRandomNumber() float64 {
	seed := uint64(time.Now().UnixNano())
	r := rand.New(rand.NewPCG(seed, 2))
	return r.Float64()
}
