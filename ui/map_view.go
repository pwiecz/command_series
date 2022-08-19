package ui

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type MapView struct {
	minMapX, minMapY, maxMapX, maxMapY int // map bounds to draw (in map coordinates)
	cursorXY                           lib.MapCoords

	colors         *lib.ColorSchemes
	mapDrawer      *MapDrawer
	terrainTypeMap *lib.TerrainTypeMap
	units          *lib.Units
	unitSprites    *unitSprites
	icons          *[24]*image.Paletted

	tileWidth, tileHeight float64

	ebitenIcons       [24]*ebiten.Image
	cursorImage       *ebiten.Image
	iconAnimationStep int
	shownIcons        []*ebiten.Image
	iconXY            lib.MapCoords
	iconDx, iconDy    float64

	isNight bool

	x, y          float64
	width, height float64
	zoomX, zoomY  float64
	// Upper left pixel of map image
	subimageDx, subimageDy float64
}

func NewMapView(
	x, y float64,
	width, height int,
	terrainMap *lib.Map,
	terrainTypeMap *lib.TerrainTypeMap,
	units *lib.Units,
	minMapX, minMapY, maxMapX, maxMapY int,
	tiles *[48]*image.Paletted,
	unitSymbols *[16]*image.Paletted,
	unitIcons *[16]*image.Paletted,
	icons *[24]*image.Paletted,
	daytimePalette *[8]byte,
	nightPalette *[8]byte) *MapView {

	colors := lib.NewColorSchemes(daytimePalette, nightPalette)
	mapDrawer := NewMapDrawer(terrainMap, minMapX, minMapY, maxMapX, maxMapY, tiles, colors)
	unitSprites := newUnitSprites(unitSymbols, unitIcons, colors)
	tileBounds := tiles[0].Bounds()
	v := &MapView{
		x:              x,
		y:              y,
		width:          float64(width),
		height:         float64(height),
		minMapX:        minMapX,
		minMapY:        minMapY,
		maxMapX:        maxMapX,
		maxMapY:        maxMapY,
		cursorXY:       lib.MapCoords{X: minMapX, Y: minMapY},
		icons:          icons,
		tileWidth:      float64(tileBounds.Dx()),
		tileHeight:     float64(tileBounds.Dy()),
		colors:         colors,
		mapDrawer:      mapDrawer,
		terrainTypeMap: terrainTypeMap,
		units:          units,
		unitSprites:    unitSprites,
		zoomX:          2,
		zoomY:          1,
		subimageDx:     float64(tileBounds.Dx() / 2)}
	return v
}

func (v *MapView) GetCursorPosition() lib.MapCoords {
	return v.cursorXY
}
func (v *MapView) SetCursorPosition(xy lib.MapCoords) {
	v.cursorXY.Y = lib.Clamp(xy.Y, v.minMapY, v.maxMapY)
	v.cursorXY.X = lib.Clamp(xy.X, v.minMapX, v.maxMapX)
	v.makeCursorVisible()
}
func (v *MapView) makeCursorVisible() {
	for {
		cursorScreenX, cursorScreenY := v.MapCoordsToScreenCoords(v.cursorXY)
		// It's ok for cursor to be half width to the left from the edge,
		// so one AreScreenCoordsVisible is not enough.
		if v.AreScreenCoordsVisible(cursorScreenX, cursorScreenY) ||
			v.AreScreenCoordsVisible(cursorScreenX+v.tileWidth/2*v.zoomX, cursorScreenY) {
			break
		}
		if cursorScreenX < v.x-v.tileWidth*v.zoomX/2 {
			v.subimageDx -= v.tileWidth
		}
		if cursorScreenY < v.y {
			v.subimageDy -= v.tileHeight
		}
		if cursorScreenX >= v.width+v.x-v.tileWidth*v.zoomX {
			v.subimageDx += v.tileWidth
		}
		if cursorScreenY >= v.height+v.y-v.tileHeight*v.zoomY {
			v.subimageDy += v.tileHeight
		}
	}
}

func (v *MapView) SetUnitDisplay(unitDisplay lib.UnitDisplay) {
	v.unitSprites.SetUnitDisplay(unitDisplay)
}
func (v *MapView) SetIsNight(isNight bool) {
	v.isNight = isNight
	v.unitSprites.SetIsNight(isNight)
}
func (v *MapView) ScreenCoordsToUnitCoords(screenX, screenY int) lib.UnitCoords {
	imageX := (float64(screenX)-v.x)/v.zoomX + v.subimageDx
	imageY := (float64(screenY)-v.y)/v.zoomY + v.subimageDy
	// Cast to float64 to perform division rounding to floor instead of rounding to zero.
	y := int(math.Floor(imageY/v.tileHeight)) + v.minMapY
	x := int(math.Floor(imageX+float64(y%2)*v.tileWidth/2)/8)*2 - y%2 + v.minMapX*2
	return lib.UnitCoords{X: x, Y: y}
}
func (v *MapView) MapCoordsToScreenCoords(mapXY lib.MapCoords) (x, y float64) {
	imageX, imageY := v.mapDrawer.MapCoordsToImageCoords(mapXY)
	x = (float64(imageX)-v.subimageDx)*v.zoomX + v.x
	y = (float64(imageY)-v.subimageDy)*v.zoomY + v.y
	return
}
func (v *MapView) AreMapCoordsVisible(mapXY lib.MapCoords) bool {
	x, y := v.MapCoordsToScreenCoords(mapXY)
	return v.AreScreenCoordsVisible(x, y)
}
func (v *MapView) AreScreenCoordsVisible(x, y float64) bool {
	return x >= v.x && x < v.x+v.width && y >= v.y && y < v.y+v.height
}
func (v *MapView) DrawSpriteBetween(sprite *ebiten.Image, mapXY0, mapXY1 lib.MapCoords, alpha float64, screen *ebiten.Image) {
	x0, y0 := v.MapCoordsToScreenCoords(mapXY0)
	x1, y1 := v.MapCoordsToScreenCoords(mapXY1)
	x, y := x0+(x1-x0)*alpha, y0+(y1-y0)*alpha
	v.drawSpriteAtCoords(sprite, x, y, screen)
}
func (v *MapView) GetSpriteForUnit(unit lib.Unit) *ebiten.Image {
	return v.unitSprites.GetSpriteForUnit(unit)
}
func (v *MapView) GetSpriteFromIcon(icon lib.IconType) *ebiten.Image {
	ebitenIcon := v.ebitenIcons[icon]
	if ebitenIcon == nil {
		iconImage := v.icons[icon]
		ebitenIcon = ebiten.NewImageFromImage(iconImage)
		v.ebitenIcons[icon] = ebitenIcon
	}
	return ebitenIcon
}

func (v *MapView) drawSpriteAtCoords(sprite *ebiten.Image, x, y float64, screen *ebiten.Image) {
	var options ebiten.DrawImageOptions
	options.GeoM.Scale(v.zoomX, v.zoomY)
	options.GeoM.Translate(x, y)
	screen.DrawImage(sprite, &options)
}
func (v *MapView) Draw(screen *ebiten.Image) {
	subimageRect := image.Rect(int(v.subimageDx), int(v.subimageDy), int(v.subimageDx+v.width), int(v.subimageDy+v.height))
	mapSubImage := v.mapDrawer.GetMapImage(v.isNight).SubImage(subimageRect).(*ebiten.Image)
	var options ebiten.DrawImageOptions
	options.GeoM.Scale(v.zoomX, v.zoomY)
	options.GeoM.Translate(v.x, v.y)
	screen.DrawImage(mapSubImage, &options)
	for _, sideUnits := range v.units {
		for _, unit := range sideUnits {
			if !unit.IsInGame || !v.terrainTypeMap.ContainsUnit(unit.XY) {
				continue
			}
			x, y := v.MapCoordsToScreenCoords(unit.XY.ToMapCoords())
			if !v.AreScreenCoordsVisible(x, y) {
				continue
			}
			sprite := v.GetSpriteForUnit(unit)
			v.drawSpriteAtCoords(sprite, x, y, screen)
		}
	}
	if v.cursorImage == nil {
		cursorSprite := v.GetSpriteFromIcon(lib.Cursor)
		cursorBounds := cursorSprite.Bounds()
		// Cursor must be offset by 6,2 and scaled 2,1 to match the tile size and position.
		v.cursorImage = ebiten.NewImage(cursorBounds.Dx()*2, cursorBounds.Dy())
		var opts ebiten.DrawImageOptions
		opts.GeoM.Scale(2, 1)
		v.cursorImage.DrawImage(cursorSprite, &opts)
	}
	cursorX, cursorY := v.MapCoordsToScreenCoords(v.cursorXY)
	cursorOffsetX := 6 * v.zoomX
	cursorOffsetY := 2 * v.zoomY
	v.drawSpriteAtCoords(v.cursorImage, cursorX-cursorOffsetX, cursorY-cursorOffsetY, screen)
	if len(v.shownIcons) > 0 && v.AreMapCoordsVisible(v.iconXY) {
		iconX, iconY := v.MapCoordsToScreenCoords(v.iconXY)
		icon := v.shownIcons[(v.iconAnimationStep/4)%len(v.shownIcons)]
		v.drawSpriteAtCoords(icon, iconX+v.iconDx*v.zoomX, iconY+v.iconDy*v.zoomY, screen)
		v.iconAnimationStep++
	}
}
func (v *MapView) ShowIcon(icon lib.IconType, xy lib.MapCoords, dx, dy float64) {
	v.shownIcons = append(v.shownIcons[:0], v.GetSpriteFromIcon(icon))
	v.iconAnimationStep = 0
	v.iconXY = xy
	v.iconDx = dx
	v.iconDy = dy
}
func (v *MapView) ShowAnimatedIcon(icons []lib.IconType, xy lib.MapCoords, dx, dy float64) {
	v.shownIcons = v.shownIcons[:0]
	for _, icon := range icons {
		v.shownIcons = append(v.shownIcons, v.GetSpriteFromIcon(icon))
	}
	v.iconAnimationStep = 0
	v.iconXY = xy
	v.iconDx = dx
	v.iconDy = dy
}
func (v *MapView) HideIcon() {
	if len(v.shownIcons) > 0 {
		v.shownIcons = v.shownIcons[:0]
	}
}
