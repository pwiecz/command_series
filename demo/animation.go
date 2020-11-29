package main

import "github.com/pwiecz/command_series/data"
import "github.com/hajimehoshi/ebiten"

type Animation struct {
	mapView *MapView
	sprite  *ebiten.Image
	hasUnit bool
	unit    data.Unit
	icon    data.IconType

	x0, y0, x1, y1 int
	frames         int
	elapsed        int
}

func NewUnitAnimation(mapView *MapView, unit data.Unit, x0, y0, x1, y1, frames int) *Animation {
	if frames <= 0 {
		panic("frames must be positive")
	}

	return &Animation{
		mapView: mapView,
		hasUnit: true,
		unit:    unit,
		x0:      x0,
		y0:      y0,
		x1:      x1,
		y1:      y1,
		frames:  frames}
}

func NewIconAnimation(mapView *MapView, icon data.IconType, x0, y0, x1, y1, frames int) *Animation {
	if frames <= 0 {
		panic("frames must be positive")
	}
	return &Animation{
		mapView: mapView,
		hasUnit: false,
		icon:    icon,
		x0:      x0,
		y0:      y0,
		x1:      x1,
		y1:      y1,
		frames:  frames}
}

func (a *Animation) Update() {
	a.elapsed++
}

func (a *Animation) Done() bool {
	return a.elapsed >= a.frames
}
func (a *Animation) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	alpha := float64(a.elapsed) / float64(a.frames)
	// Delay creating sprite to be sure that mapView.isNight is up to date.
	// Otherwise e.g. sprite may be using daytime palette at night.
	if a.sprite == nil {
		if a.hasUnit {
			a.sprite = a.mapView.GetSpriteFromTileNum(byte(a.unit.Type + a.unit.ColorPalette*16))
		} else {
			a.sprite = a.mapView.GetSpriteFromIcon(a.icon)
		}
	}
	a.mapView.DrawSpriteBetween(a.sprite, a.x0, a.y0, a.x1, a.y1, alpha, screen, options)
}
