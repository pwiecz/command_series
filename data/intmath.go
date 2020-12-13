package data

import "math/rand"

func Abs(i int) int {
	if i >= 0 {
		return i
	}
	return -i
}

func Min(i0, i1 int) int {
	if i0 <= i1 {
		return i0
	}
	return i1
}
func Clamp(v, min, max int) int {
	if v <= min {
		return min
	}
	if v >= max {
		return max
	}
	return v
}

func Sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

func InRange(v, min, max int) bool {
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
