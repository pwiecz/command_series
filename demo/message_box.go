package main

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/pwiecz/command_series/lib"
)

type MessageBox struct {
	rows []*Label
}

func NewMessageBox(x, y float64, width, height int, font *lib.Font) *MessageBox {
	b := &MessageBox{}
	fontSize := font.Size()
	numRows := (height + fontSize.Y - 1) / fontSize.Y
	for i := 0; i < numRows; i++ {
		b.rows = append(b.rows, NewLabel("", x, y+float64(fontSize.Y*i), width, fontSize.Y, font))
	}
	return b
}

func (b *MessageBox) SetRowBackground(y, color int) {
	if y >= len(b.rows) {
		return
	}
	b.rows[y].SetBackgroundColor(color)
}
func (b *MessageBox) SetTextColor(color int) {
	for _, row := range b.rows {
		row.SetTextColor(color)
	}
}
func (b *MessageBox) Clear() {
	for y := 0; y < len(b.rows); y++ {
		b.ClearRow(y)
	}
}
func (b *MessageBox) ClearRow(y int) {
	if y >= len(b.rows) {
		return
	}
	b.rows[y].Clear()
}
func (b *MessageBox) Print(str string, x, y int, inverted bool) {
	if y >= len(b.rows) {
		return
	}
	b.rows[y].SetText(str, x, inverted)
}

func (b *MessageBox) Draw(screen *ebiten.Image) {
	for _, row := range b.rows {
		row.Draw(screen)
	}
}
