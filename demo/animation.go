package main

import "github.com/pwiecz/command_series/data"
import "github.com/hajimehoshi/ebiten"

type Animation struct {
	mapView        *MapView
	unit           data.Unit
	x0, y0, x1, y1 int
	rounds         int
	elapsed        int
}

func NewAnimation(mapView *MapView, unit data.Unit, x0, y0, x1, y1, rounds int, moveToFinal bool) *Animation {
	if rounds <= 0 {
		panic("rounds must be positive")
	}
	return &Animation{mapView, unit, x0, y0, x1, y1, rounds, 0}
}

func (a *Animation) Update() {
	a.elapsed++
}

func (a *Animation) Done() bool {
	return a.elapsed >= a.rounds
}
func (a *Animation) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	alpha := float64(a.elapsed) / float64(a.rounds)
	a.mapView.DrawTileBetween(a.unit.Type+a.unit.ColorPalette*16, a.x0, a.y0, a.x1, a.y1, alpha, screen, options)
}
