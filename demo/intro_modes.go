package main

import (
	"fmt"
	"image/color"
	"io/fs"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/pwiecz/command_series/lib"
)

type ScenarioSelection struct {
	labels             []*Label
	buttons            []*Button
	onScenarioSelected func(int)
}

func NewScenarioSelection(scenarios []lib.Scenario, font *lib.Font, onScenarioSelected func(int)) *ScenarioSelection {
	labels := []*Label{
		NewLabel("SCENARIO SELECTION", 16, 32, 300, 8, font),
		NewLabel(fmt.Sprintf("TYPE (1-%d)", len(scenarios)), 16, float64(56+len(scenarios)*8), 300, 8, font)}
	for _, label := range labels {
		label.SetTextColor(0)
		label.SetBackgroundColor(15)
	}
	buttons := make([]*Button, len(scenarios))
	x, y := 16.0, 48.0
	fontSize := font.Size()
	for i, scenario := range scenarios {
		button := NewButton(fmt.Sprintf("%d. %s", i+1, scenario.Name), x, y, 300, 8, font)
		buttons[i] = button
		y += float64(fontSize.Y)
	}
	return &ScenarioSelection{
		labels:             labels,
		buttons:            buttons,
		onScenarioSelected: onScenarioSelected}
}

func (s *ScenarioSelection) Update() error {
	for i, button := range s.buttons {
		if button.Update() || (i < 10 && inpututil.IsKeyJustReleased(numToKey(i+1))) {
			s.onScenarioSelected(i)
			return nil
		}
	}
	return nil
}
func (s *ScenarioSelection) Draw(screen *ebiten.Image) {
	screen.Fill(lib.RGBPalette[15])
	for _, label := range s.labels {
		label.Draw(screen)
	}
	for _, button := range s.buttons {
		button.Draw(screen)
	}
}

type VariantSelection struct {
	labels            []*Label
	buttons           []*Button
	onVariantSelected func(int)
}

func NewVariantSelection(variants []lib.Variant, font *lib.Font, onVariantSelected func(int)) *VariantSelection {
	labels := []*Label{
		NewLabel("VARIANT SELECTION", 16, 32, 300, 8, font),
		NewLabel(fmt.Sprintf("TYPE (1-%d)", len(variants)), 16, float64(56+len(variants)*8), 300, 8, font)}
	for _, label := range labels {
		label.SetTextColor(0)
		label.SetBackgroundColor(15)
	}
	buttons := make([]*Button, len(variants))
	x, y := 16.0, 48.0
	fontSize := font.Size()
	for i, variant := range variants {
		button := NewButton(fmt.Sprintf("%d. %s", i+1, variant.Name), x, y, 300, 8, font)
		buttons[i] = button
		y += float64(fontSize.Y)
	}
	return &VariantSelection{
		labels:            labels,
		buttons:           buttons,
		onVariantSelected: onVariantSelected}
}

func (s *VariantSelection) Update() error {
	for i, button := range s.buttons {
		if button.Update() || (i < 10 && inpututil.IsKeyJustReleased(numToKey(i+1))) {
			s.onVariantSelected(i)
			return nil
		}
	}
	return nil
}
func (s *VariantSelection) Draw(screen *ebiten.Image) {
	screen.Fill(lib.RGBPalette[15])
	for _, label := range s.labels {
		label.Draw(screen)
	}
	for _, button := range s.buttons {
		button.Draw(screen)
	}
}

type GameLoading struct {
	fsys         fs.FS
	onGameLoaded func(*lib.GameData)
	loadingDone  chan error
	gameData     *lib.GameData

	turnsLoading int
	loadingRect  *ebiten.Image
}

func NewGameLoading(fsys fs.FS, onGameLoaded func(*lib.GameData)) *GameLoading {
	return &GameLoading{
		fsys:         fsys,
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
			l.onGameLoaded(l.gameData)
		default:
		}
	}
	l.turnsLoading++
	return nil
}
func (l *GameLoading) Draw(screen *ebiten.Image) {
	screen.Fill(lib.RGBPalette[15])
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

func (l *GameLoading) loadGameData() error {
	gameData, err := lib.LoadGameData(l.fsys)
	if err != nil {
		return err
	}
	l.gameData = gameData
	return nil
}

type ScenarioLoading struct {
	fsys             fs.FS
	filePrefix       string
	onScenarioLoaded func(*lib.ScenarioData)
	loadingDone      chan error
	scenarioData     *lib.ScenarioData
	loadingText      *Label
}

func NewScenarioLoading(fsys fs.FS, scenario lib.Scenario, font *lib.Font, onScenarioLoaded func(*lib.ScenarioData)) *ScenarioLoading {
	l := &ScenarioLoading{
		fsys:             fsys,
		filePrefix:       scenario.FilePrefix,
		loadingText:      NewLabel("... LOADING ...", 0, 0, 120, 8, font),
		onScenarioLoaded: onScenarioLoaded}
	l.loadingText.SetBackgroundColor(15)
	return l
}
func (l *ScenarioLoading) Update() error {
	if l.loadingDone == nil {
		l.loadingDone = make(chan error)
		go func() {
			l.loadingDone <- l.loadScenarioData()
		}()
	} else {
		select {
		case err := <-l.loadingDone:
			if err != nil {
				return err
			}
			l.onScenarioLoaded(l.scenarioData)
		default:
		}
	}
	return nil

}
func (l *ScenarioLoading) Draw(screen *ebiten.Image) {
	screen.Fill(lib.RGBPalette[15])
	l.loadingText.Draw(screen)
}
func (l *ScenarioLoading) loadScenarioData() (err error) {
	scenarioData, err := lib.LoadScenarioData(l.fsys, l.filePrefix)
	if err != nil {
		return err
	}
	l.scenarioData = scenarioData
	return nil
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
