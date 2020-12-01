package main

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

type Command int

const (
	Freeze Command = iota
	StatusReport
	UnitInfo
	GeneralInfo
	CityInfo
	HideUnits
	ShowOverviewMap
	DecreaseSpeed
	IncreaseSpeed
	SwitchUnitDisplay
	Quit
	ScrollDown
	ScrollUp
	ScrollLeft
	ScrollRight
	Reserve
	Defend
	Attack
	Move
	SetObjective
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
	} else if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return UnitInfo, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		return GeneralInfo, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		return CityInfo, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		return HideUnits, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		return ShowOverviewMap, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyComma) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return DecreaseSpeed, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		return IncreaseSpeed, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyU) {
		return SwitchUnitDisplay, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return Quit, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		return Reserve, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		return Defend, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		return Attack, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		return Move, true
	} else if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		return SetObjective, true
	} else if inpututil.KeyPressDuration(ebiten.KeyDown)%12 == 1 {
		return ScrollDown, true
	} else if inpututil.KeyPressDuration(ebiten.KeyUp)%12 == 1 {
		return ScrollUp, true
	} else if inpututil.KeyPressDuration(ebiten.KeyRight)%12 == 1 {
		return ScrollRight, true
	} else if inpututil.KeyPressDuration(ebiten.KeyLeft)%12 == 1 {
		return ScrollLeft, true
	}
	return 0, false
}
