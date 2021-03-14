package ui

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/pwiecz/command_series/lib"
)

type Flashback struct {
	mapView        *MapView
	messageBox     *MessageBox
	flashback      lib.FlashbackHistory
	terrainTypeMap *lib.TerrainTypeMap
	day            int
	shownDay       int
}

func NewFlashback(mapView *MapView, messageBox *MessageBox, flashback lib.FlashbackHistory, terrainTypeMap *lib.TerrainTypeMap) *Flashback {
	messageBox.Clear()
	messageBox.Print("FLASHBACK: DAY 1", 2, 0, false)
	messageBox.Print(" F2 ", 2, 1, true)
	messageBox.Print("NEXT DAY", 7, 1, false)
	messageBox.Print(" F3 ", 2, 2, true)
	messageBox.Print("PREVIOUS DAY", 7, 2, false)
	messageBox.Print(" F4 ", 2, 3, true)
	messageBox.Print("RETURN TO GAME", 7, 3, false)
	return &Flashback{
		mapView:        mapView,
		messageBox:     messageBox,
		flashback:      flashback,
		terrainTypeMap: terrainTypeMap,
		shownDay:       -1}
}

func (f *Flashback) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		if f.day+1 < len(f.flashback) {
			f.day++
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		if f.day > 0 {
			f.day--
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyF4) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		f.hideUnitsFromDay(f.shownDay)
		return errors.New("Exit")
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		curXY := f.mapView.GetCursorPosition()
		f.mapView.SetCursorPosition(lib.MapCoords{curXY.X, curXY.Y + 1})
	} else if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		curXY := f.mapView.GetCursorPosition()
		f.mapView.SetCursorPosition(lib.MapCoords{curXY.X, curXY.Y - 1})
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		curXY := f.mapView.GetCursorPosition()
		f.mapView.SetCursorPosition(lib.MapCoords{curXY.X + 1, curXY.Y})
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		curXY := f.mapView.GetCursorPosition()
		f.mapView.SetCursorPosition(lib.MapCoords{curXY.X - 1, curXY.Y})
	}
	return nil
}

func (f *Flashback) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if f.day != f.shownDay {
		f.hideUnitsFromDay(f.shownDay)
		if f.day < len(f.flashback) {
			for _, unit := range f.flashback[f.day] {
				f.terrainTypeMap.ShowUnitAt(unit.XY)
			}
		}
		f.messageBox.ClearRow(0)
		f.messageBox.Print(fmt.Sprintf("FLASHBACK: DAY %d", f.day+1), 2, 0, false)
		f.shownDay = f.day
	}
	f.mapView.Draw(screen, opts)
}

func (f *Flashback) hideUnitsFromDay(day int) {
	if day < 0 || day >= len(f.flashback) {
		return
	}
	for _, unit := range f.flashback[day] {
		f.terrainTypeMap.HideUnitAt(unit.XY)
	}
}
