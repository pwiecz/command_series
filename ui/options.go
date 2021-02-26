package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/pwiecz/command_series/lib"
)

type OptionSelection struct {
	font           *lib.Font
	balanceStrings []string

	onOptionsSelected func(*lib.Options)
	options           *lib.Options

	labels             []*Label
	side0Button        *Button
	side1Button        *Button
	intelligenceButton *Button
	unitDisplayButton  *Button
	balanceButton      *Button
	speedButton        *Button

	cursorImage *ebiten.Image
	cursorRow   int

	enterBounds image.Rectangle
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

	s.labels = []*Label{NewLabel("OPTION SELECTION", 24, 32, 300, 8, font)}
	labelTexts := []string{side0Command, side1Command, "Intelligence:", "Unit Display:", "Play Balance:", "Speed:"}
	maxLabelLength := 0
	fontHeight := float64(font.Size().Y)
	y := 48.0
	for _, text := range labelTexts {
		s.labels = append(s.labels, NewLabel(text, 40, y, 300, 8, font))
		y += fontHeight
		if len(text) > maxLabelLength {
			maxLabelLength = len(text)
		}
	}
	s.labels = append(s.labels,
		NewLabel("Select Options ...", 40, 128, 300, 8, font),
		NewLabel("Then press ENTER.", 40, 136, 300, 8, font))
	s.enterBounds = image.Rect(40+11*8, 136, 40+16*8, 136+16)
	for _, label := range s.labels {
		label.SetTextColor(0)
		label.SetBackgroundColor(15)
	}

	buttonX := float64(40 + (maxLabelLength+1)*8)

	if game != lib.Conflict {
		s.balanceStrings = balanceStrings[:]
	} else {
		s.balanceStrings = conflictBalanceStrings[:]
	}

	s.side0Button = NewButton(s.options.AlliedCommander.String(), buttonX, 48, 300, 8, font)
	s.side1Button = NewButton(s.options.GermanCommander.String(), buttonX, 56, 300, 8, font)
	s.intelligenceButton = NewButton(s.options.Intelligence.String(), buttonX, 64, 300, 8, font)
	s.unitDisplayButton = NewButton(s.options.UnitDisplay.String(), buttonX, 72, 300, 8, font)
	s.balanceButton = NewButton(s.balanceStrings[s.options.GameBalance], buttonX, 80, 300, 8, font)
	s.speedButton = NewButton(s.options.Speed.String(), buttonX, 88, 300, 8, font)

	return s
}

func (s *OptionSelection) changeAlliedCommander() {
	s.options.AlliedCommander = s.options.AlliedCommander.Other()
	s.side0Button.SetText(s.options.AlliedCommander.String())
}
func (s *OptionSelection) changeGermanCommander() {
	s.options.GermanCommander = s.options.GermanCommander.Other()
	s.side1Button.SetText(s.options.GermanCommander.String())
}
func (s *OptionSelection) changeIntelligence() {
	s.options.Intelligence = s.options.Intelligence.Other()
	s.intelligenceButton.SetText(s.options.Intelligence.String())
}
func (s *OptionSelection) changeUnitDisplay() {
	s.options.UnitDisplay = 1 - s.options.UnitDisplay
	s.unitDisplayButton.SetText(s.options.UnitDisplay.String())
}
func (s *OptionSelection) changeGameBalance(forward bool) {
	if forward {
		s.options.GameBalance = (s.options.GameBalance + 1) % 5
	} else {
		s.options.GameBalance = (s.options.GameBalance + 4) % 5
	}
	s.balanceButton.SetText(s.balanceStrings[s.options.GameBalance])
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
	s.speedButton.SetText(s.options.Speed.String())
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
	for _, touchID := range inpututil.JustPressedTouchIDs() {
		x, y := ebiten.TouchPosition(touchID)
		if  image.Pt(x, y).In(s.enterBounds) {
			s.onOptionsSelected(s.options)
		}
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
