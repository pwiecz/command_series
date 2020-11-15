package main

import "image"
import "github.com/hajimehoshi/ebiten"
import "github.com/pwiecz/command_series/data"

type Button struct {
	X, Y      float64
	Rect      image.Rectangle
	Text      string
	Font      *data.Font
	image     *ebiten.Image
	mouseDown bool
}

func NewButton(text string, x, y float64, font *data.Font) *Button {
	fontSize := font.Size()
	img := ebiten.NewImage(len(text)*fontSize.X, fontSize.Y)
	var rect image.Rectangle
	rect.Min.X, rect.Min.Y = int(x), int(y)
	rect.Max.X, rect.Max.Y = int(x)+len(text)*fontSize.X, int(y)+fontSize.Y
	opts := &ebiten.DrawImageOptions{}
	for _, r := range text {
		glyphImg := ebiten.NewImageFromImage(font.Glyph(r))
		img.DrawImage(glyphImg, opts)
		opts.GeoM.Translate(float64(fontSize.X), 0)
	}
	return &Button{
		X: x, Y: y,
		Rect:  rect,
		Text:  text,
		Font:  font,
		image: img,
	}
}

func (b *Button) Draw(dst *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(b.X, b.Y)
	dst.DrawImage(b.image, opts)
}

func (b *Button) Update() bool {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if b.Rect.Min.X <= x && x < b.Rect.Max.X && b.Rect.Min.Y <= y && y < b.Rect.Max.Y {
			b.mouseDown = true
		} else {
			b.mouseDown = false
		}
	} else {
		if b.mouseDown {
			b.mouseDown = false
			return true
		}
		b.mouseDown = false
	}
	return false
}
