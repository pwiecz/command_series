package main

import "fmt"
import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"
import "github.com/pwiecz/command_series/atr"
import "github.com/pwiecz/command_series/data"

type ScenarioSelection struct {
	labels           []*Button
	buttons          []*Button
	scenarioSelected func(int)
	intro            *Intro
}

func NewScenarioSelection(scenarios []data.Scenario, font *data.Font, scenarioSelected func(int)) *ScenarioSelection {
	labels := []*Button{
		NewButton("SCENARIO SELECTION", 16, 32, image.Pt(300, 8), font),
		NewButton(fmt.Sprintf("TYPE (1-%d)", len(scenarios)), 16, float64(56+len(scenarios)*8), image.Pt(300, 8), font)}
	buttons := make([]*Button, len(scenarios))
	x, y := 16.0, 48.0
	fontSize := font.Size()
	for i, scenario := range scenarios {
		button := NewButton(fmt.Sprintf("%d. %s", i+1, scenario.Name), x, y, image.Pt(300, 8), font)
		buttons[i] = button
		y += float64(fontSize.Y)
	}
	return &ScenarioSelection{
		labels:           labels,
		buttons:          buttons,
		scenarioSelected: scenarioSelected,
		intro:            NewIntro(font)}
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
	screen.Fill(data.RGBPalette[15])
	for _, label := range s.labels {
		label.Draw(screen)
	}
	for _, button := range s.buttons {
		button.Draw(screen)
	}
}
func (s *ScenarioSelection) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 336, 240
}

type VariantSelection struct {
	labels   []*Button
	buttons  []*Button
	mainGame *Game
}

func NewVariantSelection(mainGame *Game) *VariantSelection {
	font := mainGame.sprites.IntroFont
	labels := []*Button{
		NewButton("VARIANT SELECTION", 16, 32, image.Pt(300, 8), font),
		NewButton(fmt.Sprintf("TYPE (1-%d)", len(mainGame.variants)), 16, float64(56+len(mainGame.variants)*8), image.Pt(300, 8), font)}
	buttons := make([]*Button, len(mainGame.variants))
	x, y := 16.0, 48.0
	fontSize := font.Size()
	for i, variant := range mainGame.variants {
		button := NewButton(fmt.Sprintf("%d. %s", i+1, variant.Name), x, y, image.Pt(300, 8), font)
		buttons[i] = button
		y += float64(fontSize.Y)
	}
	return &VariantSelection{
		labels:   labels,
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
	screen.Fill(data.RGBPalette[15])
	for _, label := range s.labels {
		label.Draw(screen)
	}
	for _, button := range s.buttons {
		button.Draw(screen)
	}
}
func (s *VariantSelection) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 336, 240
}

type GameLoading struct {
	diskimage    atr.SectorReader
	onGameLoaded func(data.Game, []data.Scenario, data.Sprites, data.Icons, data.Map, data.Generic, data.Hexes)
	loadingDone  chan error
	game         data.Game
	scenarios    []data.Scenario
	sprites      data.Sprites
	icons        data.Icons
	terrainMap   data.Map
	generic      data.Generic
	hexes        data.Hexes

	turnsLoading int
	loadingRect  *ebiten.Image
}

func NewGameLoading(diskimage atr.SectorReader, onGameLoaded func(data.Game, []data.Scenario, data.Sprites, data.Icons, data.Map, data.Generic, data.Hexes)) *GameLoading {
	return &GameLoading{
		diskimage:    diskimage,
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
			l.onGameLoaded(l.game, l.scenarios, l.sprites, l.icons, l.terrainMap, l.generic, l.hexes)
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
	return 336, 240
}

func (l *GameLoading) loadGameData() error {
	var err error
	l.game, err = data.DetectGame(l.diskimage)
	if err != nil {
		return fmt.Errorf("Error detecting game, %v", err)
	}
	l.scenarios, err = data.ReadScenarios(l.diskimage)
	if err != nil {
		return fmt.Errorf("Error loading scenarios, %v", err)
	}
	l.sprites, err = data.ReadSprites(l.diskimage)
	if err != nil {
		return fmt.Errorf("Error loading sprites, %v", err)
	}
	l.icons, err = data.ReadIcons(l.diskimage)
	if err != nil {
		return fmt.Errorf("Error loading icons, %v", err)
	}
	l.terrainMap, err = data.ReadMap(l.diskimage, l.game)
	if err != nil {
		return fmt.Errorf("Error loading map, %v", err)
	}
	l.generic, err = data.ReadGeneric(l.diskimage)
	if err != nil {
		return fmt.Errorf("Error loading generic, %v", err)
	}
	l.hexes, err = data.ReadHexes(l.diskimage)
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

func NewVariantsLoading(scenario data.Scenario, mainGame *Game) *VariantsLoading {
	return &VariantsLoading{
		mainGame:    mainGame,
		scenario:    scenario,
		loadingText: NewButton("... LOADING ...", 0, 0, image.Pt(120, 8), mainGame.sprites.IntroFont)}
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
	return 336, 240
}
func (l *VariantsLoading) loadVariants() (err error) {
	variantsFilename := l.scenario.FilePrefix + ".VAR"
	l.mainGame.variants, err = data.ReadVariants(l.mainGame.diskimage, variantsFilename)
	if err != nil {
		return
	}

	generalsFilename := l.scenario.FilePrefix + ".GEN"
	l.mainGame.generals, err = data.ReadGenerals(l.mainGame.diskimage, generalsFilename)
	if err != nil {
		return
	}

	terrainFilename := l.scenario.FilePrefix + ".TER"
	l.mainGame.terrain, err = data.ReadTerrain(l.mainGame.diskimage, terrainFilename, l.mainGame.game)
	if err != nil {
		return
	}

	scenarioDataFilename := l.scenario.FilePrefix + ".DTA"
	l.mainGame.scenarioData, err = data.ReadScenarioData(l.mainGame.diskimage, scenarioDataFilename)
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
			l.mainGame.subGame = NewOptionSelection(l.mainGame)
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
	return 336, 240
}
func (l *VariantLoading) loadVariant() error {
	unitsFilename := l.mainGame.scenarios[l.mainGame.selectedScenario].FilePrefix + ".UNI"
	var err error
	l.mainGame.units, err = data.ReadUnits(l.mainGame.diskimage, unitsFilename, l.mainGame.game, l.mainGame.scenarioData.UnitNames, l.mainGame.generals)
	return err
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
