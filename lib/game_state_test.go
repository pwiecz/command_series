package lib

import "math/rand"
import "path"
import "os/user"
import "testing"

import "github.com/pwiecz/command_series/atr"

func CreateTestGameState(filename string, scenarioNum, variantNum int, options Options, messageSync *MessageSync) *GameState {
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
	return NewGameState(rand, gameData, scenarioData, scenarioNum, variantNum, 0, options, messageSync)
}

func TestRegression_Basic(t *testing.T) {
	messageSync := NewMessageSync()
	gameState := CreateTestGameState("crusade.atr", 0, 0, DefaultOptions(), messageSync)
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

	var numMessages, numMessagesFromUnit int
	for {
		update := messageSync.GetUpdate()
		numMessages++
		if _, ok := update.(MessageFromUnit); ok {
			numMessagesFromUnit++
		}
		if _, ok := update.(GameOver); ok {
			messageSync.Stop()
			break
		}
	}

	expectedNumMessages := 1508
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 75
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 6, 2, 6
	result, balance, rank := gameState.FinalResults()
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Fatalf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}

func TestRegression_Side1Player(t *testing.T) {
	messageSync := NewMessageSync()
	options := DefaultOptions()
	options.AlliedCommander = Computer
	options.GermanCommander = Player
	gameState := CreateTestGameState("crusade.atr", 0, 0, options, messageSync)
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

	var numMessages, numMessagesFromUnit int
	for {
		update := messageSync.GetUpdate()
		numMessages++
		if _, ok := update.(MessageFromUnit); ok {
			numMessagesFromUnit++
		}
		if _, ok := update.(GameOver); ok {
			messageSync.Stop()
			break
		}
	}

	expectedNumMessages := 4724
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 85
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 1, 2, 1
	result, balance, rank := gameState.FinalResults()
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Fatalf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}

func TestRegression_TwoPlayers(t *testing.T) {
	messageSync := NewMessageSync()
	options := DefaultOptions()
	options.GermanCommander = Player
	gameState := CreateTestGameState("decision.atr", 2, 1, options, messageSync)
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

	var numMessages, numMessagesFromUnit int
	for {
		update := messageSync.GetUpdate()
		numMessages++
		if _, ok := update.(MessageFromUnit); ok {
			numMessagesFromUnit++
		}
		if _, ok := update.(GameOver); ok {
			messageSync.Stop()
			break
		}
		if numMessages == 100 {
			gameState.SwitchSides()
		}
	}

	expectedNumMessages := 15410
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 324
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 0, 2, 0
	result, balance, rank := gameState.FinalResults()
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Fatalf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}
