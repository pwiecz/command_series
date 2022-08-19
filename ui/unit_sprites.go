package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type unitSprites struct {
	isNight     int // 0 or 1
	unitDisplay lib.UnitDisplay

	colors *lib.ColorSchemes

	unitSymbols      *[16]*image.Paletted
	unitIcons        *[16]*image.Paletted
	unitSymbolImages [2][4][16]*ebiten.Image
	unitIconImages   [2][4][16]*ebiten.Image
}

func newUnitSprites(unitSymbols *[16]*image.Paletted, unitIcons *[16]*image.Paletted, colors *lib.ColorSchemes) *unitSprites {
	return &unitSprites{
		unitSymbols: unitSymbols,
		unitIcons:   unitIcons,
		colors:      colors}
}

func (s *unitSprites) getUnitSymbolImage(colorScheme, spriteNum byte) *ebiten.Image {
	symbolImage := s.unitSymbolImages[s.isNight][colorScheme][spriteNum]
	if symbolImage == nil {
		sprite := s.unitSymbols[spriteNum]
		sprite.Palette = s.colors.GetBackgroundForegroundColors(colorScheme, s.isNight != 0)
		symbolImage = ebiten.NewImageFromImage(sprite)
		s.unitSymbolImages[s.isNight][colorScheme][spriteNum] = symbolImage
	}
	return symbolImage
}
func (s *unitSprites) getUnitIconImage(colorScheme, spriteNum byte) *ebiten.Image {
	iconImage := s.unitIconImages[s.isNight][colorScheme][spriteNum]
	if iconImage == nil {
		sprite := s.unitIcons[spriteNum]
		sprite.Palette = s.colors.GetBackgroundForegroundColors(colorScheme, s.isNight != 0)
		iconImage = ebiten.NewImageFromImage(sprite)
		s.unitIconImages[s.isNight][colorScheme][spriteNum] = iconImage
	}
	return iconImage
}

func (s *unitSprites) SetUnitDisplay(unitDisplay lib.UnitDisplay) {
	s.unitDisplay = unitDisplay
}
func (s *unitSprites) SetIsNight(isNight bool) {
	if isNight {
		s.isNight = 1
	} else {
		s.isNight = 0
	}
}
func (s *unitSprites) GetSpriteForUnit(unit lib.Unit) *ebiten.Image {
	tileNum := byte(unit.Type + unit.ColorPalette*16)
	if s.unitDisplay == lib.ShowAsIcons {
		return s.getUnitIconImage(tileNum/64, tileNum%16)
	} else {
		return s.getUnitSymbolImage(tileNum/64, tileNum%16)
	}
}
