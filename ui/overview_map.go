package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/lib"
)

type OverviewMap struct {
	image         *ebiten.Image
	terrainMap    *lib.Map
	generic       *lib.Generic
	scenarioData  *lib.Data
	units         *lib.Units
	isUnitVisible func(lib.Unit) bool
	cycle         int
}

func NewOverviewMap(terrainMap *lib.Map, units *lib.Units, generic *lib.Generic, scenarioData *lib.Data, isUnitVisible func(lib.Unit) bool) *OverviewMap {
	return &OverviewMap{
		terrainMap:    terrainMap,
		units:         units,
		generic:       generic,
		scenarioData:  scenarioData,
		isUnitVisible: isUnitVisible}
}

func (m *OverviewMap) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if m.image == nil {
		m.image = ebiten.NewImage(m.terrainMap.Width, m.terrainMap.Height)
		m.image.Fill(lib.RGBPalette[14])
		for y := 0; y < m.terrainMap.Width; y++ {
			for x := 0; x < m.terrainMap.Height; x++ {
				xy := lib.MapCoords{x, y}
				if !m.terrainMap.AreCoordsValid(xy) {
					continue
				}
				terrainTile := m.terrainMap.GetTile(xy)
				terrainType := m.generic.TerrainTypes[terrainTile%64]
				col := m.generic.Data60[terrainType/2]
				if terrainType%2 == 0 {
					col /= 16
				}
				// The logic of picking colors is a hack made to match the original look.
				// There's some logic behind the colors, chosen but the condition itself is just pure hack.
				if col&15 == 6 {
					m.image.Set(x, y, lib.RGBPalette[134])
				} else {
					m.image.Set(x, y, lib.RGBPalette[14])
				}
			}
		}
	}
	var colors [2]int
	if m.cycle < 22 {
		colors[0] = m.cycle / 2
		colors[1] = 11
	} else {
		colors[0] = 11
		colors[1] = m.cycle/2 - 11
	}
	for side, sideUnits := range m.units {
		color := m.scenarioData.SideColor[side]*16 + colors[side]
		for _, unit := range sideUnits {
			if m.isUnitVisible(unit) {
				xy := unit.XY.ToMapCoords()
				m.image.Set(xy.X, xy.Y, lib.RGBPalette[color])
			}
		}
	}
	screen.DrawImage(m.image, opts)
}

func (m *OverviewMap) Update() {
	m.cycle = (m.cycle + 1) % 44
}
