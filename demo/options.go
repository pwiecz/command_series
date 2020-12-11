package main

import "image"
import "strings"

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
}

var commanderStrings = [2]string{"PLAYER", "COMPUTER"}
var intelligenceStrings = [2]string{"FULL", "LIMITED"}
var unitDisplayStrings = [2]string{"SYMBOLS", "ICONS"}
var speedStrings = [3]string{"SLOW", "MEDIUM", "FAST"}

func NewOptionSelection(mainGame *Game) *OptionSelection {
	s := &OptionSelection{mainGame: mainGame}
	s.options.GermanCommander = 1
	s.options.Intelligence = Limited
	s.options.GameBalance = 2
	s.options.Speed = 1
	side0 := mainGame.scenarioData.Sides[0]
	side1 := mainGame.scenarioData.Sides[1]
	capitalizedSide0 := side0[0:1] + strings.ToLower(side0[1:])
	capitalizedSide1 := side1[0:1] + strings.ToLower(side1[1:])

	font := mainGame.sprites.IntroFont
	s.labels = append(s.labels,
		NewButton("OPTION SELECTION", 24, 32, image.Pt(300, 8), font),
		NewButton(capitalizedSide0+" Command:", 40, 48, image.Pt(300, 8), font),
		NewButton(capitalizedSide1+" Command:", 40, 56, image.Pt(300, 8), font),
		NewButton("Intelligence:", 40, 64, image.Pt(300, 8), font),
		NewButton("Unit Display:", 40, 72, image.Pt(300, 8), font),
		NewButton("Play Balance:", 40, 80, image.Pt(300, 8), font),
		NewButton("Speed:", 40, 88, image.Pt(300, 8), font))

	maxLabelLength := 0
	for _, label := range s.labels[1:] {
		if len(label.Text) > maxLabelLength {
			maxLabelLength = len(label.Text)
		}
	}
	buttonPosition := float64(40 + (maxLabelLength+1)*8)

	s.balanceStrings = append(s.balanceStrings, "++"+side1, "+"+side1, "FAIR", "+"+side0, "++"+side0)
	s.side0Button = NewButton(commanderStrings[s.options.AlliedCommander], buttonPosition, 48, image.Pt(300, 8), font)
	s.side1Button = NewButton(commanderStrings[s.options.GermanCommander], buttonPosition, 56, image.Pt(300, 8), font)
	s.intelligenceButton = NewButton(intelligenceStrings[s.options.Intelligence], buttonPosition, 64, image.Pt(300, 8), font)
	s.unitDisplayButton = NewButton(unitDisplayStrings[s.options.UnitDisplay], buttonPosition, 72, image.Pt(300, 8), font)
	s.balanceButton = NewButton(s.balanceStrings[s.options.GameBalance], buttonPosition, 80, image.Pt(300, 8), font)
	s.speedButton = NewButton(speedStrings[s.options.Speed], buttonPosition, 88, image.Pt(300, 8), font)

	return s
}

func (s *OptionSelection) Update() error {
	if s.side0Button.Update() {
		s.options.AlliedCommander = 1 - s.options.AlliedCommander
		s.side0Button.Text = commanderStrings[s.options.AlliedCommander]
	}
	if s.side1Button.Update() {
		s.options.GermanCommander = 1 - s.options.GermanCommander
		s.side1Button.Text = commanderStrings[s.options.GermanCommander]
	}
	if s.intelligenceButton.Update() {
		s.options.Intelligence = 1 - s.options.Intelligence
		s.intelligenceButton.Text = intelligenceStrings[s.options.Intelligence]
	}
	if s.unitDisplayButton.Update() {
		s.options.UnitDisplay = 1 - s.options.UnitDisplay
		s.unitDisplayButton.Text = unitDisplayStrings[s.options.UnitDisplay]
	}
	if s.balanceButton.Update() {
		s.options.GameBalance = (s.options.GameBalance + 1) % 5
		s.balanceButton.Text = s.balanceStrings[s.options.GameBalance]
	}
	if s.speedButton.Update() {
		s.options.Speed = (s.options.Speed + 1) % 3
		s.speedButton.Text = speedStrings[s.options.Speed]
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
}

func (s *OptionSelection) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 336, 240
}
