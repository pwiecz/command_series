package ui

func FindByte(s []byte, b byte) int {
	for i, v := range s {
		if v == b {
			return i
		}
	}
	return -1
}
