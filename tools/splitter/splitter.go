package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func readNumber(reader *bufio.Reader) int {
	number := 0
	for {
		b, err := reader.ReadByte()
		if err != nil {
			panic(err)
		}
		if b == 0x9b {
			return number
		}
		if b < '0' || b > '9' {
			panic(int(b))
		}
		number = number*10 + int(b-'0')
	}
}
func readHeader(reader *bufio.Reader) []int {
	numOffsets := readNumber(reader)
	var offsets []int
	// There are numOffsets+2 numbers following the numOffsets number.
	for i := 0; i < numOffsets+2; i++ {
		offsets = append(offsets, readNumber(reader))
	}
	return offsets[:len(offsets)-1]
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	offsets := readHeader(reader)
	buf, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	for i := 1; i < len(offsets); i++ {
		fmt.Printf("%d: %d - %d\n", i-1, offsets[i-1], offsets[i])
		program := buf[offsets[i-1]:offsets[i]]
		if os.WriteFile(fmt.Sprintf("prg_%d.sid", i-1), program, 0644) != nil {
			panic(err)
		}
	}
}
