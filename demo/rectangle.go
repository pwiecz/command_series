package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type Rectangle struct {
	x, y                      float64
	image                     *ebiten.Image
	currentColor, targetColor int
}

func NewRectangle(x, y float64, width, height int) *Rectangle {
	r := &Rectangle{
		x:     x,
		y:     y,
		image: ebiten.NewImage(width, height)}
	r.image.Fill(lib.RGBPalette[0])
	return r
}

func (r *Rectangle) SetColor(color int) {
	r.targetColor = color
}
func (r *Rectangle) Draw(screen *ebiten.Image) {
	if r.currentColor != r.targetColor {
		r.image.Fill(lib.RGBPalette[r.targetColor])
		r.currentColor = r.targetColor
	}
	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(r.x, r.y)
	screen.DrawImage(r.image, &opts)
}
