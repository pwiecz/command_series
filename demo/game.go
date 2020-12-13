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
	game             data.Game
	sprites          data.Sprites
	icons            data.Icons
	scenarios        []data.Scenario
	terrainMap       data.Map
	generic          data.Generic
	hexes            data.Hexes
	selectedScenario int
	variants         []data.Variant
	generals         [2][]data.General
	terrain          data.Terrain
	scenarioData     data.ScenarioData
	selectedVariant  int
	units            [2][]data.Unit

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

func (g *Game) onGameLoaded(game data.Game, scenarios []data.Scenario, sprites data.Sprites, icons data.Icons, terrainMap data.Map, generic data.Generic, hexes data.Hexes) {
	g.game = game
	g.scenarios = scenarios
	g.sprites = sprites
	g.icons = icons
	g.terrainMap = terrainMap
	g.generic = generic
	g.hexes = hexes
	g.subGame = NewScenarioSelection(g.scenarios, g.sprites.IntroFont, g.onScenarioSelected)
}
func (g *Game) onScenarioSelected(selectedScenario int) {
	g.selectedScenario = selectedScenario
	g.subGame = NewVariantsLoading(g.scenarios[selectedScenario], g)
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
