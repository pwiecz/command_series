package lib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwiecz/command_series/atr"
)

func readTestData(t *testing.T, filename string, scenario int) (*GameData, *ScenarioData, error) {
	t.Helper()
	dir := os.Getenv("COMMAND_SERIES_TEST_DATA")
	if dir == "" {
		dir = ".."
	}
	atrFile, err := os.Open(filepath.Join(dir, filename))
	if os.IsNotExist(err) && os.Getenv("COMMAND_SERIES_TEST_DATA") == "" {
		t.Skipf("Missing %s; set COMMAND_SERIES_TEST_DATA to the disk-image directory", filename)
	}
	if err != nil {
		return nil, nil, err
	}
	defer atrFile.Close()
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
