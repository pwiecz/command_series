package main

import "bufio"
import "fmt"
import "os"

import "github.com/pwiecz/command_series/lib"

func main() {
	reader := bufio.NewReader(os.Stdin)
	opcodes, err := lib.ReadOpcodes(reader)
	if err != nil {
		panic(err)
	}
	for _, opcode := range opcodes {
		fmt.Println(opcode.String())
	}
}
