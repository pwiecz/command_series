package main

import "fmt"
import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

import "github.com/pwiecz/command_series/data"

type IntelligenceType int

const (
	Full    IntelligenceType = 0
	Limited IntelligenceType = 1
)

type Options struct {
	AlliedCommander int
	GermanCommander int
	Intelligence    IntelligenceType
	UnitDisplay     int
	GameBalance     int // [0..4]
	Speed           int
}

func (o Options) IsPlayerControlled(side int) bool {
	if side == 0 {
		return o.AlliedCommander == 0
	}
	return o.GermanCommander == 0
}
func (o Options) Num() int {
	n := o.AlliedCommander + 2*o.GermanCommander
	if o.Intelligence == Limited {
		n += 56 - 4*(o.AlliedCommander*o.GermanCommander+o.AlliedCommander)
	}
	return n
}

type OptionSelection struct {
	mainGame       *Game
	balanceStrings []string

	options Options

	labels             []*Button
	side0Button        *Button
	side1Button        *Button
	intelligenceButton *Button
	unitDisplayButton  *Button
	balanceButton      *Button
	speedButton        *Button

	cursorImage *ebiten.Image
	cursorRow   int
}

var commanderStrings = [2]string{"PLAYER", "COMPUTER"}
var intelligenceStrings = [2]string{"FULL", "LIMITED"}
var unitDisplayStrings = [2]string{"SYMBOLS", "ICONS"}
var speedStrings = [3]string{"SLOW", "MEDIUM", "FAST"}
var crusadeSidesStrings = [2]string{"Allied", "German"}
var decisionSidesStrings = [2]string{"British", "Axis"}
var conflictSidesStrings = [2]string{"Free World", "Communist"}
var balanceStrings = [5]string{"++GERMAN", "+GERMAN", "FAIR", "+ALLIED", "++ALLIED"}
var conflictBalanceStrings = [5]string{"++COMMUNIST", "+COMMUNIST", "EVEN", "+FREEWORLD", "++FREEWORLD"}

func NewOptionSelection(mainGame *Game) *OptionSelection {
	s := &OptionSelection{mainGame: mainGame}
	s.options.GermanCommander = 1
	s.options.Intelligence = Limited
	s.options.GameBalance = 2
	s.options.Speed = 1

	var side0Command, side1Command string
	switch mainGame.game {
	case data.Crusade:
		side0Command = crusadeSidesStrings[0] + " Command:"
		side1Command = crusadeSidesStrings[1] + " Command:"
	case data.Decision:
		side0Command = decisionSidesStrings[0] + " Command:"
		side1Command = decisionSidesStrings[1] + " Command:"
	case data.Conflict:
		side0Command = conflictSidesStrings[0] + " Command:"
		side1Command = conflictSidesStrings[1] + " Command:"
	}
	font := mainGame.sprites.IntroFont
	s.labels = []*Button{
		NewButton("OPTION SELECTION", 24, 32, image.Pt(300, 8), font),
		NewButton(side0Command, 40, 48, image.Pt(300, 8), font),
		NewButton(side1Command, 40, 56, image.Pt(300, 8), font),
		NewButton("Intelligence:", 40, 64, image.Pt(300, 8), font),
		NewButton("Unit Display:", 40, 72, image.Pt(300, 8), font),
		NewButton("Play Balance:", 40, 80, image.Pt(300, 8), font),
		NewButton("Speed:", 40, 88, image.Pt(300, 8), font),
		NewButton("Select Options ...", 40, 128, image.Pt(300, 8), font),
		NewButton("Then press ENTER.", 40, 136, image.Pt(300, 8), font)}

	maxLabelLength := 0
	for _, label := range s.labels[1:] {
		if len(label.Text) > maxLabelLength {
			maxLabelLength = len(label.Text)
		}
	}
	buttonPosition := float64(40 + (maxLabelLength+1)*8)

	if mainGame.game != data.Conflict {
		s.balanceStrings = balanceStrings[:]
	} else {
		s.balanceStrings = conflictBalanceStrings[:]
	}

	s.side0Button = NewButton(commanderStrings[s.options.AlliedCommander], buttonPosition, 48, image.Pt(300, 8), font)
	s.side1Button = NewButton(commanderStrings[s.options.GermanCommander], buttonPosition, 56, image.Pt(300, 8), font)
	s.intelligenceButton = NewButton(intelligenceStrings[s.options.Intelligence], buttonPosition, 64, image.Pt(300, 8), font)
	s.unitDisplayButton = NewButton(unitDisplayStrings[s.options.UnitDisplay], buttonPosition, 72, image.Pt(300, 8), font)
	s.balanceButton = NewButton(s.balanceStrings[s.options.GameBalance], buttonPosition, 80, image.Pt(300, 8), font)
	s.speedButton = NewButton(speedStrings[s.options.Speed], buttonPosition, 88, image.Pt(300, 8), font)

	return s
}

func (s *OptionSelection) changeAlliedCommander() {
	s.options.AlliedCommander = 1 - s.options.AlliedCommander
	s.side0Button.Text = commanderStrings[s.options.AlliedCommander]
}
func (s *OptionSelection) changeGermanCommander() {
	s.options.GermanCommander = 1 - s.options.GermanCommander
	s.side1Button.Text = commanderStrings[s.options.GermanCommander]
}
func (s *OptionSelection) changeIntelligence() {
	s.options.Intelligence = 1 - s.options.Intelligence
	s.intelligenceButton.Text = intelligenceStrings[s.options.Intelligence]
}
func (s *OptionSelection) changeUnitDisplay() {
	s.options.UnitDisplay = 1 - s.options.UnitDisplay
	s.unitDisplayButton.Text = unitDisplayStrings[s.options.UnitDisplay]
}
func (s *OptionSelection) changeGameBalance(forward bool) {
	if forward {
		s.options.GameBalance = (s.options.GameBalance + 1) % 5
	} else {
		s.options.GameBalance = (s.options.GameBalance + 4) % 5
	}
	s.balanceButton.Text = s.balanceStrings[s.options.GameBalance]
}
func (s *OptionSelection) changeGameSpeed(forward bool) {
	if forward {
		s.options.Speed = (s.options.Speed + 1) % 3
	} else {
		s.options.Speed = (s.options.Speed + 2) % 3
	}
	s.speedButton.Text = speedStrings[s.options.Speed]
}
func (s *OptionSelection) Update() error {
	if s.side0Button.Update() {
		s.changeAlliedCommander()
	}
	if s.side1Button.Update() {
		s.changeGermanCommander()
	}
	if s.intelligenceButton.Update() {
		s.changeIntelligence()
	}
	if s.unitDisplayButton.Update() {
		s.changeUnitDisplay()
	}
	if s.balanceButton.Update() {
		s.changeGameBalance(true)
	}
	if s.speedButton.Update() {
		s.changeGameSpeed(true)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) && s.cursorRow < 5 {
		s.cursorRow++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && s.cursorRow > 0 {
		s.cursorRow--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		switch s.cursorRow {
		case 0:
			s.changeAlliedCommander()
		case 1:
			s.changeGermanCommander()
		case 2:
			s.changeIntelligence()
		case 3:
			s.changeUnitDisplay()
		case 4:
			s.changeGameBalance(false)
		case 5:
			s.changeGameSpeed(false)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		switch s.cursorRow {
		case 0:
			s.changeAlliedCommander()
		case 1:
			s.changeGermanCommander()
		case 2:
			s.changeIntelligence()
		case 3:
			s.changeUnitDisplay()
		case 4:
			s.changeGameBalance(true)
		case 5:
			s.changeGameSpeed(true)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		s.mainGame.subGame = NewShowMap(s.mainGame, s.options)
	}
	return nil
}

func (s *OptionSelection) Draw(screen *ebiten.Image) {
	screen.Fill(data.RGBPalette[15])
	for _, label := range s.labels {
		label.Draw(screen)
	}
	s.side0Button.Draw(screen)
	s.side1Button.Draw(screen)
	s.intelligenceButton.Draw(screen)
	s.unitDisplayButton.Draw(screen)
	s.balanceButton.Draw(screen)
	s.speedButton.Draw(screen)

	if s.cursorImage == nil {
		cursorImage := *s.mainGame.sprites.IntroFont.Glyph(' ')
		cursorImage.Palette = []color.Color{data.RGBPalette[0x84], data.RGBPalette[0x84]}
		s.cursorImage = ebiten.NewImageFromImage(&cursorImage)
	}
	var opts ebiten.DrawImageOptions
	if false {
		fmt.Println(24, float64(s.cursorRow*8+56))
	}
	opts.GeoM.Translate(32, float64(s.cursorRow*8+48))
	screen.DrawImage(s.cursorImage, &opts)
}

func (s *OptionSelection) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 336, 240
}
