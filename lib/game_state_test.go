package lib

import (
	"math/rand"
	"testing"
)

func createTestGameState(filename string, scenarioNum, variantNum int, options Options, messageSync *MessageSync, t *testing.T) *GameState {
	rand := rand.New(rand.NewSource(1))
	gameData, scenarioData, err := readTestData(filename, scenarioNum)
	if err != nil {
		t.Fatal("Error reading game data,", err)
	}

	return NewGameState(rand, gameData, scenarioData, scenarioNum, variantNum, 0, &options, messageSync)
}

func TestRegression_Basic(t *testing.T) {
	messageSync := NewMessageSync()
	gameState := createTestGameState("crusade.atr", 0, 0, DefaultOptions(), messageSync, t)
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

	expectedNumMessages := 1040
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 64
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 0, 2, 0
	result, balance, rank := gameState.FinalResults(0)
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Errorf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}

func TestRegression_Side1Player(t *testing.T) {
	messageSync := NewMessageSync()
	options := DefaultOptions()
	options.AlliedCommander = Computer
	options.GermanCommander = Player
	gameState := createTestGameState("crusade.atr", 0, 0, options, messageSync, t)
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

	expectedNumMessages := 4606
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 101
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 8, 2, 8
	result, balance, rank := gameState.FinalResults(0)
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Errorf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}

func TestRegression_TwoPlayers(t *testing.T) {
	messageSync := NewMessageSync()
	options := DefaultOptions()
	options.GermanCommander = Player
	gameState := createTestGameState("decision.atr", 2, 1, options, messageSync, t)
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

	expectedNumMessages := 15379
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 116
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 0, 2, 0
	result, balance, rank := gameState.FinalResults(0)
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Errorf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}

func TestRegression_RegressionPanicInCampaign(t *testing.T) {
	messageSync := NewMessageSync()
	options := DefaultOptions()
	options.AlliedCommander = Computer
	options.GermanCommander = Computer
	gameState := createTestGameState("crusade.atr", 4, 0, options, messageSync, t)
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

	expectedNumMessages := 433202
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 3324
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 4, 2, 4
	result, balance, rank := gameState.FinalResults(0)
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Errorf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}

func TestRegression_Conflict_FullIntelligence(t *testing.T) {
	messageSync := NewMessageSync()
	options := DefaultOptions()
	options.AlliedCommander = Computer
	options.GermanCommander = Computer
	options.Intelligence = Full
	gameState := createTestGameState("conflict.atr", 4, 1, options, messageSync, t)
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

	expectedNumMessages := 43043
	if numMessages != expectedNumMessages {
		t.Errorf("Expecting %d messages, got %d", expectedNumMessages, numMessages)
	}
	expectedNumMessagesFromUnit := 1278
	if numMessagesFromUnit != expectedNumMessagesFromUnit {
		t.Errorf("Expecting %d messages from a unit, got %d", expectedNumMessagesFromUnit, numMessagesFromUnit)
	}

	expectedResult, expectedBalance, expectedRank := 3, 2, 3
	result, balance, rank := gameState.FinalResults(0)
	if result != expectedResult || balance != expectedBalance || rank != expectedRank {
		t.Errorf("Expecting %d,%d,%d final results, got %d,%d,%d",
			expectedResult, expectedBalance, expectedRank, result, balance, rank)
	}
}
