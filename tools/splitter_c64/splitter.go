package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	buf, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	started := false
	progIdx := 0
	var startPosition int
	var currentProgram []byte
	for i, b := range buf {
		if !started && b != 0x18 {
			continue
		}
		started = true
		currentProgram = append(currentProgram, b)
		if b == 0x0c {
			fmt.Printf("%d\n", i+1-len(currentProgram)-startPosition)
			ioutil.WriteFile(fmt.Sprintf("prg_%d.sid", progIdx), currentProgram, 0644)
			currentProgram = nil
			progIdx++
		}
	}
}
