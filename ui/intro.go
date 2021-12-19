package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type Intro struct {
	font       *lib.Font
	ebitenFont map[rune]*ebiten.Image
}

func NewIntro(font *lib.Font) *Intro {
	return &Intro{
		font:       font,
		ebitenFont: make(map[rune]*ebiten.Image)}
}

func (i *Intro) getSprite(r rune) *ebiten.Image {
	if e, ok := i.ebitenFont[r]; ok {
		return e
	}
	e := ebiten.NewImageFromImage(i.font.Glyph(r))
	i.ebitenFont[r] = e
	return e
}
func GetDecisionFlagImage() [][]rune {
	return [][]rune{
		[]rune{5, 6, 7, 8, 8, 8, 8, ' ', 9, 9, 9, 9, ' ', 8, 8, 8, 8, 10, 11, 12},
		[]rune{16, 17, 5, 6, 7, 8, 8, ' ', 9, 9, 9, 9, ' ', 8, 8, 10, 11, 12, 18, 19},
		[]rune{8, 20, 16, 17, 5, 6, 7, ' ', 9, 9, 9, 9, ' ', 10, 11, 12, 18, 19, 13, 8},
		[]rune{8, 8, 8, 20, 16, 17, 5, ' ', 9, 9, 9, 9, ' ', 12, 18, 19, 13, 8, 8, 8},
		[]rune{14, 14, 14, 14, 14, 15, 29, ' ', 9, 9, 9, 9, ' ', 30, 31, 14, 14, 14, 14, 14},
		[]rune{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
		[]rune{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
		[]rune{'#', '#', '#', '#', '#', '$', '%', ' ', 9, 9, 9, 9, ' ', '&', 3, '#', '#', '#', '#', '#'},
		[]rune{8, 8, 8, 10, 11, 12, 18, ' ', 9, 9, 9, 9, ' ', 17, 5, 6, 7, 8, 8, 8},
		[]rune{8, 10, 11, 12, 18, 19, 13, ' ', 9, 9, 9, 9, ' ', 20, 16, 17, 5, 6, 7, 8},
		[]rune{11, 12, 18, 19, 13, 8, 8, ' ', 9, 9, 9, 9, ' ', 8, 8, 20, 16, 17, 5, 6},
		[]rune{18, 19, 13, 8, 8, 8, 8, ' ', 9, 9, 9, 9, ' ', 8, 8, 8, 8, 20, 16, 17},
	}
}
func GetCrusadeFlagImage() [][]rune {
	return [][]rune{
		[]rune{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5},
		[]rune{5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5},
		[]rune{5, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 5, 5},
		[]rune{5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5},
		[]rune{5, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 5, 5},
		[]rune{5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5},
		[]rune{5, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 5, 5},
		[]rune{5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5},
		[]rune{5, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 10, 11, 5, 5, 5},
		[]rune{5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5, 8, 9, 5},
		[]rune{6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6},
	}
}
func GetPeopleRows() [][]rune {
	return [][]rune{
		[]rune{' ', ' ', 25, 26, 25, 26, ' ', ' ', 25, 26, 25, 26, 25, 26, ' ', ' ', 25, 26, ' ', ' ', ' ', ' ', 25, 26, 25, 26, ' ', ' ', 25, 26, 25, 26, 25, 26, ' ', ' ', 25, 26},
		[]rune{' ', 25, 22, 24, 23, 24, 23, 21, 22, 21, 22, 24, 23, 21, 22, 21, 22, 21, 26, ' ', ' ', 25, 22, 21, 22, 24, 23, 21, 22, 21, 22, 21, 22, 21, 22, 24, 23, 21, 26},
		[]rune{25, 22, 21, 26, 24, 23, 25, 22, 21, 22, 21, 26, 24, 23, 21, 22, 21, 22, 24, ' ', ' ', 23, 21, 22, 24, 23, 24, 23, 24, 23, 21, 22, 24, 23, 21, 22, 24, 23, 21, 26},
		[]rune{23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, ' ', ' ', ' ', ' ', 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23, 24, 23},
	}
}
func (i *Intro) Draw(screen *ebiten.Image) {
	str := GetCrusadeFlagImage()
	opts := ebiten.DrawImageOptions{}
	width := float64(i.font.Size().X)
	height := float64(i.font.Size().Y)
	for _, line := range str {
		for _, r := range line {
			if r < ' ' {
				r += 1000
			}
			s := i.getSprite(r)
			screen.DrawImage(s, &opts)
			opts.GeoM.Translate(width, 0)
		}
		opts.GeoM.Translate(-width*float64(len(line)), height)
	}
	opts.GeoM.Translate(0, height*6)
	for _, line := range GetPeopleRows() {
		for _, r := range line {
			if r < ' ' {
				r += 1000
			}
			s := i.getSprite(r)
			screen.DrawImage(s, &opts)
			opts.GeoM.Translate(width, 0)
		}
		opts.GeoM.Translate(-width*float64(len(line)), height)
	}

}
