package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type Animation interface {
	Update()
	Done() bool
	Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions)
}

type UnitAnimation struct {
	mapView *MapView
	player  *AudioPlayer
	sprite  *ebiten.Image
	unit    lib.Unit

	x0, y0, x1, y1 int
	frames         int
	elapsed        int
}

func NewUnitAnimation(mapView *MapView, player *AudioPlayer, unit lib.Unit, x0, y0, x1, y1, frames int) Animation {
	if frames <= 0 {
		panic("frames must be positive")
	}

	return &UnitAnimation{
		mapView: mapView,
		player:  player,
		unit:    unit,
		x0:      x0,
		y0:      y0,
		x1:      x1,
		y1:      y1,
		frames:  frames}
}

func (a *UnitAnimation) Update() {
	a.elapsed++
	if a.player != nil {
		if a.elapsed < a.frames {
			a.player.SetFrequency(0, 70)
			freq := byte(54 + 9*a.elapsed/a.frames)
			a.player.SetFrequency(1, freq)
		} else {
			a.player.SetFrequency(0, 0)
			a.player.SetFrequency(1, 0)
		}
	}
}

func (a *UnitAnimation) Done() bool {
	return a.elapsed >= a.frames
}
func (a *UnitAnimation) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	alpha := float64(a.elapsed) / float64(a.frames)
	// Delay creating sprite to be sure that mapView.isNight is up to date.
	// Otherwise e.g. sprite may be using daytime palette at night.
	if a.sprite == nil {
		a.sprite = a.mapView.GetSpriteFromTileNum(byte(a.unit.Type + a.unit.ColorPalette*16))
	}
	a.mapView.DrawSpriteBetween(a.sprite, a.x0, a.y0, a.x1, a.y1, alpha, screen, options)
}

type IconAnimation struct {
	mapView *MapView
	sprite  *ebiten.Image

	x0, y0, x1, y1 int
	frames         int
	elapsed        int
}

func NewIconAnimation(mapView *MapView, icon lib.IconType, x0, y0, x1, y1, frames int) Animation {
	if frames <= 0 {
		panic("frames must be positive")
	}
	return &IconAnimation{
		mapView: mapView,
		sprite:  mapView.GetSpriteFromIcon(icon),
		x0:      x0,
		y0:      y0,
		x1:      x1,
		y1:      y1,
		frames:  frames}
}

func (a *IconAnimation) Update() {
	a.elapsed++
}

func (a *IconAnimation) Done() bool {
	return a.elapsed >= a.frames
}
func (a *IconAnimation) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	alpha := float64(a.elapsed) / float64(a.frames)
	a.mapView.DrawSpriteBetween(a.sprite, a.x0, a.y0, a.x1, a.y1, alpha, screen, options)
}

type IconsAnimation struct {
	mapView *MapView
	sprite  *ebiten.Image
	icons   []lib.IconType

	x, y    int
	elapsed int
}

func NewIconsAnimation(mapView *MapView, icons []lib.IconType, x, y int) Animation {
	if len(icons) == 0 {
		panic("icons cannot be empty")
	}
	return &IconsAnimation{
		mapView: mapView,
		icons:   icons,
		x:       x,
		y:       y}
}
func (a *IconsAnimation) Update() {
	a.elapsed++
	iconIndex := a.elapsed / 3
	if iconIndex < len(a.icons)-1 {
		a.mapView.ShowIcon(a.icons[iconIndex], a.x, a.y, -1, -5)
	} else {
		a.mapView.ShowIcon(a.icons[len(a.icons)-1], a.x, a.y, -1, -5)
	}
}
func (a *IconsAnimation) Done() bool {
	return a.elapsed/3 >= len(a.icons)-1
}
func (a *IconsAnimation) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
}