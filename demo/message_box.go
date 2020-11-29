package main

import "image"
import "github.com/hajimehoshi/ebiten"
import "github.com/pwiecz/command_series/data"

type coordsContent struct {
	textColor, backgroundColor int
	rune                       rune
}
type MessageBox struct {
	messageImage        *ebiten.Image
	size                image.Point
	font                *data.Font
	numRows, numColumns int
	rowBackgrounds []int
	currentContent, targetContent [][]coordsContent
}

func NewMessageBox(size image.Point, font *data.Font) *MessageBox {
	b := &MessageBox{
		messageImage: ebiten.NewImage(size.X, size.Y),
		size:         size,
		font:         font}
	fontSize := font.Size()
	b.numRows = (size.Y + fontSize.Y - 1) / fontSize.Y
	b.numColumns = (size.X + fontSize.X - 1) / fontSize.X
	b.rowBackgrounds = make([]int, b.numRows)
	b.currentContent = make([][]coordsContent, b.numRows)
	b.targetContent = make([][]coordsContent, b.numRows)
	for y := 0; y < b.numRows; y++ {
		b.currentContent[y] = make([]coordsContent, b.numColumns)
		b.targetContent[y] = make([]coordsContent, b.numColumns)
		for x := 0; x < b.numColumns; x++ {
			b.targetContent[y][x].rune = ' '
		}
	}
	return b
}

func (b *MessageBox) SetRowBackground(y, color int) {
	if y >= b.numRows {
		return
	}
	row := b.targetContent[y]
	for x := 0; x < b.numColumns; x++ {
		if row[x].backgroundColor == b.rowBackgrounds[y] {
			row[x].backgroundColor = color
		}
		if row[x].textColor == b.rowBackgrounds[y] {
			row[x].textColor = color
		}
	}
	b.rowBackgrounds[y] = color
}
func (b *MessageBox) Clear() {
	for y := 0; y < b.numRows; y++ {
		b.ClearRow(y)
	}
}
func (b *MessageBox) ClearRow(y int) {
	if y >= b.numRows {
		return
	}
	backgroundColor := b.rowBackgrounds[y]
	row := b.targetContent[y]
	for x := 0; x < b.numColumns; x++ {
		row[x].backgroundColor = backgroundColor
		row[x].rune = ' '
	}
}
func (b *MessageBox) Print(str string, x, y int, inverted bool) {
	if y >= b.numRows {
		return
	}
	row := b.targetContent[y]
	background := b.rowBackgrounds[y]
	for _, r := range str {
		if x >= b.numColumns {
			return
		}
		if !inverted {
			row[x].textColor = 0
			row[x].backgroundColor = background
		} else {
			row[x].backgroundColor = 0
			row[x].textColor = background
		}
		row[x].rune = r
		x++
	}
}

func (b *MessageBox) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if b.messageImage == nil {
		b.messageImage = ebiten.NewImage(b.size.X, b.size.Y)
	}
	fontSize := b.font.Size()
	y0 := 0
	for y := 0; y < b.numRows; y++ {
		x0 := 0
		for x := 0; x < b.numColumns; x++ {
			if b.currentContent[y][x] != b.targetContent[y][x] {
				glyph := b.font.Glyph(b.targetContent[y][x].rune)
				glyph.Palette[0] = data.RGBPalette[b.targetContent[y][x].backgroundColor]
				glyph.Palette[1] = data.RGBPalette[b.targetContent[y][x].textColor]
				glyphImg := ebiten.NewImageFromImage(glyph)
				var opts ebiten.DrawImageOptions
				opts.GeoM.Translate(float64(x0), float64(y0))
				b.messageImage.DrawImage(glyphImg, &opts)
				b.currentContent[y][x] = b.targetContent[y][x]
			}
			x0 += fontSize.X
		}
		y0 += fontSize.Y
	}
	screen.DrawImage(b.messageImage, opts)
}
