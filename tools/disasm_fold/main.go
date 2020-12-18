package main

import "bufio"
import "os"
import "github.com/pwiecz/command_series/lib"

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
