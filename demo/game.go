package main

import "github.com/hajimehoshi/ebiten"
import "github.com/pwiecz/command_series/data"

type Game struct {
	subGame          ebiten.Game
	gameDirname      string
	sprites          data.Sprites
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

	isLeftButtonDown bool
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

func (g *Game) onGameLoaded(scenarios []data.Scenario, sprites data.Sprites, terrainMap data.Map, generic data.Generic, hexes data.Hexes) {
	g.scenarios = scenarios
	g.sprites = sprites
	g.terrainMap = terrainMap
	g.generic = generic
	g.hexes = hexes
	g.subGame = NewScenarioSelection(g.scenarios, g.sprites.IntroFont, g.onScenarioSelected)
}
func (g *Game) onScenarioSelected(selectedScenario int) {
	g.selectedScenario = selectedScenario
	g.subGame = NewVariantsLoading(g.scenarios[selectedScenario], g)
}

func (g *Game) Update(screen *ebiten.Image) error {
	if g.subGame != nil {
		return g.subGame.Update(screen)
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
	return 320, 192
}

func (g *Game) showString(screen *ebiten.Image, text string, x, y int) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(2., 1.)
	opts.GeoM.Translate(float64(x), float64(y))
	for _, c := range text {
		glyph := g.sprites.IntroFont.Glyph(c)

		screen.DrawImage(glyph, opts)
		opts.GeoM.Translate(8, 0)
	}
}

func (g *Game) showSpriteString(screen *ebiten.Image, text []byte, x, y int) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(x), float64(y))
	for _, b := range text {
		var sprite *ebiten.Image
		if int(b) < len(g.sprites.IntroSprites) {
			sprite = g.sprites.IntroSprites[b]
		} else {
			sprite = g.sprites.IntroSprites[0]
		}

		screen.DrawImage(sprite, opts)
		opts.GeoM.Translate(4, 0)
	}
}
