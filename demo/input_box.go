package main

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

import "github.com/pwiecz/command_series/lib"

type InputBox struct {
	label          *Label
	width          int
	text           string
	cursorPosition int
	onEnter        func(string)
}

func NewInputBox(x, y float64, width int, font *lib.Font, onEnter func(string)) *InputBox {
	fontSize := font.Size()
	return &InputBox{
		label:   NewLabel(x, y, (width+1)*fontSize.X, fontSize.Y, font),
		width:   width,
		onEnter: onEnter}
}

func (i *InputBox) SetText(text string) {
	i.text = text
	i.cursorPosition = len(i.text)
}
func (i *InputBox) SetTextColor(textColor int) {
	i.label.SetTextColor(textColor)
}
func (i *InputBox) SetBackgroundColor(backgroundColor int) {
	i.label.SetBackgroundColor(backgroundColor)
}

func (i *InputBox) Update() {
	modified := false
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		i.onEnter(i.text)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		i.onEnter("")
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if i.cursorPosition > 0 {
			i.cursorPosition--
			modified = true
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if i.cursorPosition < len(i.text) {
			i.cursorPosition++
			modified = true
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		i.cursorPosition = 0
		modified = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		i.cursorPosition = len(i.text)
		modified = true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if i.cursorPosition > 0 {
			i.text = i.text[:i.cursorPosition-1] + i.text[i.cursorPosition:]
			i.cursorPosition--
			modified = true
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDelete) {
		if i.cursorPosition < len(i.text) {
			i.text = i.text[:i.cursorPosition] + i.text[i.cursorPosition+1:]
			modified = true
		}
	} else if len(i.text) < i.width {
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
			i.insertStringAtCursor("-")
			modified = true
		} else {
			for k := ebiten.Key(0); k < ebiten.KeyMax; k++ {
				// hacky...
				if inpututil.IsKeyJustPressed(k) && len(k.String()) == 1 {
					i.insertStringAtCursor(k.String())
					modified = true
					break
				}
			}
		}
	}
	if modified {
		i.label.Clear()
		i.label.SetText(i.text, 0, false)
		i.label.SetCharInverted(i.cursorPosition, true)
	}
}

func (i *InputBox) insertStringAtCursor(s string) {
	i.text = i.text[:i.cursorPosition] + s + i.text[i.cursorPosition:]
	i.cursorPosition++

}
func (i *InputBox) Draw(screen *ebiten.Image) {
	i.label.Draw(screen)
}
