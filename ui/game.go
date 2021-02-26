package ui

import (
	"fmt"
	"io/fs"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/oto"
	"github.com/pwiecz/command_series/lib"
)

type SubGame interface {
	Update() error
	Draw(screen *ebiten.Image)
}
type Game struct {
	subGame          SubGame
	fsys             fs.FS
	rand             *rand.Rand
	gameData         *lib.GameData
	selectedScenario int
	scenarioData     *lib.ScenarioData
	selectedVariant  int
	options          *lib.Options

	otoContext  *oto.Context
	audioPlayer *AudioPlayer
}

func NewGame(fsys fs.FS, rand *rand.Rand) (*Game, error) {
	otoContext, err := oto.NewContext(44100, 2 /* num channels */, 1 /* num bytes per sample */, 4096 /* buffer size */)
	if err != nil {
		return nil, fmt.Errorf("Cannot create Oto context (%v)", err)
	}
	game := &Game{
		fsys:             fsys,
		rand:             rand,
		selectedScenario: -1,
		selectedVariant:  -1,
		otoContext:       otoContext,
		audioPlayer:      NewAudioPlayer(otoContext)}
	game.subGame = NewGameLoading(fsys, game.onGameLoaded)
	return game, nil
}

func (g *Game) onGameLoaded(gameData *lib.GameData) {
	g.gameData = gameData
	g.subGame = NewScenarioSelection(g.gameData.Scenarios, g.gameData.Sprites.IntroFont, g.onScenarioSelected)
}
func (g *Game) onRestartGame() {
	g.subGame = NewGameLoading(g.fsys, g.onGameLoaded)
}
func (g *Game) onScenarioSelected(selectedScenario int) {
	g.selectedScenario = selectedScenario
	g.subGame = NewScenarioLoading(g.fsys, g.gameData.Scenarios[selectedScenario], g.gameData.Sprites.IntroFont, g.onScenarioLoaded)
}
func (g *Game) onScenarioLoaded(scenarioData *lib.ScenarioData) {
	g.scenarioData = scenarioData
	g.subGame = NewVariantSelection(g.scenarioData.Variants, g.gameData.Sprites.IntroFont, g.onVariantSelected)
}
func (g *Game) onVariantSelected(selectedVariant int) {
	if g.gameData.Game == lib.Conflict {
		if g.selectedScenario == 2 && selectedVariant == 4 {
			selectedVariant = 1 + g.rand.Intn(3)
		} else if g.selectedScenario == 3 && selectedVariant == 7 {
			selectedVariant = 1 + g.rand.Intn(6)
		}
	}
	g.selectedVariant = selectedVariant
	g.subGame = NewOptionSelection(g.gameData.Game, g.gameData.Sprites.IntroFont, g.onOptionsSelected)
}
func (g *Game) onOptionsSelected(options *lib.Options) {
	g.options = options
	g.subGame = NewMainScreen(g, g.options, g.audioPlayer, g.rand, g.onGameOver)
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
		screen.Fill(lib.RGBPalette[15])
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 336, 240
}
