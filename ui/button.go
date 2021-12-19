package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/pwiecz/command_series/lib"
)

type Button struct {
	label           *Label
	rect            image.Rectangle
	pressedTouchIDs []ebiten.TouchID // store it here to avoid reallocating it for each Update
}

func NewButton(text string, x, y float64, width, height int, font *lib.Font) *Button {
	b := &Button{
		label: NewLabel(text, x, y, width, height, font),
		rect:  image.Rect(int(x), int(y), int(x)+width, int(y)+height)}
	b.label.SetTextColor(0)
	b.label.SetBackgroundColor(15)
	return b
}

func (b *Button) SetText(text string) {
	b.label.Clear()
	b.label.SetText(text, 0)
}
func (b *Button) Draw(dst *ebiten.Image) {
	b.label.Draw(dst)
}

func (b *Button) Update() bool {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if image.Pt(x, y).In(b.rect) {
			return true
		}
	}
	b.pressedTouchIDs = b.pressedTouchIDs[:0]
	for _, touchID := range inpututil.AppendJustPressedTouchIDs(b.pressedTouchIDs) {
		x, y := ebiten.TouchPosition(touchID)
		if image.Pt(x, y).In(b.rect) {
			return true
		}
	}
	return false
}
