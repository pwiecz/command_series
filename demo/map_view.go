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
	unitSprites *[16]*image.Paletted
	palette     *[8]byte

	ebitenTiles   [4][48]*ebiten.Image
	ebitenSprites [4][16]*ebiten.Image
}

func NewMapView(terrainMap *data.Map, minX, minY, maxX, maxY int, tiles *[48]*image.Paletted,
	unitSprites *[16]*image.Paletted, palette *[8]byte) *MapView {
	v := &MapView{
		terrainMap:  terrainMap,
		minX:        minX,
		minY:        minY,
		maxX:        maxX,
		maxY:        maxY,
		tiles:       tiles,
		unitSprites: unitSprites,
		palette:     palette}
	return v
}
func (v *MapView) getTileImage(paletteNum, tileNum int) *ebiten.Image {
	ebitenTile := v.ebitenTiles[paletteNum][tileNum]
	if ebitenTile == nil {
		tile := v.tiles[tileNum]
		tile.Palette = GetPalette(paletteNum, v.palette)
		ebitenTile = ebiten.NewImageFromImage(tile)
		v.ebitenTiles[paletteNum][tileNum] = ebitenTile
	}
	return ebitenTile
}
func (v *MapView) getSpriteImage(paletteNum, spriteNum int) *ebiten.Image {
	ebitenSprite := v.ebitenSprites[paletteNum][spriteNum]
	if ebitenSprite == nil {
		sprite := v.unitSprites[spriteNum]
		sprite.Palette = GetPalette(paletteNum, v.palette)
		ebitenSprite = ebiten.NewImageFromImage(sprite)
		v.ebitenSprites[paletteNum][spriteNum] = ebitenSprite
	}
	return ebitenSprite
}
func (v *MapView) ToMapCoords(imageX, imageY int) (x, y int) {
	y = imageY/8 + v.minY
	x = ((imageX+(y%2)*4)/8)*2 - y%2 + v.minX*2
	return

}
func (v *MapView) SetUnitSprites(unitSprites *[16]*image.Paletted) {
	if unitSprites == v.unitSprites {
		return
	}
	v.unitSprites = unitSprites
	for paletteNum := 0; paletteNum < 4; paletteNum++ {
		for i := range unitSprites {
			v.ebitenSprites[paletteNum][i] = nil
		}
	}
}
func (v *MapView) SetPalette(palette *[8]byte) {
	if palette == v.palette {
		return
	}
	v.palette = palette
	for paletteNum := 0; paletteNum < 4; paletteNum++ {
		for i := range v.tiles {
			v.ebitenTiles[paletteNum][i] = nil
		}
		for i := range v.unitSprites {
			v.ebitenSprites[paletteNum][i] = nil
		}
	}
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
			} else {
				tileImage = v.getSpriteImage(tileNum/64, tileNum%16)
			}
			options.GeoM = geoM
			options.GeoM.Translate(float64(x*tileSize.X+(my%2)*tileSize.X/2), float64(y*tileSize.Y))
			screen.DrawImage(tileImage, options)
		}
	}
	options.GeoM = geoM
}
