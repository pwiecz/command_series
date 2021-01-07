package main

import "github.com/pwiecz/command_series/lib"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

type ListBox struct {
	rows         []*Label
	x, y         float64
	width        int
	height       int
	items        []string
	topItem      int
	selectedItem int
	onEnter      func(string)
}

func NewListBox(x, y float64, width, height int, items []string, font *lib.Font, onEnter func(string)) *ListBox {
	l := &ListBox{
		width:   width,
		height:  height,
		items:   items,
		onEnter: onEnter}
	fontSize := font.Size()
	for i := 0; i < height; i++ {
		l.rows = append(l.rows, NewLabel(items[i], x, y+float64(fontSize.Y*i), width*fontSize.X, fontSize.Y, font))
	}
	if len(items) == 0 || height == 0 {
		return l
	}
	for i := len(items[0]); i < width; i++ {
		l.rows[0].SetCharInverted(i, true)
	}
	return l
}

func (l *ListBox) SetTextColor(textColor int) {
	for _, row := range l.rows {
		row.SetTextColor(textColor)
	}
}
func (l *ListBox) SetBackgroundColor(backgroundColor int) {
	for _, row := range l.rows {
		row.SetBackgroundColor(backgroundColor)
	}
}
func (l *ListBox) Update() {
	modified := false
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		l.onEnter(l.items[l.selectedItem])
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		l.onEnter("")
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		if l.selectedItem+1 < len(l.items) {
			l.selectedItem++
			for l.selectedItem-l.topItem >= l.height {
				l.topItem++
			}
			modified = true
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		if l.selectedItem > 0 {
			l.selectedItem--
			for l.selectedItem < l.topItem {
				l.topItem--
			}
			modified = true
		}
	}
	if modified {
		for i := 0; i < len(l.rows); i++ {
			l.rows[i].Clear()
			itemIx := l.topItem + i
			if itemIx < len(l.items) {
				text := l.items[itemIx]
				if itemIx == l.selectedItem {
					l.rows[i].SetText(text, 0, true)
					for x := len(text); x < l.width; x++ {
						l.rows[i].SetCharInverted(x, true)
					}
				} else {
					l.rows[i].SetText(text, 0, false)
				}
			}
		}

	}
}

func (l *ListBox) Draw(screen *ebiten.Image) {
	for _, row := range l.rows {
		row.Draw(screen)
	}
}
