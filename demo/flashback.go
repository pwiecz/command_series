package main

import "errors"
import "fmt"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

import "github.com/pwiecz/command_series/data"

type Flashback struct {
	mapView    *MapView
	messageBox *MessageBox
	terrainMap *data.Map
	flashback  [][]data.FlashbackUnit
	day        int
	shownDay   int
}

func NewFlashback(mapView *MapView, messageBox *MessageBox, terrainMap *data.Map, flashback [][]data.FlashbackUnit) *Flashback {
	messageBox.Clear()
	messageBox.Print("FLASHBACK: DAY 1", 2, 0, false)
	messageBox.Print(" F2 ", 2, 1, true)
	messageBox.Print("NEXT DAY", 7, 1, false)
	messageBox.Print(" F3 ", 2, 2, true)
	messageBox.Print("PREVIOUS DAY", 7, 2, false)
	messageBox.Print(" F4 ", 2, 3, true)
	messageBox.Print("RETURN TO GAME", 7, 3, false)
	return &Flashback{
		mapView:    mapView,
		messageBox: messageBox,
		terrainMap: terrainMap,
		flashback:  flashback,
		shownDay:   -1}
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
	} else if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		f.hideUnitsFromDay(f.shownDay)
		return errors.New("Exit")
	}
	return nil
}

func (f *Flashback) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if f.day != f.shownDay {
		f.hideUnitsFromDay(f.shownDay)
		if f.day < len(f.flashback) {
			for _, unit := range f.flashback[f.day] {
				f.terrainMap.SetTile(unit.X/2, unit.Y, byte(unit.Type+unit.ColorPalette*16))
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
		f.terrainMap.SetTile(unit.X/2, unit.Y, byte(unit.Terrain))
	}
}
