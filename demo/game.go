package main

import "fmt"
import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/oto"

import "github.com/pwiecz/command_series/atr"
import "github.com/pwiecz/command_series/data"

type SubGame interface {
	Update() error
	Draw(screen *ebiten.Image)
}
type Game struct {
	subGame          SubGame
	diskimage        atr.SectorReader
	gameData         *data.GameData
	selectedScenario int
	scenarioData     *data.ScenarioData
	selectedVariant  int
	options          data.Options

	otoContext  *oto.Context
	audioPlayer *AudioPlayer
}

func NewGame(filename string) (*Game, error) {
	diskimage, err := atr.NewAtrSectorReader(filename)
	if err != nil {
		return nil, fmt.Errorf("Cannot open atr image file %s (%v)", filename, err)
	}
	otoContext, err := oto.NewContext(44100, 2 /* num channels */, 1 /* num bytes per sample */, 4096 /* buffer size */)
	if err != nil {
		return nil, fmt.Errorf("Cannot create Oto context (%v)", err)
	}
	game := &Game{
		diskimage:        diskimage,
		selectedScenario: -1,
		selectedVariant:  -1,
		otoContext:       otoContext,
		audioPlayer:      NewAudioPlayer(otoContext)}
	game.subGame = NewGameLoading(diskimage, game.onGameLoaded)
	return game, nil
}

func (g *Game) onGameLoaded(gameData *data.GameData) {
	g.gameData = gameData
	g.subGame = NewScenarioSelection(g.gameData.Scenarios, g.gameData.Sprites.IntroFont, g.onScenarioSelected)
}
func (g *Game) onRestartGame() {
	g.subGame = NewGameLoading(g.diskimage, g.onGameLoaded)
}
func (g *Game) onScenarioSelected(selectedScenario int) {
	g.selectedScenario = selectedScenario
	g.subGame = NewScenarioLoading(g.diskimage, g.gameData.Scenarios[selectedScenario], g.gameData.Sprites.IntroFont, g.onScenarioLoaded)
}
func (g *Game) onScenarioLoaded(scenarioData *data.ScenarioData) {
	g.scenarioData = scenarioData
	g.subGame = NewVariantSelection(g.scenarioData.Variants, g.gameData.Sprites.IntroFont, g.onVariantSelected)
}
func (g *Game) onVariantSelected(selectedVariant int) {
	g.selectedVariant = selectedVariant
	g.subGame = NewOptionSelection(g.gameData.Game, g.gameData.Sprites.IntroFont, g.onOptionsSelected)
}
func (g *Game) onOptionsSelected(options data.Options) {
	g.options = options
	g.subGame = NewShowMap(g, g.options, g.audioPlayer, g.onGameOver)
}
func (g *Game) onGameOver(result, balance, rank int) {
	g.subGame = NewFinalResult(result, balance, rank, g.gameData.Sprites.IntroFont, g.onRestartGame)
}

func (g *Game) Update() error {
	if g.subGame != nil {
		return g.subGame.Update()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.subGame != nil {
		g.subGame.Draw(screen)
	} else {
		screen.Fill(data.RGBPalette[15])
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 336, 240
}
