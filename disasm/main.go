package main

import "bufio"
import "fmt"
import "os"

import "github.com/pwiecz/command_series/lib"

type scopeType int

const (
	IF  scopeType = 0
	FOR scopeType = 1
)

func printIndent(indent int) {
	for i := 0; i < indent; i++ {
		fmt.Print("  ")
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	opcodes, err := lib.ReadOpcodes(reader)
	if err != nil {
		panic(err)
	}
	var scopes []scopeType
	for _, opcode := range opcodes {
		switch opcode.(type) {
		case lib.IfGreaterThenZero:
			printIndent(len(scopes))
			scopes = append(scopes, IF)
		case lib.IfZero:
			printIndent(len(scopes))
			scopes = append(scopes, IF)
		case lib.IfNotEqual:
			printIndent(len(scopes))
			scopes = append(scopes, IF)
		case lib.IfSignEq:
			printIndent(len(scopes))
			scopes = append(scopes, IF)
		case lib.IfCmp:
			printIndent(len(scopes))
			scopes = append(scopes, IF)
		case lib.Fi:
			scopes = scopes[:len(scopes)-1]
			printIndent(len(scopes))
		case lib.FiAll:
			for i := 0; i < len(scopes); i++ {
				if scopes[i] == IF {
					scopes = scopes[:i]
					break
				}
			}
			printIndent(len(scopes))
		case lib.For:
			printIndent(len(scopes))
			scopes = append(scopes, FOR)
		case lib.Done:
			scopes = scopes[:len(scopes)-1]
			printIndent(len(scopes))
		default:
			printIndent(len(scopes))
		}
		fmt.Println(opcode.String())
	}
}
