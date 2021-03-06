package main

import (
	"bufio"
	"os"

	"github.com/pwiecz/command_series/tools/lib"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	opcodes, err := lib.ReadOpcodes(reader)
	if err != nil {
		panic(err)
	}

	f := &lib.FoldingDecoder{}
	for _, opcode := range opcodes {
		f.Apply(opcode)
	}
	f.DumpStack()
}
