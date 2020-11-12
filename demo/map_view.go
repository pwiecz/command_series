package main

import "fmt"
import "image"
import "image/color"
import "image/draw"
import "github.com/pwiecz/command_series/data"
import "github.com/hajimehoshi/ebiten"

type MapView struct {
	terrainMap             *data.Map
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	dx, dy                 int
	tiles                  *[48]*image.Paletted
	unitSprites            *[16]*image.Paletted
	palette                *[8]byte
	mapImage               *image.NRGBA
	ebitenImage            *ebiten.Image
	isDirty                bool
}

func NewMapView(terrainMap *data.Map, minX, minY, maxX, maxY int, tiles *[48]*image.Paletted,
	unitSprites *[16]*image.Paletted, palette *[8]byte) *MapView {
	return &MapView{
		terrainMap:  terrainMap,
		minX:        minX,
		minY:        minY,
		maxX:        maxX,
		maxY:        maxY,
		tiles:       tiles,
		unitSprites: unitSprites,
		palette:     palette,
		isDirty:     true}
}
func (v *MapView) ToMapCoords(imageX, imageY int) (x, y int) {
	y = imageY/8 + v.minY
	x = ((imageX+(y%2)*4)/8)*2 - y%2 + v.minX*2
	return

}
func (v *MapView) SetTiles(tiles *[48]*image.Paletted) {
	if tiles == v.tiles {
		return
	}
	v.tiles = tiles
	v.isDirty = true
}
func (v *MapView) SetUnitSprites(unitSprites *[16]*image.Paletted) {
	if unitSprites == v.unitSprites {
		return
	}
	v.unitSprites = unitSprites
	v.isDirty = true
}
func (v *MapView) SetPalette(palette *[8]byte) {
	if palette == v.palette {
		return
	}
	fmt.Println("Palette change..")
	v.palette = palette
	v.isDirty = true
}
func (v *MapView) Redraw() {
	v.isDirty = true
}
func GetPalette(n int, palette *[8]byte) []color.Color {
	pal := make([]color.Color, 2)
	// just guessing here
	pal[0] = &data.RGBPalette[palette[2]]
	switch n {
	case 0:
		pal[1] = &data.RGBPalette[palette[3]] // or 7
	case 1:
		pal[1] = &data.RGBPalette[palette[6]]
	case 2:
		pal[1] = &data.RGBPalette[palette[0]]
	case 3:
		pal[1] = &data.RGBPalette[palette[4]]
	}
	return pal
}
func (v *MapView) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	if v.isDirty || v.mapImage == nil {
		tileBounds := v.tiles[0].Bounds()
		tileSize := tileBounds.Max.Sub(tileBounds.Min)
		imageWidth, imageHeight := tileSize.X*v.terrainMap.Width, tileSize.Y*v.terrainMap.Height
		if v.mapImage == nil || v.mapImage.Bounds().Max != image.Pt(imageWidth, imageHeight) {
			v.mapImage = image.NewNRGBA(image.Rect(0, 0, imageWidth, imageHeight))
		}
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
				topLeft := image.Point{x*tileSize.X + (y%2)*tileSize.X/2, y * tileSize.Y}
				tileRect := image.Rectangle{topLeft, topLeft.Add(tileSize)}
				var tileImage image.Paletted
				if tileNum%64 < 48 {
					tileImage = *v.tiles[tileNum%64]
				} else {
					tileImage = *v.unitSprites[tileNum%16]
				}
				tileImage.Palette = GetPalette(tileNum/64, v.palette)
				draw.Draw(v.mapImage, tileRect, &tileImage, image.Point{}, draw.Over)
			}
		}
		v.ebitenImage = ebiten.NewImageFromImage(v.mapImage)
		v.isDirty = false
	}
	screen.DrawImage(v.ebitenImage, options)
}
