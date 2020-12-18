package main

import "image"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

import "github.com/pwiecz/command_series/lib"

type FinalResult struct {
	onRestartGame func()

	text []*Button
}

var resultStrings = []string{"TOTAL DEFEAT", "DECISIVE DEFEAT", "TACTICAL DEFEAT", "MARGINAL DEFEAT", "DISADVANTAGE", "ADVANTAGE", "MARGINAL VICTORY", "TACTICAL VICTORY", "DECISIVE VICTORY", "TOTAL VICTORY"}
var difficultyStrings = []string{"VERY DIFFICULT", "DIFFICULT", "FAIR", "EASY", "VERY EASY"}
var rankStrings = []string{"PRIVATE", "SERGEANT", "LIEUTENANT", "CAPTAIN", "MAJOR", "LIEUTENANT-COLONEL", "COLONEL", "BRIGADIER-GENERAL", "MAJOR-GENERAL", "LIEUTENANT-GENERAL", "FIELD MARSHAL", "SUPREME COMMANDER"}

func NewFinalResult(result, difficulty, rank int, font *lib.Font, onRestartGame func()) *FinalResult {
	text := []*Button{
		NewButton("PRESS ENTER", 184, 40, image.Pt(200, 8), font),
		NewButton("FOR NEW GAME", 216, 64, image.Pt(200, 8), font),
		NewButton("FINAL RESULT: "+resultStrings[result], 56, 112, image.Pt(300, 8), font),
		NewButton("PLAY BALANCE: "+difficultyStrings[difficulty], 56, 123, image.Pt(300, 8), font),
		NewButton("YOUR RANK:    "+rankStrings[rank], 56, 134, image.Pt(300, 8), font)}
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
