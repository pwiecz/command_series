package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/pwiecz/command_series/lib"
	"github.com/pwiecz/go-fltk"
)

var purple = imgui.Packed(color.NRGBA{100, 50, 225, 255})

type glTexture uint32
type MapWindow struct {
	*fltk.GlWindow
	context               *imgui.Context
	renderer              *OpenGL3
	firstShow             bool
	gameData              *lib.GameData
	scenarioData          *lib.ScenarioData
	selectedScenario      int
	colorSchemes          *lib.ColorSchemes
	zoom                  float32
	dx, dy                float64
	width, height         float32
	tileWidth, tileHeight float32
	isNight               bool
	tileImages            [2][4][48]glTexture
}

func NewMapWindow(x, y, w, h int) *MapWindow {
	win := &MapWindow{
		zoom:  1,
		width: float32(w), height: float32(h),
	}
	win.GlWindow = fltk.NewGlWindow(x, y, w, h, win.drawMap)
	win.GlWindow.SetEventHandler(win.handleEvent)
	win.GlWindow.SetResizeHandler(win.onResize)
	win.Resizable(win.GlWindow)
	win.firstShow = true
	return win
}

func (w *MapWindow) redraw() {
	fltk.Awake(w.Redraw)
}

func (w *MapWindow) SetGameData(gameData *lib.GameData, scenarioData *lib.ScenarioData, selectedScenario int) {
	w.gameData = gameData
	w.scenarioData = scenarioData
	w.selectedScenario = selectedScenario
	tileBounds := w.gameData.Sprites.TerrainTiles[0].Bounds()
	w.tileWidth, w.tileHeight = float32(tileBounds.Dx()), float32(tileBounds.Dy())
	w.zoom = lib.Min(
		w.width/((float32(w.gameData.Map.Width)+0.5)*w.tileWidth),
		w.height/(float32(w.gameData.Map.Height)*w.tileHeight))
	w.colorSchemes = lib.NewColorSchemes(
		&scenarioData.Data.DaytimePalette, &scenarioData.Data.NightPalette)
	for isNight := 0; isNight < 2; isNight++ {
		for colorScheme := 0; colorScheme < 4; colorScheme++ {
			for tileIndex := 0; tileIndex < 48; tileIndex++ {
				texture := w.tileImages[isNight][colorScheme][tileIndex]
				if texture > 0 {
					deleteTexture(texture)
					w.tileImages[isNight][colorScheme][tileIndex] = 0
				}
			}
		}
	}

	w.redraw()
}

func (w *MapWindow) drawMap() {
	if !w.ContextValid() {
		_, _, width, height := fltk.ScreenWorkArea(0 /* main screen */)
		context := imgui.CreateContext(nil)
		imgui.CurrentIO().SetDisplaySize(imgui.Vec2{X: float32(width), Y: float32(height)})
		renderer, err := NewOpenGL3(imgui.CurrentIO())
		if err != nil {
			panic(err)
		}
		w.renderer = renderer
		w.context, w.renderer = context, renderer
	}
	if w.gameData == nil {
		gl.ClearColor(1, 1, 1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		return
	}
	isNightIx := 0
	if w.isNight {
		isNightIx = 1
	}
	imgui.NewFrame()
	drawList := imgui.BackgroundDrawList()
	backgroundColor := w.colorSchemes.GetMapBackgroundColor(w.isNight)
	r, g, b, _ := backgroundColor.RGBA()
	gl.ClearColor(float32(r)/255, float32(g)/255, float32(b)/255, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	dy := float32(0)
	tileSize := imgui.Vec2{X: w.tileWidth * w.zoom, Y: w.tileWidth * w.zoom}
	for y := 0; y < w.gameData.Map.Width; y++ {
		dx := float32(y%2) * w.tileWidth / 2 * w.zoom
		for x := 0; x < w.gameData.Map.Height; x++ {
			tile := w.gameData.Map.GetTile(lib.MapCoords{X: x, Y: y})
			if tile%64 < 48 {
				colorScheme := tile / 64
				tileIndex := tile % 64
				texture := w.tileImages[isNightIx][colorScheme][tileIndex]
				if texture == 0 {
					tile := w.gameData.Sprites.TerrainTiles[tileIndex]
					tile.Palette = w.colorSchemes.GetBackgroundForegroundColors(colorScheme, w.isNight)
					texture = newTexture(tile)
					w.tileImages[isNightIx][colorScheme][tileIndex] = texture
				}
				tilePos := imgui.Vec2{X: dx, Y: dy}
				drawList.AddImage(imgui.TextureID(texture), tilePos, tilePos.Plus(tileSize))
			}
			dx += w.tileWidth * w.zoom
		}
		dy += w.tileHeight * w.zoom
	}
	scenario := w.gameData.Scenarios[w.selectedScenario]
	rangeMin := imgui.Vec2{
		X: (float32(scenario.MinX) + 0.5) * w.tileWidth * w.zoom,
		Y: float32(scenario.MinY) * w.tileWidth * w.zoom}
	rangeMax := imgui.Vec2{
		X: (float32(scenario.MaxX) + 1.5) * w.tileWidth * w.zoom,
		Y: float32(scenario.MaxY+1) * w.tileWidth * w.zoom}
	drawList.AddRect(rangeMin, rangeMax, purple)
	imgui.Render()
	size := [2]float32{w.width, w.height}
	w.renderer.Render(size, size, imgui.RenderedDrawData())
}

func (w *MapWindow) handleEvent(event fltk.Event) bool {
	switch event {
	case fltk.SHOW:
		if w.firstShow && w.IsShown() {
			w.firstShow = false
			w.MakeCurrent()
			if err := gl.Init(); err != nil {
				log.Fatal("Cannot initialize OpenGL", err)
			}
			w.redraw()
		}
	}
	return false
}

func (w *MapWindow) onResize() {
	w.width, w.height = float32(w.W()), float32(w.H())
	if w.gameData != nil {
		w.zoom = lib.Min(
			w.width/((float32(w.gameData.Map.Width)+0.5)*w.tileWidth),
			w.height/(float32(w.gameData.Map.Height)*w.tileHeight))
	}
	w.redraw()
}

func newTexture(img image.Image) glTexture {
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		if rgba.Stride != rgba.Rect.Size().X*4 {
			panic("unsupported stride")
		}
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return glTexture(texture)
}

func deleteTexture(tex glTexture) {
	texIx := uint32(tex)
	gl.DeleteTextures(1, &texIx)
}
