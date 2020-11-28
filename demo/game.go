package main

import "github.com/hajimehoshi/ebiten"
import "github.com/pwiecz/command_series/data"

type Game struct {
	subGame          ebiten.Game
	gameDirname      string
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
}

func NewGame(gameDirname string) *Game {
	game := &Game{
		gameDirname:      gameDirname,
		selectedScenario: -1,
		selectedVariant:  -1,
	}
	game.subGame = NewGameLoading(gameDirname, game.onGameLoaded)
	return game
}

func (g *Game) onGameLoaded(scenarios []data.Scenario, sprites data.Sprites, icons data.Icons, terrainMap data.Map, generic data.Generic, hexes data.Hexes) {
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
	g.subGame = NewVariantsLoading(g.scenarios[selectedScenario], g, g.sprites.IntroFont)
}

func (g *Game) Update() error {
	if g.subGame != nil {
		return g.subGame.Update()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.subGame != nil {
		if gameWithDraw, ok := g.subGame.(interface{ Draw(*ebiten.Image) }); ok {
			gameWithDraw.Draw(screen)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	if g.subGame != nil {
		return g.subGame.Layout(outsideWidth, outsideHeight)
	}
	return 600, 450
}
