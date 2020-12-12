package main

import "github.com/hajimehoshi/ebiten"
import "github.com/pwiecz/command_series/data"

type OverviewMap struct {
	image        *ebiten.Image
	terrainMap   *data.Map
	generic      *data.Generic
	scenarioData *data.ScenarioData
	options      *Options
	units        *[2][]data.Unit
	cycle        int
}

func NewOverviewMap(terrainMap *data.Map, units *[2][]data.Unit, generic *data.Generic, scenarioData *data.ScenarioData, options *Options) *OverviewMap {
	return &OverviewMap{
		terrainMap:   terrainMap,
		units:        units,
		generic:      generic,
		scenarioData: scenarioData,
		options:      options}
}

func (m *OverviewMap) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	if m.image == nil {
		m.image = ebiten.NewImage(m.terrainMap.Width, m.terrainMap.Height)
		m.image.Fill(data.RGBPalette[14])
		for y := 0; y < m.terrainMap.Width; y++ {
			for x := 0; x < m.terrainMap.Height; x++ {
				if !m.terrainMap.AreCoordsValid(x, y) {
					continue
				}
				terrainTile := m.terrainMap.GetTile(x, y)
				terrainType := m.generic.TerrainTypes[terrainTile%64]
				col := m.generic.Data60[terrainType/2]
				if terrainType%2 == 0 {
					col /= 16
				}
				// The logic of picking colors is a hack made to match the original look.
				// There's some logic behind the colors, chosen but the condition itself is just pure hack.
				if col&15 == 6 {
					m.image.Set(x, y, data.RGBPalette[134])
				} else {
					m.image.Set(x, y, data.RGBPalette[14])
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
			if unit.IsInGame && (unit.InContactWithEnemy || unit.SeenByEnemy || m.options.IsPlayerControlled(side) || m.options.Intelligence == Full) {
				m.image.Set(unit.X/2, unit.Y, data.RGBPalette[color])
			}
		}
	}
	screen.DrawImage(m.image, opts)
}

func (m *OverviewMap) Update() {
	m.cycle = (m.cycle + 1) % 44
}
