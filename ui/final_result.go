package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/pwiecz/command_series/lib"
)

type FinalResult struct {
	onRestartGame func()
	text          []*Label
}

var resultStrings = []string{"TOTAL DEFEAT", "DECISIVE DEFEAT", "TACTICAL DEFEAT", "MARGINAL DEFEAT", "DISADVANTAGE", "ADVANTAGE", "MARGINAL VICTORY", "TACTICAL VICTORY", "DECISIVE VICTORY", "TOTAL VICTORY"}
var difficultyStrings = []string{"VERY DIFFICULT", "DIFFICULT", "FAIR", "EASY", "VERY EASY"}
var rankStrings = []string{"PRIVATE", "SERGEANT", "LIEUTENANT", "CAPTAIN", "MAJOR", "LIEUTENANT-COLONEL", "COLONEL", "BRIGADIER-GENERAL", "MAJOR-GENERAL", "LIEUTENANT-GENERAL", "FIELD MARSHAL", "SUPREME COMMANDER"}

func NewFinalResult(result, difficulty, rank int, font *lib.Font, onRestartGame func()) *FinalResult {
	text := []*Label{
		NewLabel("PRESS ENTER", 184, 40, 200, 8, font),
		NewLabel("FOR NEW GAME", 216, 64, 200, 8, font),
		NewLabel("FINAL RESULT: "+resultStrings[result], 56, 112, 300, 8, font),
		NewLabel("PLAY BALANCE: "+difficultyStrings[difficulty], 56, 123, 300, 8, font),
		NewLabel("YOUR RANK:    "+rankStrings[rank], 56, 134, 300, 8, font)}
	for _, label := range text {
		label.SetBackgroundColor(15)
	}

	return &FinalResult{
		onRestartGame: onRestartGame,
		text:          text}
}

func (s *FinalResult) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.onRestartGame()
	}
	return nil
}

func (s *FinalResult) Draw(screen *ebiten.Image) {
	screen.Fill(lib.RGBPalette[15])
	for _, text := range s.text {
		text.Draw(screen)
	}
}
