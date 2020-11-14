package main

import "image"
import "image/color"
import "github.com/pwiecz/command_series/data"
import "github.com/hajimehoshi/ebiten"

type MapView struct {
	terrainMap             *data.Map
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	dx, dy                 int

	tiles       *[48]*image.Paletted
	unitSymbols *[16]*image.Paletted
	unitIcons   *[16]*image.Paletted
	icons       *[24]*image.Paletted

	isNight  int // 0 or 1
	useIcons bool

	daytimePalette *[8]byte
	nightPalette   *[8]byte

	colorSchemes [2][4][]color.Color

	ebitenTiles   [2][4][48]*ebiten.Image
	ebitenSymbols [2][4][16]*ebiten.Image
	ebitenIcons   [2][4][16]*ebiten.Image
}

func NewMapView(terrainMap *data.Map,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	unitSymbols *[16]*image.Paletted,
	unitIcons *[16]*image.Paletted,
	icons *[24]*image.Paletted,
	daytimePalette *[8]byte,
	nightPalette *[8]byte) *MapView {
	v := &MapView{
		terrainMap:     terrainMap,
		minX:           minX,
		minY:           minY,
		maxX:           maxX,
		maxY:           maxY,
		tiles:          tiles,
		unitSymbols:    unitSymbols,
		unitIcons:      unitIcons,
		icons:          icons,
		daytimePalette: daytimePalette,
		nightPalette:   nightPalette}
	return v
}

func (v *MapView) getTileImage(colorScheme, tileNum int) *ebiten.Image {
	ebitenTile := v.ebitenTiles[v.isNight][colorScheme][tileNum]
	if ebitenTile == nil {
		tile := v.tiles[tileNum]
		tile.Palette = v.getBackgroundForegroundColors(colorScheme)
		ebitenTile = ebiten.NewImageFromImage(tile)
		v.ebitenTiles[v.isNight][colorScheme][tileNum] = ebitenTile
	}
	return ebitenTile
}
func (v *MapView) getUnitSymbolImage(colorScheme, spriteNum int) *ebiten.Image {
	ebitenSprite := v.ebitenSymbols[v.isNight][colorScheme][spriteNum]
	if ebitenSprite == nil {
		sprite := v.unitSymbols[spriteNum]
		sprite.Palette = v.getBackgroundForegroundColors(colorScheme)
		ebitenSprite = ebiten.NewImageFromImage(sprite)
		v.ebitenSymbols[v.isNight][colorScheme][spriteNum] = ebitenSprite
	}
	return ebitenSprite
}
func (v *MapView) getUnitIconImage(colorScheme, spriteNum int) *ebiten.Image {
	ebitenSprite := v.ebitenIcons[v.isNight][colorScheme][spriteNum]
	if ebitenSprite == nil {
		sprite := v.unitIcons[spriteNum]
		sprite.Palette = v.getBackgroundForegroundColors(colorScheme)
		ebitenSprite = ebiten.NewImageFromImage(sprite)
		v.ebitenIcons[v.isNight][colorScheme][spriteNum] = ebitenSprite
	}
	return ebitenSprite
}
func (v *MapView) ToMapCoords(imageX, imageY int) (x, y int) {
	y = imageY/8 + v.minY
	x = ((imageX+(y%2)*4)/8)*2 - y%2 + v.minX*2
	return

}
func (v *MapView) SetUseIcons(useIcons bool) {
	v.useIcons = useIcons
}
func (v *MapView) SetIsNight(isNight bool) {
	if isNight {
		v.isNight = 1
	} else {
		v.isNight = 0
	}
}
func (v *MapView) getBackgroundForegroundColors(colorScheme int) []color.Color {
	colors := v.colorSchemes[v.isNight][colorScheme]
	if colors == nil {
		var palette *[8]byte
		if v.isNight != 0 {
			palette = v.nightPalette
		} else {
			palette = v.daytimePalette
		}
		colors = make([]color.Color, 2)
		// just guessing here
		colors[0] = &data.RGBPalette[palette[2]]
		switch colorScheme {
		case 0:
			colors[1] = &data.RGBPalette[palette[3]] // or 7
		case 1:
			colors[1] = &data.RGBPalette[palette[6]]
		case 2:
			colors[1] = &data.RGBPalette[palette[0]]
		case 3:
			colors[1] = &data.RGBPalette[palette[4]]
		}
		v.colorSchemes[v.isNight][colorScheme] = colors
	}
	return colors
}
func (v *MapView) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	geoM := options.GeoM
	tileBounds := v.tiles[0].Bounds()
	tileSize := tileBounds.Max.Sub(tileBounds.Min)
	for my := v.minY; my <= v.maxY; my++ {
		if my >= v.terrainMap.Height {
			break
		}
		y := my - v.minY
		for mx := v.minX; mx <= v.maxX; mx++ {
			if mx >= v.terrainMap.Width-my%2 {
				break
			}
			x := mx - v.minX
			tileNum := int(v.terrainMap.GetTile(mx, my))
			var tileImage *ebiten.Image
			if tileNum%64 < 48 {
				tileImage = v.getTileImage(tileNum/64, tileNum%64)
			} else if v.useIcons {
				tileImage = v.getUnitIconImage(tileNum/64, tileNum%16)
			} else {
				tileImage = v.getUnitSymbolImage(tileNum/64, tileNum%16)
			}
			options.GeoM = geoM
			options.GeoM.Translate(float64(x*tileSize.X+(my%2)*tileSize.X/2), float64(y*tileSize.Y))
			screen.DrawImage(tileImage, options)
		}
	}
	options.GeoM = geoM
}
