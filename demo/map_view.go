package main

import "image"
import "image/color"
import "github.com/pwiecz/command_series/data"
import "github.com/hajimehoshi/ebiten"

type MapView struct {
	terrainMap             *data.Map
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	cursorX, cursorY       int

	tiles       *[48]*image.Paletted
	unitSymbols *[16]*image.Paletted
	unitIcons   *[16]*image.Paletted
	icons       *[24]*image.Paletted

	visibleBounds     image.Rectangle
	mapImage          *ebiten.Image
	isDirty           bool
	drawnTerrainTiles [][]byte
	subImage          *ebiten.Image

	tileWidth, tileHeight int

	isNight  int // 0 or 1
	useIcons bool

	daytimePalette *[8]byte
	nightPalette   *[8]byte

	colorSchemes [2][4][]color.Color

	ebitenTiles       [2][4][48]*ebiten.Image
	ebitenUnitSymbols [2][4][16]*ebiten.Image
	ebitenUnitIcons   [2][4][16]*ebiten.Image
	ebitenIcons       [24]*ebiten.Image
	cursorImage       *ebiten.Image
	iconAnimationStep int
	shownIcons        []*ebiten.Image
	iconX, iconY      int
}

func NewMapView(terrainMap *data.Map,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	unitSymbols *[16]*image.Paletted,
	unitIcons *[16]*image.Paletted,
	icons *[24]*image.Paletted,
	daytimePalette *[8]byte,
	nightPalette *[8]byte,
	size image.Point) *MapView {

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
		tileWidth:      tileBounds.Dx(),
		tileHeight:     tileBounds.Dy(),
		daytimePalette: daytimePalette,
		nightPalette:   nightPalette}
	v.visibleBounds = image.Rect(v.tileWidth/2, 0, v.tileWidth/2+size.X, size.Y)
	return v
}

func (v *MapView) getTileImage(colorScheme, tileNum byte) *ebiten.Image {
	ebitenTile := v.ebitenTiles[v.isNight][colorScheme][tileNum]
	if ebitenTile == nil {
		tile := v.tiles[tileNum]
		tile.Palette = v.getBackgroundForegroundColors(colorScheme)
		ebitenTile = ebiten.NewImageFromImage(tile)
		v.ebitenTiles[v.isNight][colorScheme][tileNum] = ebitenTile
	}
	return ebitenTile
}
func (v *MapView) getUnitSymbolImage(colorScheme, spriteNum byte) *ebiten.Image {
	ebitenSprite := v.ebitenUnitSymbols[v.isNight][colorScheme][spriteNum]
	if ebitenSprite == nil {
		sprite := v.unitSymbols[spriteNum]
		sprite.Palette = v.getBackgroundForegroundColors(colorScheme)
		ebitenSprite = ebiten.NewImageFromImage(sprite)
		v.ebitenUnitSymbols[v.isNight][colorScheme][spriteNum] = ebitenSprite
	}
	return ebitenSprite
}
func (v *MapView) getUnitIconImage(colorScheme, spriteNum byte) *ebiten.Image {
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
	imageX += v.visibleBounds.Min.X
	imageY += v.visibleBounds.Min.Y
	y = imageY/8 + v.minY
	x = ((imageX+(y%2)*4)/8)*2 - y%2 + v.minX*2
	return

}
func (v *MapView) GetCursorPosition() (int, int) {
	return v.cursorX, v.cursorY
}
func (v *MapView) SetCursorPosition(x, y int) {
	v.cursorX = data.Clamp(x, v.minX, v.maxX)
	v.cursorY = data.Clamp(y, v.minY, v.maxY)
	v.makeCursorVisible()
}
func (v *MapView) makeCursorVisible() {
	cursorScreenX, cursorScreenY := v.MapCoordsToScreenCoords(v.cursorX, v.cursorY)
	newBounds := v.visibleBounds
	if cursorScreenX < 0 {
		newBounds = newBounds.Add(image.Pt(cursorScreenX, 0))
	}
	if cursorScreenY < 0 {
		newBounds = newBounds.Add(image.Pt(0, cursorScreenY))
	}
	if cursorScreenX >= newBounds.Dx() {
		newBounds = newBounds.Add(image.Pt(cursorScreenX-newBounds.Dx()+v.tileWidth, 0))
	}
	if cursorScreenY >= newBounds.Dy() {
		newBounds = newBounds.Add(image.Pt(0, cursorScreenY-newBounds.Dy()+v.tileHeight))
	}
	if newBounds.Min.X < v.tileWidth/2 {
		newBounds = newBounds.Add(image.Pt(v.tileWidth/2-newBounds.Min.X, 0))
	}

	if !newBounds.Eq(v.visibleBounds) {
		v.visibleBounds = newBounds
		if v.mapImage != nil {
			v.subImage = v.mapImage.SubImage(v.visibleBounds).(*ebiten.Image)
		}
	}
	return
}

func (v *MapView) SetUseIcons(useIcons bool) {
	if v.useIcons != useIcons {
		v.useIcons = useIcons
		v.isDirty = true
	}
}
func (v *MapView) SetIsNight(isNight bool) {
	if isNight != (v.isNight == 1) {
		if isNight {
			v.isNight = 1
		} else {
			v.isNight = 0
		}
		v.isDirty = true
	}
}
func (v *MapView) getBackgroundForegroundColors(colorScheme byte) []color.Color {
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

func (v *MapView) MapCoordsToImageCoords(mapX, mapY int) (x, y int) {
	x = (mapX-v.minX)*v.tileWidth + (mapY%2)*v.tileWidth/2
	y = (mapY - v.minY) * v.tileHeight
	return
}
func (v *MapView) MapCoordsToScreenCoords(mapX, mapY int) (x, y int) {
	x = (mapX-v.minX)*v.tileWidth + (mapY%2)*v.tileWidth/2 - v.visibleBounds.Min.X
	y = (mapY-v.minY)*v.tileHeight - v.visibleBounds.Min.Y
	return
}
func (v *MapView) AreMapCoordsVisible(mapX, mapY int) bool {
	x, y := v.MapCoordsToImageCoords(mapX, mapY)
	return v.AreScreenCoordsVisible(x, y)
}
func (v *MapView) AreScreenCoordsVisible(x, y int) bool {
	return image.Pt(x, y).In(v.visibleBounds)
}
func (v *MapView) DrawTileAt(tileNum byte, mapX, mapY int, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	x, y := v.MapCoordsToImageCoords(mapX, mapY)
	v.drawTileAtImageCoords(tileNum, x, y, screen, options)
}
func (v *MapView) DrawSpriteBetween(sprite *ebiten.Image, mapX0, mapY0, mapX1, mapY1 int, alpha float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	x0, y0 := v.MapCoordsToScreenCoords(mapX0, mapY0)
	x1, y1 := v.MapCoordsToScreenCoords(mapX1, mapY1)
	x, y := float64(x0)+float64(x1-x0)*alpha, float64(y0)+float64(y1-y0)*alpha
	v.drawSpriteAtCoords(sprite, x, y, screen, options)
}
func (v *MapView) DrawSpriteAt(sprite *ebiten.Image, mapX, mapY int, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	x, y := v.MapCoordsToScreenCoords(mapX, mapY)
	v.drawSpriteAtCoords(sprite, float64(x), float64(y), screen, options)
}

func (v *MapView) GetSpriteFromTileNum(tileNum byte) *ebiten.Image {
	if tileNum%64 < 48 {
		return v.getTileImage(tileNum/64, tileNum%64)
	} else if v.useIcons {
		return v.getUnitIconImage(tileNum/64, tileNum%16)
	} else {
		return v.getUnitSymbolImage(tileNum/64, tileNum%16)
	}
}
func (v *MapView) GetSpriteFromIcon(icon data.IconType) *ebiten.Image {
	ebitenIcon := v.ebitenIcons[icon]
	if ebitenIcon == nil {
		iconImage := v.icons[icon]
		ebitenIcon = ebiten.NewImageFromImage(iconImage)
		v.ebitenIcons[icon] = ebitenIcon
	}
	return ebitenIcon
}

func (v *MapView) drawTileAtImageCoords(tileNum byte, x, y int, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	tileImage := v.GetSpriteFromTileNum(tileNum)
	v.drawSpriteAtCoords(tileImage, float64(x), float64(y), screen, options)
}
func (v *MapView) drawSpriteAtCoords(sprite *ebiten.Image, x, y float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	geoM := options.GeoM
	options.GeoM.Reset()
	options.GeoM.Translate(x, y)
	options.GeoM.Concat(geoM)
	screen.DrawImage(sprite, options)
	options.GeoM = geoM
}
func (v *MapView) getDrawnTile(x, y int) byte {
	dx, dy := x-v.minX, y-v.minY
	for dy >= len(v.drawnTerrainTiles) {
		v.drawnTerrainTiles = append(v.drawnTerrainTiles, make([]byte, v.maxX-v.minX+1))
	}
	for dx >= len(v.drawnTerrainTiles[dy]) {
		v.drawnTerrainTiles[dy] = append(v.drawnTerrainTiles[dy], 0)
	}
	return v.drawnTerrainTiles[dy][dx]
}
func (v *MapView) setDrawnTile(x, y int, tile byte) {
	dx, dy := x-v.minX, y-v.minY
	for dy >= len(v.drawnTerrainTiles) {
		v.drawnTerrainTiles = append(v.drawnTerrainTiles, make([]byte, v.maxX-v.minX+1))
	}
	for dx >= len(v.drawnTerrainTiles[dy]) {
		v.drawnTerrainTiles[dy] = append(v.drawnTerrainTiles[dy], 0)
	}
	v.drawnTerrainTiles[dy][dx] = tile
}
func (v *MapView) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	if v.mapImage == nil {
		v.mapImage = ebiten.NewImage((v.maxX-v.minX+1)*v.tileWidth+v.tileWidth/2, (v.maxY-v.minY+1)*v.tileHeight)
		v.subImage = v.mapImage.SubImage(v.visibleBounds).(*ebiten.Image)
		v.isDirty = true
	}
	var opts ebiten.DrawImageOptions
	for y := v.minY; y <= v.maxY; y++ {
		if y >= v.terrainMap.Height {
			break
		}
		for x := v.minX; x <= v.maxX; x++ {
			if x >= v.terrainMap.Width-y%2 {
				break
			}
			tileNum := v.terrainMap.GetTile(x, y)
			if v.isDirty || tileNum != v.getDrawnTile(x, y) {
				v.DrawTileAt(tileNum, x, y, v.mapImage, &opts)
				v.setDrawnTile(x, y, tileNum)
			}
		}
	}
	v.isDirty = false
	screen.DrawImage(v.subImage, options)
	if v.cursorImage == nil {
		cursorSprite := v.GetSpriteFromIcon(data.Cursor)
		cursorBounds := cursorSprite.Bounds()
		v.cursorImage = ebiten.NewImage(cursorBounds.Dx()*2, cursorBounds.Dy())
		var opts ebiten.DrawImageOptions
		opts.GeoM.Scale(2, 1)
		v.cursorImage.DrawImage(cursorSprite, &opts)
	}
	cursorX, cursorY := v.MapCoordsToScreenCoords(v.cursorX, v.cursorY)
	v.drawSpriteAtCoords(v.cursorImage, float64(cursorX-6), float64(cursorY-2), screen, options)
	if len(v.shownIcons) > 0 && v.AreMapCoordsVisible(v.iconX, v.iconY) {
		iconX, iconY := v.MapCoordsToScreenCoords(v.iconX, v.iconY)
		icon := v.shownIcons[(v.iconAnimationStep/4)%len(v.shownIcons)]
		v.drawSpriteAtCoords(icon, float64(iconX), float64(iconY-5), screen, options)
		v.iconAnimationStep++
	}
	return
}
func (v *MapView) ShowIcon(icon data.IconType, x, y int) {
	v.shownIcons = append(v.shownIcons[:0], v.GetSpriteFromIcon(icon))
	v.iconAnimationStep = 0
	v.iconX = x
	v.iconY = y
}
func (v *MapView) ShowAnimatedIcon(icons []data.IconType, x, y int) {
	v.shownIcons = v.shownIcons[:0]
	for _, icon := range icons {
		v.shownIcons = append(v.shownIcons, v.GetSpriteFromIcon(icon))
	}
	v.iconAnimationStep = 0
	v.iconX = x
	v.iconY = y
}
func (v *MapView) HideIcon() {
	if len(v.shownIcons) > 0 {
		v.shownIcons = v.shownIcons[:0]
	}
}
