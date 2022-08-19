package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type MapDrawer struct {
	terrainMap             *lib.Map
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	images                 [2]*ebiten.Image
	tileWidth, tileHeight  int
	colors                 *lib.ColorSchemes
	tiles                  *[48]*image.Paletted
	tileImages             [2][4][48]*ebiten.Image
}

func NewMapDrawer(
	terrainMap *lib.Map,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	colors *lib.ColorSchemes) *MapDrawer {
	tileBounds := tiles[0].Bounds()
	return &MapDrawer{
		terrainMap: terrainMap,
		minX:       minX,
		minY:       minY,
		maxX:       maxX,
		maxY:       maxY,
		tiles:      tiles,
		tileWidth:  tileBounds.Dx(),
		tileHeight: tileBounds.Dy(),
		colors:     colors}
}

func (d *MapDrawer) GetMapImage(isNight bool) *ebiten.Image {
	isNightIx := 0
	if isNight {
		isNightIx = 1
	}
	if d.images[isNightIx] != nil {
		return d.images[isNightIx]
	}
	d.images[isNightIx] = d.drawMapImage(isNight)
	return d.images[isNightIx]
}
func (d *MapDrawer) drawMapImage(isNight bool) *ebiten.Image {
	image := ebiten.NewImage((d.maxX-d.minX+1)*d.tileWidth+d.tileWidth/2, (d.maxY-d.minY+1)*d.tileHeight)
	image.Fill(d.colors.GetMapBackgroundColor(isNight))
	for y := d.minY; y <= d.maxY; y++ {
		if y >= d.terrainMap.Height {
			break
		}
		for x := d.minX; x <= d.maxX; x++ {
			if x >= d.terrainMap.Width-y%2 {
				break
			}
			tileNum := d.terrainMap.GetTile(lib.MapCoords{X: x, Y: y})
			d.drawTileAt(tileNum, isNight, lib.MapCoords{X: x, Y: y}, image)
		}
	}
	return image
}

func (d *MapDrawer) getTileImage(colorScheme, tileNum byte, isNight bool) *ebiten.Image {
	isNightIx := 0
	if isNight {
		isNightIx = 1
	}
	tileImage := d.tileImages[isNightIx][colorScheme][tileNum]
	if tileImage == nil {
		tile := d.tiles[tileNum]
		tile.Palette = d.colors.GetBackgroundForegroundColors(colorScheme, isNight)
		tileImage = ebiten.NewImageFromImage(tile)
		d.tileImages[isNightIx][colorScheme][tileNum] = tileImage
	}
	return tileImage
}
func (d *MapDrawer) drawTileAt(tileNum byte, isNight bool, mapXY lib.MapCoords, screen *ebiten.Image) {
	x, y := d.MapCoordsToImageCoords(mapXY)
	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(d.GetSpriteFromTileNum(tileNum, isNight), &opts)
}
func (d *MapDrawer) MapCoordsToImageCoords(mapXY lib.MapCoords) (x, y int) {
	x = (mapXY.X-d.minX)*d.tileWidth + (mapXY.Y%2)*d.tileWidth/2
	y = (mapXY.Y - d.minY) * d.tileHeight
	return
}
func (d *MapDrawer) GetSpriteFromTileNum(tileNum byte, isNight bool) *ebiten.Image {
	if tileNum&63 < 48 {
		return d.getTileImage(tileNum/64, tileNum%64, isNight)
	}
	panic(tileNum)
}
