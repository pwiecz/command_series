package ui

import (
	"image/color"

	"github.com/pwiecz/command_series/lib"
)

type colorSchemes struct {
	daytimePalette      *[8]byte
	nightPalette        *[8]byte
	daytimeColorSchemes [4][]color.Color
	nightColorSchemes   [4][]color.Color
}

func newColorSchemes(daytimePalette *[8]byte, nightPalette *[8]byte) *colorSchemes {
	return &colorSchemes{
		daytimePalette: daytimePalette,
		nightPalette: nightPalette}
}
func (c *colorSchemes) GetBackgroundForegroundColors(colorScheme byte, isNight bool) []color.Color {
	if isNight {
		return c.getColors(colorScheme, c.nightPalette, &c.nightColorSchemes)
	} else {
		return c.getColors(colorScheme, c.daytimePalette, &c.daytimeColorSchemes)
	}
}

func (c *colorSchemes) getColors(colorScheme byte, palette *[8]byte, colorSchemes *[4][]color.Color) []color.Color {
	colors := colorSchemes[colorScheme]
	if colors == nil {
		colors = make([]color.Color, 2)
		// just guessing here
		colors[0] = &lib.RGBPalette[palette[2]]
		switch colorScheme {
		case 0:
			colors[1] = &lib.RGBPalette[palette[3]] // or 7
		case 1:
			colors[1] = &lib.RGBPalette[palette[6]]
		case 2:
			colors[1] = &lib.RGBPalette[palette[0]]
		case 3:
			colors[1] = &lib.RGBPalette[palette[4]]
		}
		colorSchemes[colorScheme] = colors
	}
	return colors
}
