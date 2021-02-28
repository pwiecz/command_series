package lib

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/pwiecz/command_series/atr"
)

func readTestData(filename string, scenario int) (*GameData, *ScenarioData, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, nil, err
	}
	atrFilename := filepath.Join(currentUser.HomeDir, "command_series", filename)
	atrFile, err := os.Open(atrFilename)
	if err != nil {
		return nil, nil, err
	}
	fsys, err := atr.NewAtrFS(atrFile)
	if err != nil {
		return nil, nil, err
	}
	gameData, err := LoadGameData(fsys)
	if err != nil {
		return nil, nil, err
	}
	scenarioData, err := LoadScenarioData(fsys, gameData.Scenarios[scenario].FilePrefix)
	if err != nil {
		return nil, nil, err
	}
	return gameData, scenarioData, nil
}
