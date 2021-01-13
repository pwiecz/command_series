package main

import (
	"image"

	"github.com/hajimehoshi/ebiten"
	"github.com/pwiecz/command_series/lib"
)

type MapView struct {
	minX, minY, maxX, maxY int // map bounds to draw in map coordinates
	cursorX, cursorY       int

	mapDrawer *MapDrawer
	icons     *[24]*image.Paletted

	visibleBounds image.Rectangle
	subImage      *ebiten.Image

	tileWidth, tileHeight int

	ebitenIcons       [24]*ebiten.Image
	cursorImage       *ebiten.Image
	iconAnimationStep int
	shownIcons        []*ebiten.Image
	iconX, iconY      int
	iconDx, iconDy    float64
}

func NewMapView(
	width, height int,
	terrainMap *lib.Map,
	minX, minY, maxX, maxY int,
	tiles *[48]*image.Paletted,
	unitSymbols *[16]*image.Paletted,
	unitIcons *[16]*image.Paletted,
	icons *[24]*image.Paletted,
	daytimePalette *[8]byte,
	nightPalette *[8]byte) *MapView {

	tileBounds := tiles[0].Bounds()

	drawer := NewMapDrawer(terrainMap, minX, minY, maxX, maxY, tiles, unitSymbols, unitIcons,
		daytimePalette, nightPalette)
	v := &MapView{
		minX:       minX,
		minY:       minY,
		maxX:       maxX,
		maxY:       maxY,
		cursorX:    minX,
		cursorY:    minY,
		icons:      icons,
		tileWidth:  tileBounds.Dx(),
		tileHeight: tileBounds.Dy(),
		mapDrawer:  drawer}
	v.visibleBounds = image.Rect(v.tileWidth/2, 0, v.tileWidth/2+width, height)
	v.subImage = drawer.GetSubImage(v.visibleBounds)
	return v
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
	v.cursorX = lib.Clamp(x, v.minX, v.maxX)
	v.cursorY = lib.Clamp(y, v.minY, v.maxY)
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
		v.subImage = v.mapDrawer.GetSubImage(v.visibleBounds)
	}
	return
}

func (v *MapView) SetUnitDisplay(unitDisplay lib.UnitDisplay) {
	v.mapDrawer.SetUnitDisplay(unitDisplay)
}
func (v *MapView) SetIsNight(isNight bool) {
	v.mapDrawer.SetIsNight(isNight)
}
func (v *MapView) MapCoordsToScreenCoords(mapX, mapY int) (x, y int) {
	x, y = v.mapDrawer.MapCoordsToImageCoords(mapX, mapY)
	x -= v.visibleBounds.Min.X
	y -= v.visibleBounds.Min.Y
	return
}
func (v *MapView) AreMapCoordsVisible(mapX, mapY int) bool {
	x, y := v.mapDrawer.MapCoordsToImageCoords(mapX, mapY)
	return v.AreScreenCoordsVisible(x, y)
}
func (v *MapView) AreScreenCoordsVisible(x, y int) bool {
	return image.Pt(x, y).In(v.visibleBounds)
}
func (v *MapView) DrawSpriteBetween(sprite *ebiten.Image, mapX0, mapY0, mapX1, mapY1 int, alpha float64, screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	x0, y0 := v.MapCoordsToScreenCoords(mapX0, mapY0)
	x1, y1 := v.MapCoordsToScreenCoords(mapX1, mapY1)
	x, y := float64(x0)+float64(x1-x0)*alpha, float64(y0)+float64(y1-y0)*alpha
	v.drawSpriteAtCoords(sprite, x, y, screen, options)
}

func (v *MapView) GetSpriteFromTileNum(tileNum byte) *ebiten.Image {
	return v.mapDrawer.GetSpriteFromTileNum(tileNum)
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
	if v.cursorImage == nil {
		cursorSprite := v.GetSpriteFromIcon(lib.Cursor)
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
		v.drawSpriteAtCoords(icon, float64(iconX)+v.iconDx, float64(iconY)+v.iconDy, screen, options)
		v.iconAnimationStep++
	}
	return
}
func (v *MapView) ShowIcon(icon lib.IconType, x, y int, dx, dy float64) {
	v.shownIcons = append(v.shownIcons[:0], v.GetSpriteFromIcon(icon))
	v.iconAnimationStep = 0
	v.iconX = x
	v.iconY = y
	v.iconDx = dx
	v.iconDy = dy
}
func (v *MapView) ShowAnimatedIcon(icons []lib.IconType, x, y int, dx, dy float64) {
	v.shownIcons = v.shownIcons[:0]
	for _, icon := range icons {
		v.shownIcons = append(v.shownIcons, v.GetSpriteFromIcon(icon))
	}
	v.iconAnimationStep = 0
	v.iconX = x
	v.iconY = y
	v.iconDx = dx
	v.iconDy = dy
}
func (v *MapView) HideIcon() {
	if len(v.shownIcons) > 0 {
		v.shownIcons = v.shownIcons[:0]
	}
}
