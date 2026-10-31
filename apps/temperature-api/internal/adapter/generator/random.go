package generator

import "math/rand"

// Random generates uniformly distributed values.
type Random struct{}

// NewRandom returns a random generator backed by the standard library.
func NewRandom() *Random {
	return &Random{}
}

// Float64 returns a random value in the range [min, max).
func (g *Random) Float64(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}
