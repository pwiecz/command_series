package data

import "math/rand"
import "path"
import "os/user"
import "testing"

import "github.com/pwiecz/command_series/atr"

func CreateTestGameState(filename string, scenarioNum, variantNum int, messageSync *MessageSync) *GameState {
	rand := rand.New(rand.NewSource(1))
	currentUser, err := user.Current()
	if err != nil {
		return nil
	}
	atrFile := path.Join(currentUser.HomeDir, "command_series", filename)
	diskimage, err := atr.NewAtrSectorReader(atrFile)
	if err != nil {
		return nil
	}
	gameData, err := LoadGameData(diskimage)
	if err != nil {
		return nil
	}
	scenarioData, err := LoadScenarioData(diskimage, gameData.Scenarios[scenarioNum].FilePrefix)
	if err != nil {
		return nil
	}
	var options Options
	options.AlliedCommander = Player
	options.GermanCommander = Computer
	options.Intelligence = Limited
	options.GameBalance = 2
	return NewGameState(rand, gameData, scenarioData, scenarioNum, variantNum, 0, options, messageSync)
}

func TestRegression(t *testing.T) {
	messageSync := NewMessageSync()
	gameState := CreateTestGameState("crusade.atr", 0, 0, messageSync)
	if gameState == nil {
		t.FailNow()
	}
	go func() {
		if !messageSync.Wait() {
			return
		}
		if !gameState.Init() {
			return
		}
		for gameState.Update() {
		}
	}()

	var messages []interface{}
	for {
		update := messageSync.GetUpdate()
		messages = append(messages, update)
		if _, ok := update.(GameOver); ok {
			messageSync.Stop()
			break
		}
	}
	expectedNumMessages := 1463
	if len(messages) != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, len(messages))
	}
	expectedResult, expectedBalance, expectedRank := 6, 2, 6
	result, balance, rank := gameState.FinalResults()
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Fatalf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}
