package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type MapDrawer struct {
	terrainMap             *lib.Map
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	image                  *ebiten.Image
	isDirty                bool
	subImage               *ebiten.Image

	tileWidth, tileHeight int

	isNight    int // 0 or 1
	colors     *colorSchemes
	tiles      *[48]*image.Paletted
	tileImages [2][4][48]*ebiten.Image
}

func NewMapDrawer(
	terrainMap *lib.Map,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	colors *colorSchemes) *MapDrawer {
	tileBounds := tiles[0].Bounds()
	tileWidth := tileBounds.Dx()
	tileHeight := tileBounds.Dy()
	return &MapDrawer{
		terrainMap: terrainMap,
		minX:       minX,
		minY:       minY,
		maxX:       maxX,
		maxY:       maxY,
		image:      ebiten.NewImage((maxX-minX+1)*tileWidth+tileWidth/2, (maxY-minY+1)*tileHeight),
		isDirty:    true,
		tiles:      tiles,
		tileWidth:  tileWidth,
		tileHeight: tileHeight,
		colors:     colors}
}

func (d *MapDrawer) getTileImage(colorScheme, tileNum byte) *ebiten.Image {
	tileImage := d.tileImages[d.isNight][colorScheme][tileNum]
	if tileImage == nil {
		tile := d.tiles[tileNum]
		tile.Palette = d.colors.GetBackgroundForegroundColors(colorScheme, d.isNight != 0)
		tileImage = ebiten.NewImageFromImage(tile)
		d.tileImages[d.isNight][colorScheme][tileNum] = tileImage
	}
	return tileImage
}
func (d *MapDrawer) SetIsNight(isNight bool) {
	if isNight != (d.isNight != 0) {
		if isNight {
			d.isNight = 1
		} else {
			d.isNight = 0
		}
		d.isDirty = true
	}
}
func (d *MapDrawer) drawTileAt(tileNum byte, mapX, mapY int, screen *ebiten.Image) {
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
	if tileNum&63 < 48 {
		return d.getTileImage(tileNum/64, tileNum%64)
	}
	panic(tileNum)
}
func (d *MapDrawer) Draw() {
	if d.isDirty {
		for y := d.minY; y <= d.maxY; y++ {
			if y >= d.terrainMap.Height {
				break
			}
			for x := d.minX; x <= d.maxX; x++ {
				if x >= d.terrainMap.Width-y%2 {
					break
				}
				tileNum := d.terrainMap.GetTile(x, y)
				d.drawTileAt(tileNum, x, y, d.image)
			}
		}
		d.isDirty = false
	}
}
func (d *MapDrawer) GetSubImage(bounds image.Rectangle) *ebiten.Image {
	return d.image.SubImage(bounds).(*ebiten.Image)
}
