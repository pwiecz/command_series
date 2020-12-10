package data

import "fmt"
import "path/filepath"
import "strings"

import "github.com/pwiecz/command_series/atr"

type Game int

const (
	Crusade  Game = 0
	Decision Game = 1
	Conflict Game = 2
)

func FilenameToGame(filename string) Game {
	baseName := filepath.Base(filename)
	if strings.HasPrefix(baseName, "DDAY.") ||
		strings.HasPrefix(baseName, "RACE.") ||
		strings.HasPrefix(baseName, "ARNHEM.") ||
		strings.HasPrefix(baseName, "BULGE.") ||
		strings.HasPrefix(baseName, "CAMPAIGN.") {
		return Crusade
	}
	if strings.HasPrefix(baseName, "SIDI.") ||
		strings.HasPrefix(baseName, "CRUSADER.") ||
		strings.HasPrefix(baseName, "GAZALA.") ||
		strings.HasPrefix(baseName, "FIRST.") ||
		strings.HasPrefix(baseName, "HALFA.") {
		return Decision
	}
	if strings.HasPrefix(baseName, "DINBINFU.") ||
		strings.HasPrefix(baseName, "IADRANG.") ||
		strings.HasPrefix(baseName, "KHESANH.") ||
		strings.HasPrefix(baseName, "FISHOOK.") ||
		strings.HasPrefix(baseName, "EASTER.") {
		return Conflict
	}
	panic(fmt.Errorf("Cannot infer game from the filename: %s", filename))
}

func DetectGame(diskimage atr.SectorReader) (Game, error) {
	files, err := atr.GetDirectory(diskimage)
	if err != nil {
		return Game(0), fmt.Errorf("Cannot list contents of the disk image (%v)", err)
	}

	var game Game
	var foundScenarioFiles bool
	for _, file := range files {
		if strings.HasSuffix(file.Name, ".SCN") {
			scenarioGame := FilenameToGame(file.Name)
			if foundScenarioFiles && scenarioGame != game {
				return Game(0), fmt.Errorf("Mismatched game files found %v and %v", game, scenarioGame)
			}
			game = scenarioGame
			foundScenarioFiles = true
		}
	}
	if !foundScenarioFiles {
		return Game(0), fmt.Errorf("No game files found")
	}
	return game, nil

}
