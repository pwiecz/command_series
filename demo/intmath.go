package main

func AbsInt(i int) int {
	if i >= 0 {
		return i
	}
	return -i
}

func ClampInt(v, min, max int) int {
	if v <= min {
		return min
	}
	if v >= max {
		return max
	}
	return v
}

func SignInt(v int) int {
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
