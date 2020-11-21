package main

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

type Command int

const (
	Freeze Command = iota
	StatusReport
	UnitInfo
	DecreaseSpeed
	IncreaseSpeed
	SwitchUnitDisplay
	Quit
	ScrollDown
	ScrollUp
	ScrollLeft
	ScrollRight
)

type CommandBuffer struct {
	Commands chan Command
}

func NewCommandBuffer(size uint) *CommandBuffer {
	return &CommandBuffer{
		Commands: make(chan Command, 20)}
}
func (b *CommandBuffer) Update() {
	if command, ok := b.triggeredCommand(); ok {
		select {
		case b.Commands <- command:
		default:
		}
	}
}

func (b *CommandBuffer) triggeredCommand() (Command, bool) {
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		return Freeze, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return StatusReport, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyComma) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return DecreaseSpeed, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return IncreaseSpeed, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyU) {
		return SwitchUnitDisplay, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return Quit, true
	} else if inpututil.KeyPressDuration(ebiten.KeyDown)%12 == 1 || inpututil.KeyPressDuration(ebiten.KeyJ)%12 == 1 {
		return ScrollDown, true
	} else if inpututil.KeyPressDuration(ebiten.KeyUp)%12 == 1 || inpututil.KeyPressDuration(ebiten.KeyK)%12 == 1 {
		return ScrollUp, true
	} else if inpututil.KeyPressDuration(ebiten.KeyRight)%12 == 1 || inpututil.KeyPressDuration(ebiten.KeyL)%12 == 1 {
		return ScrollRight, true
	} else if inpututil.KeyPressDuration(ebiten.KeyLeft)%12 == 1 || inpututil.KeyPressDuration(ebiten.KeyH)%12 == 1 {
		return ScrollLeft, true
	}
	return 0, false
}
