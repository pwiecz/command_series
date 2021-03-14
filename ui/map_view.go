package ui

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type MapView struct {
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	cursorXY               lib.MapCoords

	colors         *colorSchemes
	mapDrawer      *MapDrawer
	terrainTypeMap *lib.TerrainTypeMap
	units          *lib.Units
	unitSprites    *unitSprites
	icons          *[24]*image.Paletted

	visibleBounds image.Rectangle
	subImage      *ebiten.Image

	tileWidth, tileHeight int

	ebitenIcons       [24]*ebiten.Image
	cursorImage       *ebiten.Image
	iconAnimationStep int
	shownIcons        []*ebiten.Image
	iconXY            lib.MapCoords
	iconDx, iconDy    float64
}

func NewMapView(
	width, height int,
	terrainMap *lib.Map,
	terrainTypeMap *lib.TerrainTypeMap,
	units *lib.Units,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	unitSymbols *[16]*image.Paletted,
	unitIcons *[16]*image.Paletted,
	icons *[24]*image.Paletted,
	daytimePalette *[8]byte,
	nightPalette *[8]byte) *MapView {

	tileBounds := tiles[0].Bounds()
	colors := newColorSchemes(daytimePalette, nightPalette)
	mapDrawer := NewMapDrawer(terrainMap, minX, minY, maxX, maxY, tiles, colors)
	unitSprites := newUnitSprites(unitSymbols, unitIcons, colors)
	v := &MapView{
		minX:           minX,
		minY:           minY,
		maxX:           maxX,
		maxY:           maxY,
		cursorXY:       lib.MapCoords{minX, minY},
		icons:          icons,
		tileWidth:      tileBounds.Dx(),
		tileHeight:     tileBounds.Dy(),
		colors:         colors,
		mapDrawer:      mapDrawer,
		terrainTypeMap: terrainTypeMap,
		units:          units,
		unitSprites:    unitSprites}
	v.visibleBounds = image.Rect(v.tileWidth/2, 0, v.tileWidth/2+width, height)
	v.subImage = mapDrawer.GetSubImage(v.visibleBounds)
	return v
}

func (v *MapView) ToUnitCoords(imageX, imageY int) lib.UnitCoords {
	imageX += v.visibleBounds.Min.X
	imageY += v.visibleBounds.Min.Y
	// Cast to float64 to perform division rounding to floor instead of rounding to zero.
	y := int(math.Floor(float64(imageY)/8)) + v.minY
	x := int(math.Floor(float64(imageX+(y%2)*4)/8))*2 - y%2 + v.minX*2
	return lib.UnitCoords{x, y}
}
func (v *MapView) GetCursorPosition() lib.MapCoords {
	return v.cursorXY
}
func (v *MapView) SetCursorPosition(xy lib.MapCoords) {
	v.cursorXY = lib.MapCoords{lib.Clamp(xy.X, v.minX, v.maxX), lib.Clamp(xy.Y, v.minY, v.maxY)}
	v.makeCursorVisible()
}
func (v *MapView) makeCursorVisible() {
	cursorScreenX, cursorScreenY := v.MapCoordsToScreenCoords(v.cursorXY)
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
		v.subImage = v.mapDrawer.GetSubImage(v.visibleBounds)
	}
	return
}

func (v *MapView) SetUnitDisplay(unitDisplay lib.UnitDisplay) {
	v.unitSprites.SetUnitDisplay(unitDisplay)
}
func (v *MapView) SetIsNight(isNight bool) {
	v.mapDrawer.SetIsNight(isNight)
	v.unitSprites.SetIsNight(isNight)
}
func (v *MapView) MapCoordsToScreenCoords(mapXY lib.MapCoords) (x, y int) {
	x, y = v.mapDrawer.MapCoordsToImageCoords(mapXY)
	x -= v.visibleBounds.Min.X
	y -= v.visibleBounds.Min.Y
	return
}
func (v *MapView) AreMapCoordsVisible(mapXY lib.MapCoords) bool {
	x, y := v.mapDrawer.MapCoordsToImageCoords(mapXY)
	return v.AreScreenCoordsVisible(x, y)
}
func (v *MapView) AreScreenCoordsVisible(x, y int) bool {
	return x >= 0 && x < v.visibleBounds.Dx() && y >= 0 && y < v.visibleBounds.Dy()
}
func (v *MapView) DrawSpriteBetween(sprite *ebiten.Image, mapXY0, mapXY1 lib.MapCoords, alpha float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	x0, y0 := v.MapCoordsToScreenCoords(mapXY0)
	x1, y1 := v.MapCoordsToScreenCoords(mapXY1)
	x, y := float64(x0)+float64(x1-x0)*alpha, float64(y0)+float64(y1-y0)*alpha
	v.drawSpriteAtCoords(sprite, x, y, screen, options)
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

func (v *MapView) drawSpriteAtCoords(sprite *ebiten.Image, x, y float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	geoM := options.GeoM
	options.GeoM.Reset()
	options.GeoM.Translate(x, y)
	options.GeoM.Concat(geoM)
	screen.DrawImage(sprite, options)
	options.GeoM = geoM
}
func (v *MapView) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	v.mapDrawer.Draw()
	screen.DrawImage(v.subImage, options)
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
			v.drawSpriteAtCoords(sprite, float64(x), float64(y), screen, options)
		}
	}
	if v.cursorImage == nil {
		cursorSprite := v.GetSpriteFromIcon(lib.Cursor)
		cursorBounds := cursorSprite.Bounds()
		v.cursorImage = ebiten.NewImage(cursorBounds.Dx()*2, cursorBounds.Dy())
		var opts ebiten.DrawImageOptions
		opts.GeoM.Scale(2, 1)
		v.cursorImage.DrawImage(cursorSprite, &opts)
	}
	cursorX, cursorY := v.MapCoordsToScreenCoords(v.cursorXY)
	v.drawSpriteAtCoords(v.cursorImage, float64(cursorX-6), float64(cursorY-2), screen, options)
	if len(v.shownIcons) > 0 && v.AreMapCoordsVisible(v.iconXY) {
		iconX, iconY := v.MapCoordsToScreenCoords(v.iconXY)
		icon := v.shownIcons[(v.iconAnimationStep/4)%len(v.shownIcons)]
		v.drawSpriteAtCoords(icon, float64(iconX)+v.iconDx, float64(iconY)+v.iconDy, screen, options)
		v.iconAnimationStep++
	}
	return
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
