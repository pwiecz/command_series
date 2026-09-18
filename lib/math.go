package lib

import (
	"math/rand"

	"golang.org/x/exp/constraints"
)

type SignedNumber interface {
	constraints.Signed | constraints.Float
}

func Abs[T SignedNumber](i T) T {
	if i >= T(0) {
		return i
	}
	return -i
}

// Works only for positive arguments.
func DivRoundUp(n, d int) int {
	return (n + (d - 1)) / d
}

func Min[T constraints.Ordered](i0, i1 T) T {
	if i0 <= i1 {
		return i0
	}
	return i1
}

func Max[T constraints.Ordered](i0, i1 T) T {
	if i0 >= i1 {
		return i0
	}
	return i1
}

func Clamp[T constraints.Ordered](v, min, max T) T {
	if v <= min {
		return min
	}
	if v >= max {
		return max
	}
	return v
}

func Sign[T SignedNumber](v T) int {
	if v > T(0) {
		return 1
	}
	if v < T(0) {
		return -1
	}
	return 0
}

func InRange[T constraints.Ordered](v, min, max T) bool {
	if v < min || v >= max {
		return false
	}
	return true
}

func Rand(n int, rnd *rand.Rand) int {
	if n == 0 {
		return 0
	}
	return rnd.Intn(n)
}
