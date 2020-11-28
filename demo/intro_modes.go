package main

import "fmt"
import "image/color"
import "path"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"
import "github.com/pwiecz/command_series/data"

type ScenarioSelection struct {
	buttons          []*Button
	scenarioSelected func(int)
}

func numToKey(n int) ebiten.Key {
	switch n {
	case 0:
		return ebiten.Key0
	case 1:
		return ebiten.Key1
	case 2:
		return ebiten.Key2
	case 3:
		return ebiten.Key3
	case 4:
		return ebiten.Key4
	case 5:
		return ebiten.Key5
	case 6:
		return ebiten.Key6
	case 7:
		return ebiten.Key7
	case 8:
		return ebiten.Key8
	case 9:
		return ebiten.Key9
	}
	panic(fmt.Errorf("No key for num %d", n))
}

func NewScenarioSelection(scenarios []data.Scenario, font *data.Font, scenarioSelected func(int)) *ScenarioSelection {
	buttons := make([]*Button, len(scenarios))
	x, y := 16.0, 16.0
	fontSize := font.Size()
	for i, scenario := range scenarios {
		button := NewButton(fmt.Sprintf("%d: %s", i+1, scenario.Name), x, y, font)
		buttons[i] = button
		y += float64(fontSize.Y)
	}
	return &ScenarioSelection{
		buttons:          buttons,
		scenarioSelected: scenarioSelected}
}

func (s *ScenarioSelection) Update() error {
	for i, button := range s.buttons {
		if button.Update() || (i < 10 && inpututil.IsKeyJustReleased(numToKey(i+1))) {
			s.scenarioSelected(i)
			return nil
		}
	}
	return nil
}
func (s *ScenarioSelection) Draw(screen *ebiten.Image) {
	screen.Fill(color.Gray{255})
	for _, button := range s.buttons {
		button.Draw(screen)
	}
}
func (s *ScenarioSelection) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 400, 300
}

type VariantSelection struct {
	buttons  []*Button
	mainGame *Game
}

func NewVariantSelection(mainGame *Game) *VariantSelection {
	buttons := make([]*Button, len(mainGame.variants))
	x, y := 16.0, 16.0
	fontSize := mainGame.sprites.IntroFont.Size()
	for i, variant := range mainGame.variants {
		button := NewButton(fmt.Sprintf("%d: %s", i+1, variant.Name), x, y, mainGame.sprites.IntroFont)
		buttons[i] = button
		y += float64(fontSize.Y)
	}
	return &VariantSelection{
		buttons:  buttons,
		mainGame: mainGame}
}

func (s *VariantSelection) Update() error {
	for i, button := range s.buttons {
		if button.Update() || (i < 10 && inpututil.IsKeyJustReleased(numToKey(i+1))) {
			s.mainGame.selectedVariant = i
			s.mainGame.subGame = NewVariantLoading(s.mainGame)
			return nil
		}
	}
	return nil
}
func (s *VariantSelection) Draw(screen *ebiten.Image) {
	screen.Fill(color.Gray{255})
	for _, button := range s.buttons {
		button.Draw(screen)
	}
}
func (s *VariantSelection) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 400, 300
}

type GameLoading struct {
	gameDirname  string
	onGameLoaded func([]data.Scenario, data.Sprites, data.Icons, data.Map, data.Generic, data.Hexes)
	loadingDone  chan error
	scenarios    []data.Scenario
	sprites      data.Sprites
	icons        data.Icons
	terrainMap   data.Map
	generic      data.Generic
	hexes        data.Hexes

	turnsLoading int
	loadingRect  *ebiten.Image
}

func NewGameLoading(gameDirname string, onGameLoaded func([]data.Scenario, data.Sprites, data.Icons, data.Map, data.Generic, data.Hexes)) *GameLoading {
	return &GameLoading{
		gameDirname:  gameDirname,
		onGameLoaded: onGameLoaded}
}

func (l *GameLoading) Update() error {
	if l.loadingDone == nil {
		l.loadingDone = make(chan error)
		go func() {
			l.loadingDone <- l.loadGameData()
		}()
	} else {
		select {
		case err := <-l.loadingDone:
			if err != nil {
				return err
			}
			l.onGameLoaded(l.scenarios, l.sprites, l.icons, l.terrainMap, l.generic, l.hexes)
		default:
		}
	}
	l.turnsLoading++
	return nil
}
func (l *GameLoading) Draw(screen *ebiten.Image) {
	if l.loadingRect == nil {
		l.loadingRect = ebiten.NewImage(100, 1)
	}
	if l.turnsLoading%200 < 100 {
		l.loadingRect.Set(l.turnsLoading%100, 0, color.White)
	} else {
		l.loadingRect.Set(l.turnsLoading%100, 0, color.Black)
	}
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(100, 50)
	l.loadingRect.DrawImage(screen, &opts)
}

func (s *GameLoading) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 400, 300
}

func (l *GameLoading) loadGameData() error {
	var err error
	l.scenarios, err = data.ReadScenarios(l.gameDirname)
	if err != nil {
		return fmt.Errorf("Error loading scenarios, %v", err)
	}
	l.sprites, err = data.ReadSprites(l.gameDirname)
	if err != nil {
		return fmt.Errorf("Error loading sprites, %v", err)
	}
	l.icons, err = data.ReadIcons(l.gameDirname)
	if err != nil {
		return fmt.Errorf("Error loading icons, %v", err)
	}
	l.terrainMap, err = data.ReadMap(l.gameDirname)
	if err != nil {
		return fmt.Errorf("Error loading map, %v", err)
	}
	l.generic, err = data.ReadGeneric(l.gameDirname)
	if err != nil {
		return fmt.Errorf("Error loading generic, %v", err)
	}
	l.hexes, err = data.ReadHexes(l.gameDirname)
	if err != nil {
		return fmt.Errorf("Error loading hexes, %v", err)
	}
	return nil
}

type VariantsLoading struct {
	mainGame    *Game
	scenario    data.Scenario
	loadingDone chan error
	loadingText *Button
}

func NewVariantsLoading(scenario data.Scenario, mainGame *Game, font *data.Font) *VariantsLoading {
	return &VariantsLoading{
		mainGame:    mainGame,
		scenario:    scenario,
		loadingText: NewButton("... LOADING ...", 0, 0, font)}
}
func (l *VariantsLoading) Update() error {
	if l.loadingDone == nil {
		l.loadingDone = make(chan error)
		go func() {
			l.loadingDone <- l.loadVariants()
		}()
	} else {
		select {
		case err := <-l.loadingDone:
			if err != nil {
				return err
			}
			l.mainGame.subGame = NewVariantSelection(l.mainGame)
		default:
		}
	}
	return nil

}
func (l *VariantsLoading) Draw(screen *ebiten.Image) {
	l.loadingText.Draw(screen)
}
func (l *VariantsLoading) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 400, 300
}
func (l *VariantsLoading) loadVariants() (err error) {
	variantsFilename := path.Join(l.mainGame.gameDirname, l.scenario.FilePrefix+".VAR")
	l.mainGame.variants, err = data.ReadVariants(variantsFilename)
	if err != nil {
		return
	}

	generalsFilename := path.Join(l.mainGame.gameDirname, l.scenario.FilePrefix+".GEN")
	l.mainGame.generals, err = data.ReadGenerals(generalsFilename)
	if err != nil {
		return
	}

	terrainFilename := path.Join(l.mainGame.gameDirname, l.scenario.FilePrefix+".TER")
	l.mainGame.terrain, err = data.ReadTerrain(terrainFilename)
	if err != nil {
		return
	}

	scenarioDataFilename := path.Join(l.mainGame.gameDirname, l.scenario.FilePrefix+".DTA")
	l.mainGame.scenarioData, err = data.ReadScenarioData(scenarioDataFilename)
	if err != nil {
		return
	}

	return
}

type VariantLoading struct {
	mainGame    *Game
	loadingDone chan error
}

func NewVariantLoading(mainGame *Game) *VariantLoading {
	return &VariantLoading{
		mainGame: mainGame,
	}
}
func (l *VariantLoading) Update() error {
	if l.loadingDone == nil {
		l.loadingDone = make(chan error)
		go func() {
			l.loadingDone <- l.loadVariant()
		}()
	} else {
		select {
		case err := <-l.loadingDone:
			if err != nil {
				return err
			}
			l.mainGame.subGame = NewShowMap(l.mainGame)
			l.mainGame.subGame.Update()
		default:
		}
	}
	return nil

}
func (l *VariantLoading) Draw(screen *ebiten.Image) {
	//	ebitenutil.DebugPrint(screen, "... LOADING ...")
}
func (l *VariantLoading) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 400, 300
}
func (l *VariantLoading) loadVariant() error {
	unitsFilename := path.Join(l.mainGame.gameDirname, l.mainGame.scenarios[l.mainGame.selectedScenario].FilePrefix+".UNI")
	var err error
	l.mainGame.units, err = data.ReadUnits(unitsFilename, l.mainGame.scenarioData.UnitNames, l.mainGame.generals)
	return err
}
