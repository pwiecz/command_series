package main

import "image"
import "github.com/hajimehoshi/ebiten"
import "github.com/pwiecz/command_series/data"

type Rectangle struct {
	image                     *ebiten.Image
	currentColor, targetColor int
}

func NewRectangle(size image.Point, color int) *Rectangle {
	r := &Rectangle{
		image:        ebiten.NewImage(size.X, size.Y),
		targetColor:  color,
		currentColor: color}
	r.image.Fill(data.RGBPalette[color])
	return r
}

func (r *Rectangle) SetColor(color int) {
	r.targetColor = color
}
func (r *Rectangle) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if r.currentColor != r.targetColor {
		r.image.Fill(data.RGBPalette[r.targetColor])
		r.currentColor = r.targetColor
	}
	screen.DrawImage(r.image, opts)
}
