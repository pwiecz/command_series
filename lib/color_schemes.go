package lib

import (
	"image/color"
)

type ColorSchemes struct {
	daytimePalette      *[8]byte
	nightPalette        *[8]byte
	daytimeColorSchemes [4][]color.Color
	nightColorSchemes   [4][]color.Color
}

func NewColorSchemes(daytimePalette *[8]byte, nightPalette *[8]byte) *ColorSchemes {
	return &ColorSchemes{
		daytimePalette: daytimePalette,
		nightPalette:   nightPalette}
}
func (c *ColorSchemes) GetBackgroundForegroundColors(colorScheme byte, isNight bool) []color.Color {
	if isNight {
		return c.getColors(colorScheme, c.nightPalette, &c.nightColorSchemes)
	} else {
		return c.getColors(colorScheme, c.daytimePalette, &c.daytimeColorSchemes)
	}
}

func (c *ColorSchemes) GetMapBackgroundColor(isNight bool) color.Color {
	if isNight {
		return RGBPalette[c.nightPalette[2]]
	} else {
		return RGBPalette[c.daytimePalette[2]]
	}
}

func (c *ColorSchemes) getColors(colorScheme byte, palette *[8]byte, colorSchemes *[4][]color.Color) []color.Color {
	colors := colorSchemes[colorScheme]
	if colors == nil {
		colors = make([]color.Color, 2)
		// just guessing here
		colors[0] = &RGBPalette[palette[2]]
		switch colorScheme {
		case 0:
			colors[1] = &RGBPalette[palette[3]] // or 7
		case 1:
			colors[1] = &RGBPalette[palette[6]]
		case 2:
			colors[1] = &RGBPalette[palette[0]]
		case 3:
			colors[1] = &RGBPalette[palette[4]]
		}
		colorSchemes[colorScheme] = colors
	}
	return colors
}
