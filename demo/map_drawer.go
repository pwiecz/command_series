package main

import "image"
import "image/color"
import "github.com/hajimehoshi/ebiten"

import "github.com/pwiecz/command_series/lib"

type MapDrawer struct {
	terrainMap             *lib.Map
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	image                  *ebiten.Image
	isDirty                bool
	drawnTerrainTiles      [][]byte
	subImage               *ebiten.Image

	tileWidth, tileHeight int

	isNight     int // 0 or 1
	unitDisplay lib.UnitDisplay

	daytimePalette *[8]byte
	nightPalette   *[8]byte

	colorSchemes [2][4][]color.Color

	tiles            *[48]*image.Paletted
	unitSymbols      *[16]*image.Paletted
	unitIcons        *[16]*image.Paletted
	tileImages       [2][4][48]*ebiten.Image
	unitSymbolImages [2][4][16]*ebiten.Image
	unitIconImages   [2][4][16]*ebiten.Image
}

func NewMapDrawer(
	terrainMap *lib.Map,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	unitSymbols *[16]*image.Paletted,
	unitIcons *[16]*image.Paletted,
	daytimePalette *[8]byte,
	nightPalette *[8]byte) *MapDrawer {
	tileBounds := tiles[0].Bounds()
	tileWidth := tileBounds.Dx()
	tileHeight := tileBounds.Dy()
	return &MapDrawer{
		terrainMap:     terrainMap,
		minX:           minX,
		minY:           minY,
		maxX:           maxX,
		maxY:           maxY,
		image:          ebiten.NewImage((maxX-minX+1)*tileWidth+tileWidth/2, (maxY-minY+1)*tileHeight),
		isDirty:        true,
		tiles:          tiles,
		unitSymbols:    unitSymbols,
		unitIcons:      unitIcons,
		tileWidth:      tileWidth,
		tileHeight:     tileHeight,
		daytimePalette: daytimePalette,
		nightPalette:   nightPalette}
}

func (d *MapDrawer) getTileImage(colorScheme, tileNum byte) *ebiten.Image {
	tileImage := d.tileImages[d.isNight][colorScheme][tileNum]
	if tileImage == nil {
		tile := d.tiles[tileNum]
		tile.Palette = d.getBackgroundForegroundColors(colorScheme)
		tileImage = ebiten.NewImageFromImage(tile)
		d.tileImages[d.isNight][colorScheme][tileNum] = tileImage
	}
	return tileImage
}
func (d *MapDrawer) getUnitSymbolImage(colorScheme, spriteNum byte) *ebiten.Image {
	symbolImage := d.unitSymbolImages[d.isNight][colorScheme][spriteNum]
	if symbolImage == nil {
		sprite := d.unitSymbols[spriteNum]
		sprite.Palette = d.getBackgroundForegroundColors(colorScheme)
		symbolImage = ebiten.NewImageFromImage(sprite)
		d.unitSymbolImages[d.isNight][colorScheme][spriteNum] = symbolImage
	}
	return symbolImage
}
func (d *MapDrawer) getUnitIconImage(colorScheme, spriteNum byte) *ebiten.Image {
	iconImage := d.unitIconImages[d.isNight][colorScheme][spriteNum]
	if iconImage == nil {
		sprite := d.unitIcons[spriteNum]
		sprite.Palette = d.getBackgroundForegroundColors(colorScheme)
		iconImage = ebiten.NewImageFromImage(sprite)
		d.unitIconImages[d.isNight][colorScheme][spriteNum] = iconImage
	}
	return iconImage
}

func (d *MapDrawer) SetUnitDisplay(unitDisplay lib.UnitDisplay) {
	if d.unitDisplay != unitDisplay {
		d.unitDisplay = unitDisplay
		d.isDirty = true
	}
}
func (d *MapDrawer) SetIsNight(isNight bool) {
	if isNight != (d.isNight == 1) {
		if isNight {
			d.isNight = 1
		} else {
			d.isNight = 0
		}
		d.isDirty = true
	}
}
func (d *MapDrawer) getBackgroundForegroundColors(colorScheme byte) []color.Color {
	colors := d.colorSchemes[d.isNight][colorScheme]
	if colors == nil {
		var palette *[8]byte
		if d.isNight != 0 {
			palette = d.nightPalette
		} else {
			palette = d.daytimePalette
		}
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
		d.colorSchemes[d.isNight][colorScheme] = colors
	}
	return colors
}
func (d *MapDrawer) getDrawnTile(x, y int) byte {
	dx, dy := x-d.minX, y-d.minY
	for dy >= len(d.drawnTerrainTiles) {
		d.drawnTerrainTiles = append(d.drawnTerrainTiles, make([]byte, d.maxX-d.minX+1))
	}
	for dx >= len(d.drawnTerrainTiles[dy]) {
		d.drawnTerrainTiles[dy] = append(d.drawnTerrainTiles[dy], 0)
	}
	return d.drawnTerrainTiles[dy][dx]
}
func (d *MapDrawer) setDrawnTile(x, y int, tile byte) {
	dx, dy := x-d.minX, y-d.minY
	for dy >= len(d.drawnTerrainTiles) {
		d.drawnTerrainTiles = append(d.drawnTerrainTiles, make([]byte, d.maxX-d.minX+1))
	}
	for dx >= len(d.drawnTerrainTiles[dy]) {
		d.drawnTerrainTiles[dy] = append(d.drawnTerrainTiles[dy], 0)
	}
	d.drawnTerrainTiles[dy][dx] = tile
}
func (d *MapDrawer) DrawTileAt(tileNum byte, mapX, mapY int, screen *ebiten.Image) {
	x, y := d.MapCoordsToImageCoords(mapX, mapY)
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(d.GetSpriteFromTileNum(tileNum), &opts)
}
func (d *MapDrawer) MapCoordsToImageCoords(mapX, mapY int) (x, y int) {
	x = (mapX-d.minX)*d.tileWidth + (mapY%2)*d.tileWidth/2
	y = (mapY - d.minY) * d.tileHeight
	return
}
func (d *MapDrawer) GetSpriteFromTileNum(tileNum byte) *ebiten.Image {
	if tileNum%64 < 48 {
		return d.getTileImage(tileNum/64, tileNum%64)
	} else if d.unitDisplay == lib.ShowAsIcons {
		return d.getUnitIconImage(tileNum/64, tileNum%16)
	} else {
		return d.getUnitSymbolImage(tileNum/64, tileNum%16)
	}
}
func (d *MapDrawer) Draw() {
	for y := d.minY; y <= d.maxY; y++ {
		if y >= d.terrainMap.Height {
			break
		}
		for x := d.minX; x <= d.maxX; x++ {
			if x >= d.terrainMap.Width-y%2 {
				break
			}
			tileNum := d.terrainMap.GetTile(x, y)
			if d.isDirty || tileNum != d.getDrawnTile(x, y) {
				d.DrawTileAt(tileNum, x, y, d.image)
				d.setDrawnTile(x, y, tileNum)
			}
		}
	}
	d.isDirty = false
}
func (d *MapDrawer) GetSubImage(bounds image.Rectangle) *ebiten.Image {

	return d.image.SubImage(bounds).(*ebiten.Image)
}
