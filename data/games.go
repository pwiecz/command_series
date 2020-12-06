package data

import "fmt"
import "path/filepath"
import "strings"

type Game int

const (
	Crusade  Game = 0
	Decision Game = 1
	Conflict Game = 2
)

func FileNameToGame(filename string) Game {
	baseName := filepath.Base(filename)
	if strings.HasPrefix(baseName, "DDAY") ||
		strings.HasPrefix(baseName, "RACE") ||
		strings.HasPrefix(baseName, "ARNHEM") ||
		strings.HasPrefix(baseName, "BULGE") ||
		strings.HasPrefix(baseName, "CAMPAIGN") {
		return Crusade
	}
	if strings.HasPrefix(baseName, "SIDI") ||
		strings.HasPrefix(baseName, "CRUSADER") ||
		strings.HasPrefix(baseName, "GAZALA") ||
		strings.HasPrefix(baseName, "FIRST") ||
		strings.HasPrefix(baseName, "HALFA") {
		return Decision
	}
	if strings.HasPrefix(baseName, "DINBINFU") ||
		strings.HasPrefix(baseName, "IADRANG") ||
		strings.HasPrefix(baseName, "KHESANH") ||
		strings.HasPrefix(baseName, "FISHOOK") ||
		strings.HasPrefix(baseName, "EASTER") {
		return Conflict
	}
	panic(fmt.Errorf("Cannot infer game from the filename: %s", filename))
}
