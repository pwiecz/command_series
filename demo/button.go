package main

import "image"
import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"
import "github.com/pwiecz/command_series/data"

type Button struct {
	Text      string
	x, y      float64
	rect      image.Rectangle
	font      *data.Font
	image     *ebiten.Image
	shownText string
}

func NewButton(text string, x, y float64, size image.Point, font *data.Font) *Button {
	var rect image.Rectangle
	rect.Min.X, rect.Min.Y = int(x), int(y)
	rect.Max.X, rect.Max.Y = int(x)+size.X, int(y)+size.Y
	return &Button{
		x: x, y: y,
		rect: rect,
		Text: text,
		font: font}
}

func (b *Button) Draw(dst *ebiten.Image) {
	if b.image == nil || b.shownText != b.Text {
		if b.image == nil {
			b.image = ebiten.NewImage(b.rect.Dx(), b.rect.Dy())
		}
		b.image.Fill(data.RGBPalette[15])
		fontSize := b.font.Size()
		var opts ebiten.DrawImageOptions
		for _, r := range b.Text {
			glyphImg := ebiten.NewImageFromImage(b.font.Glyph(r))
			b.image.DrawImage(glyphImg, &opts)
			opts.GeoM.Translate(float64(fontSize.X), 0)
		}
		b.shownText = b.Text
	}
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(b.x, b.y)
	dst.DrawImage(b.image, opts)
}

func (b *Button) Update() bool {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if image.Pt(x, y).In(b.rect) {
			return true
		}
	}
	return false
}
