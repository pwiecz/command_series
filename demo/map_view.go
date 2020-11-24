package main

import "image"
import "image/color"
import "github.com/pwiecz/command_series/data"
import "github.com/hajimehoshi/ebiten"

type MapView struct {
	terrainMap             *data.Map
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates

	cursorX, cursorY int

	tiles       *[48]*image.Paletted
	unitSymbols *[16]*image.Paletted
	unitIcons   *[16]*image.Paletted
	icons       *[24]*image.Paletted

	tileWidth, tileHeight float64

	isNight  int // 0 or 1
	useIcons bool

	daytimePalette *[8]byte
	nightPalette   *[8]byte

	colorSchemes [2][4][]color.Color

	ebitenTiles       [2][4][48]*ebiten.Image
	ebitenUnitSymbols [2][4][16]*ebiten.Image
	ebitenUnitIcons   [2][4][16]*ebiten.Image
	ebitenIcons       [24]*ebiten.Image
}

func NewMapView(terrainMap *data.Map,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	unitSymbols *[16]*image.Paletted,
	unitIcons *[16]*image.Paletted,
	icons *[24]*image.Paletted,
	daytimePalette *[8]byte,
	nightPalette *[8]byte) *MapView {

	tileBounds := tiles[0].Bounds()

	v := &MapView{
		terrainMap:     terrainMap,
		minX:           minX,
		minY:           minY,
		maxX:           maxX,
		maxY:           maxY,
		cursorX:        minX,
		cursorY:        minY,
		tiles:          tiles,
		unitSymbols:    unitSymbols,
		unitIcons:      unitIcons,
		icons:          icons,
		tileWidth:      float64(tileBounds.Dx()),
		tileHeight:     float64(tileBounds.Dy()),
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
	ebitenSprite := v.ebitenUnitSymbols[v.isNight][colorScheme][spriteNum]
	if ebitenSprite == nil {
		sprite := v.unitSymbols[spriteNum]
		sprite.Palette = v.getBackgroundForegroundColors(colorScheme)
		ebitenSprite = ebiten.NewImageFromImage(sprite)
		v.ebitenUnitSymbols[v.isNight][colorScheme][spriteNum] = ebitenSprite
	}
	return ebitenSprite
}
func (v *MapView) getUnitIconImage(colorScheme, spriteNum int) *ebiten.Image {
	ebitenSprite := v.ebitenUnitIcons[v.isNight][colorScheme][spriteNum]
	if ebitenSprite == nil {
		sprite := v.unitIcons[spriteNum]
		sprite.Palette = v.getBackgroundForegroundColors(colorScheme)
		ebitenSprite = ebiten.NewImageFromImage(sprite)
		v.ebitenUnitIcons[v.isNight][colorScheme][spriteNum] = ebitenSprite
	}
	return ebitenSprite
}
func (v *MapView) ToUnitCoords(imageX, imageY int) (x, y int) {
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

func (v *MapView) UnitCoordsToScreenCoords(mapX, mapY int) (x, y float64) {
	x = float64(mapX-v.minX)*v.tileWidth + float64(mapY%2)*v.tileWidth/2
	y = float64(mapY-v.minY) * v.tileHeight
	return
}
func (v *MapView) DrawTileAt(tileNum int, mapX, mapY int, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	x, y := v.UnitCoordsToScreenCoords(mapX, mapY)
	v.drawTileAtScreenCoords(tileNum, x, y, screen, options)
}
func (v *MapView) DrawSpriteBetween(sprite *ebiten.Image, mapX0, mapY0, mapX1, mapY1 int, alpha float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	x0, y0 := v.UnitCoordsToScreenCoords(mapX0, mapY0)
	x1, y1 := v.UnitCoordsToScreenCoords(mapX1, mapY1)
	x, y := x0+(x1-x0)*alpha, y0+(y1-y0)*alpha
	v.drawSpriteAtScreenCoords(sprite, x, y, screen, options)
}

func (v *MapView) GetSpriteFromTileNum(tileNum int) *ebiten.Image {
	if tileNum%64 < 48 {
		return v.getTileImage(tileNum/64, tileNum%64)
	} else if v.useIcons {
		return v.getUnitIconImage(tileNum/64, tileNum%16)
	} else {
		return v.getUnitSymbolImage(tileNum/64, tileNum%16)
	}
}
func (v *MapView) GetSpriteFromIcon(icon data.IconType) *ebiten.Image {
	ebitenTile := v.ebitenIcons[icon]
	if ebitenTile == nil {
		tile := v.icons[icon]
		ebitenTile = ebiten.NewImageFromImage(tile)
		v.ebitenIcons[icon] = ebitenTile
	}
	return ebitenTile
}

func (v *MapView) drawTileAtScreenCoords(tileNum int, x, y float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	tileImage := v.GetSpriteFromTileNum(tileNum)
	v.drawSpriteAtScreenCoords(tileImage, x, y, screen, options)
}
func (v *MapView) drawSpriteAtScreenCoords(sprite *ebiten.Image, x, y float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	geoM := options.GeoM
	options.GeoM.Translate(x, y)
	screen.DrawImage(sprite, options)
	options.GeoM = geoM
}

func (v *MapView) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	for y := v.minY; y <= v.maxY; y++ {
		if y >= v.terrainMap.Height {
			break
		}
		for x := v.minX; x <= v.maxX; x++ {
			if x >= v.terrainMap.Width-y%2 {
				break
			}
			tileNum := int(v.terrainMap.GetTile(x, y))
			v.DrawTileAt(tileNum, x, y, screen, options)
		}
	}
	cursorSprite := v.GetSpriteFromIcon(data.Cursor)
	cursorX, cursorY := v.UnitCoordsToScreenCoords(v.cursorX, v.cursorY)
	geoM := options.GeoM
	options.GeoM.Scale(2, 1)
	v.drawSpriteAtScreenCoords(cursorSprite, cursorX-6, cursorY-2, screen, options)
	options.GeoM = geoM
	return
}
