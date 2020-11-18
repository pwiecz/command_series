package main

import "github.com/pwiecz/command_series/data"
import "github.com/hajimehoshi/ebiten"

type Animation struct {
	mapView        *MapView
	sprite         *ebiten.Image
	x0, y0, x1, y1 int
	rounds         int
	elapsed        int
}

func NewUnitAnimation(mapView *MapView, unit data.Unit, x0, y0, x1, y1, rounds int) *Animation {
	if rounds <= 0 {
		panic("rounds must be positive")
	}
	sprite := mapView.GetSpriteFromTileNum(unit.Type+unit.ColorPalette*16)
	return &Animation{mapView, sprite, x0, y0, x1, y1, rounds, 0}
}

func NewIconAnimation(mapView *MapView, icon data.IconType, x0, y0, x1, y1, rounds int) *Animation {
	if rounds <= 0 {
		panic("rounds must be positive")
	}
	sprite := mapView.GetSpriteFromIcon(icon)
	return &Animation{mapView, sprite, x0, y0, x1, y1, rounds, 0}
}

func (a *Animation) Update() {
	a.elapsed++
}

func (a *Animation) Done() bool {
	return a.elapsed >= a.rounds
}
func (a *Animation) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	alpha := float64(a.elapsed) / float64(a.rounds)
	a.mapView.DrawSpriteBetween(a.sprite, a.x0, a.y0, a.x1, a.y1, alpha, screen, options)
}
