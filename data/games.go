package data

import "fmt"
import "path/filepath"
import "os"
import "strings"

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

func DetectGame(dirname string) (Game, error) {
	dir, err := os.Open(dirname)
	if err != nil {
		return Game(0), fmt.Errorf("Cannot open directory %s, %v\n", dirname, err)
	}
	defer dir.Close()
	dirInfo, err := dir.Stat()
	if err != nil {
		return Game(0), fmt.Errorf("Cannot get info about directory %s, %v\n", dirname, err)
	}
	if !dirInfo.IsDir() {
		return Game(0), fmt.Errorf("%s is not a directory\n", dirname)
	}

	filenames, err := dir.Readdirnames(0)
	if err != nil {
		return Game(0), fmt.Errorf("Cannot list directory %s, %v\n", dirname, err)
	}

	var game Game
	var foundScenarioFiles bool
	for _, filename := range filenames {
		if strings.HasSuffix(filename, ".SCN") {
			scenarioGame := FilenameToGame(filename)
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
