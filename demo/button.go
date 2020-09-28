package main

import "image"
import "github.com/hajimehoshi/ebiten"

type Button struct {
	X, Y float64
	Rect image.Rectangle
	Text string
	Font *Font
	image *ebiten.Image
	mouseDown bool
}

func NewButton(text string, x, y float64, font *Font) (*Button, error) {
	img, err := ebiten.NewImage(len(text) * font.Width * 2., font.Height, ebiten.FilterNearest)
	var rect image.Rectangle
	rect.Min.X, rect.Min.Y = int(x), int(y)
	rect.Max.X, rect.Max.Y = int(x) + len(text) * font.Width * 2., int(y)+font.Height
	if err != nil {
		return nil, err
	}
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(2., 1.)
	for _, r := range text {
		img.DrawImage(font.Glyph(r), opts)
		opts.GeoM.Translate(float64(font.Width)*2., 0)
	}
	return &Button{
		X : x, Y: y,
		Rect: rect,
		Text: text,
		Font: font,
		image: img,
	}, nil
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
