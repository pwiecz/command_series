package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type labelCell struct {
	rune     rune
	inverted bool
}

type Label struct {
	x, y                       float64
	font                       *lib.Font
	image                      *ebiten.Image
	textColor, backgroundColor int
	dirty                      bool
	cells, targetCells         []labelCell
}

func NewLabel(text string, x, y float64, width, height int, font *lib.Font) *Label {
	numCells := (width + font.Size().X - 1) / font.Size().X
	l := &Label{
		x:               x,
		y:               y,
		font:            font,
		image:           ebiten.NewImage(width, height),
		textColor:       15,
		backgroundColor: 0,
		dirty:           true,
		cells:           make([]labelCell, numCells),
		targetCells:     make([]labelCell, numCells)}
	l.SetText(text, 0, false)
	for x := len(text); x < numCells; x++ {
		l.targetCells[x].rune = ' '

	}
	return l
}

func (l *Label) SetBackgroundColor(color int) {
	if color == l.backgroundColor {
		return
	}
	l.backgroundColor = color
	l.dirty = true
}
func (l *Label) SetTextColor(color int) {
	if color == l.textColor {
		return
	}
	l.textColor = color
	l.dirty = true
}
func (l *Label) Clear() {
	for i := 0; i < len(l.targetCells); i++ {
		l.targetCells[i].rune = ' '
		l.targetCells[i].inverted = false
	}
}
func (l *Label) SetText(text string, x int, inverted bool) {
	for _, r := range text {
		if x >= len(l.targetCells) {
			return
		}
		l.targetCells[x].rune = r
		l.targetCells[x].inverted = inverted
		x++
	}
}
func (l *Label) SetCharInverted(x int, inverted bool) {
	if x >= len(l.targetCells) {
		return
	}
	l.targetCells[x].inverted = inverted
}
func (l *Label) Draw(screen *ebiten.Image) {
	if l.dirty {
		l.image.Fill(lib.RGBPalette[l.backgroundColor])
	}
	fontWidth := float64(l.font.Size().X)
	var opts ebiten.DrawImageOptions
	for x := 0; x < len(l.cells); x++ {
		if l.dirty || l.cells[x] != l.targetCells[x] {
			glyph := l.font.Glyph(l.targetCells[x].rune)
			if !l.targetCells[x].inverted {
				glyph.Palette[0] = lib.RGBPalette[l.backgroundColor]
				glyph.Palette[1] = lib.RGBPalette[l.textColor]
			} else {
				glyph.Palette[0] = lib.RGBPalette[l.textColor]
				glyph.Palette[1] = lib.RGBPalette[l.backgroundColor]
			}
			l.image.DrawImage(ebiten.NewImageFromImage(glyph), &opts)
			l.cells[x] = l.targetCells[x]
		}
		opts.GeoM.Translate(fontWidth, 0)
	}
	l.dirty = false
	opts.GeoM.Reset()
	opts.GeoM.Translate(l.x, l.y)
	screen.DrawImage(l.image, &opts)
}
