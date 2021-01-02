package main

import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

import "github.com/pwiecz/command_series/lib"

type OptionSelection struct {
	font           *lib.Font
	balanceStrings []string

	onOptionsSelected func(*lib.Options)
	options           *lib.Options

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

var crusadeSidesStrings = [2]string{"Allied", "German"}
var decisionSidesStrings = [2]string{"British", "Axis"}
var conflictSidesStrings = [2]string{"Free World", "Communist"}
var balanceStrings = [5]string{"++GERMAN", "+GERMAN", "FAIR", "+ALLIED", "++ALLIED"}
var conflictBalanceStrings = [5]string{"++COMMUNIST", "+COMMUNIST", "EVEN", "+FREEWORLD", "++FREEWORLD"}

func NewOptionSelection(game lib.Game, font *lib.Font, onOptionsSelected func(*lib.Options)) *OptionSelection {
	options := lib.DefaultOptions()
	s := &OptionSelection{
		font:              font,
		onOptionsSelected: onOptionsSelected,
		options:           &options}

	var side0Command, side1Command string
	switch game {
	case lib.Crusade:
		side0Command = crusadeSidesStrings[0] + " Command:"
		side1Command = crusadeSidesStrings[1] + " Command:"
	case lib.Decision:
		side0Command = decisionSidesStrings[0] + " Command:"
		side1Command = decisionSidesStrings[1] + " Command:"
	case lib.Conflict:
		side0Command = conflictSidesStrings[0] + " Command:"
		side1Command = conflictSidesStrings[1] + " Command:"
	}
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

	if game != lib.Conflict {
		s.balanceStrings = balanceStrings[:]
	} else {
		s.balanceStrings = conflictBalanceStrings[:]
	}

	s.side0Button = NewButton(s.options.AlliedCommander.String(), buttonPosition, 48, image.Pt(300, 8), font)
	s.side1Button = NewButton(s.options.GermanCommander.String(), buttonPosition, 56, image.Pt(300, 8), font)
	s.intelligenceButton = NewButton(s.options.Intelligence.String(), buttonPosition, 64, image.Pt(300, 8), font)
	s.unitDisplayButton = NewButton(s.options.UnitDisplay.String(), buttonPosition, 72, image.Pt(300, 8), font)
	s.balanceButton = NewButton(s.balanceStrings[s.options.GameBalance], buttonPosition, 80, image.Pt(300, 8), font)
	s.speedButton = NewButton(s.options.Speed.String(), buttonPosition, 88, image.Pt(300, 8), font)

	return s
}

func (s *OptionSelection) changeAlliedCommander() {
	s.options.AlliedCommander = s.options.AlliedCommander.Other()
	s.side0Button.Text = s.options.AlliedCommander.String()
}
func (s *OptionSelection) changeGermanCommander() {
	s.options.GermanCommander = s.options.GermanCommander.Other()
	s.side1Button.Text = s.options.GermanCommander.String()
}
func (s *OptionSelection) changeIntelligence() {
	s.options.Intelligence = s.options.Intelligence.Other()
	s.intelligenceButton.Text = s.options.Intelligence.String()
}
func (s *OptionSelection) changeUnitDisplay() {
	s.options.UnitDisplay = 1 - s.options.UnitDisplay
	s.unitDisplayButton.Text = s.options.UnitDisplay.String()
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
		s.options.Speed++
		if s.options.Speed > 3 {
			s.options.Speed = 1
		}
	} else {
		s.options.Speed--
		if s.options.Speed < 1 {
			s.options.Speed = 3
		}
	}
	s.speedButton.Text = s.options.Speed.String()
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
		s.onOptionsSelected(s.options)
	}
	return nil
}

func (s *OptionSelection) Draw(screen *ebiten.Image) {
	screen.Fill(lib.RGBPalette[15])
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
		cursorImage := *s.font.Glyph(' ')
		cursorImage.Palette = []color.Color{lib.RGBPalette[0x84], lib.RGBPalette[0x84]}
		s.cursorImage = ebiten.NewImageFromImage(&cursorImage)
	}
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(32, float64(s.cursorRow*8+48))
	screen.DrawImage(s.cursorImage, &opts)
}
