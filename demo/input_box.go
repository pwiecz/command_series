package main

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

import "github.com/pwiecz/command_series/lib"

type InputBox struct {
	x, y            float64
	size            int
	textColor       int
	backgroundColor int
	font            *lib.Font
	image           *ebiten.Image
	shownText       string
	cursorPosition  int
	onEnter         func(string)
}

func NewInputBox(x, y float64, size int, font *lib.Font, onEnter func(string)) *InputBox {
	return &InputBox{
		x:       x,
		y:       y,
		size:    size,
		font:    font,
		onEnter: onEnter}
}

func (i *InputBox) SetTextColor(textColor int) {
	i.textColor = textColor
}
func (i *InputBox) SetBackgroundColor(backgroundColor int) {
	i.backgroundColor = backgroundColor
}
func (i *InputBox) Update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		i.onEnter(i.shownText)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if i.cursorPosition > 0 {
			i.cursorPosition--
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if i.cursorPosition < len(i.shownText) {
			i.cursorPosition++
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if i.cursorPosition > 0 {
			i.shownText = i.shownText[:i.cursorPosition-1] + i.shownText[i.cursorPosition:]
			i.cursorPosition--
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		if i.cursorPosition < len(i.shownText) {
			i.shownText = i.shownText[:i.cursorPosition] + i.shownText[i.cursorPosition+1:]
		}
	} else if len(i.shownText) < i.size {
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
			i.insertStringAtCursor("-")
		} else {
			for k := ebiten.Key(0); k < ebiten.KeyMax; k++ {
				// hacky...
				if inpututil.IsKeyJustPressed(k) && len(k.String()) == 1 {
					i.insertStringAtCursor(k.String())
					break
				}
			}
		}
	}
}

func (i *InputBox) insertStringAtCursor(s string) {
	i.shownText = i.shownText[:i.cursorPosition] + s + i.shownText[i.cursorPosition:]
	i.cursorPosition++

}
func (i *InputBox) Draw(screen *ebiten.Image) {
	fontSize := i.font.Size()
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(i.x, i.y)
	for ix, r := range i.shownText {
		glyph := i.font.Glyph(r)
		if ix != i.cursorPosition {
			glyph.Palette[0] = lib.RGBPalette[i.backgroundColor]
			glyph.Palette[1] = lib.RGBPalette[i.textColor]
		} else {
			glyph.Palette[0] = lib.RGBPalette[i.textColor]
			glyph.Palette[1] = lib.RGBPalette[i.backgroundColor]
		}
		screen.DrawImage(ebiten.NewImageFromImage(glyph), opts)
		opts.GeoM.Translate(float64(fontSize.X), 0)
	}
	if i.cursorPosition >= len(i.shownText) {
		glyph := i.font.Glyph(' ')
		glyph.Palette[0] = lib.RGBPalette[i.textColor]
		glyph.Palette[1] = lib.RGBPalette[i.backgroundColor]
		screen.DrawImage(ebiten.NewImageFromImage(glyph), opts)
	}
}
