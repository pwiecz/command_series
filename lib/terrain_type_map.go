package lib

type TerrainTypeMap struct {
	terrainMap *Map
	units      []bool
	generic    *Generic
}

func newTerrainTypeMap(terrainMap *Map, generic *Generic) *TerrainTypeMap {
	return &TerrainTypeMap{
		terrainMap: terrainMap,
		units:      make([]bool, terrainMap.Width*terrainMap.Height),
		generic:    generic,
	}
}

func (m *TerrainTypeMap) terrainOrUnitTypeAt(xy UnitCoords) int {
	if !m.AreCoordsValid(xy.ToMapCoords()) {
		return 7
	}
	mapCoords := xy.ToMapCoords()
	ix := m.terrainMap.CoordsToIndex(mapCoords)
	if ix < 0 || ix >= len(m.units) || m.units[ix] {
		return 7
	}
	return m.generic.TerrainTypes[m.terrainMap.GetTile(mapCoords)%64]
}

func (m *TerrainTypeMap) terrainTypeAt(xy UnitCoords) int {
	mapCoords := xy.ToMapCoords()
	return m.generic.TerrainTypes[m.terrainMap.GetTile(mapCoords)%64]
}

func (m *TerrainTypeMap) showUnit(unit Unit) {
	m.ShowUnitAt(unit.XY)
}
func (m *TerrainTypeMap) hideUnit(unit Unit) {
	m.HideUnitAt(unit.XY)
}
func (m *TerrainTypeMap) AreCoordsValid(xy MapCoords) bool {
	return m.terrainMap.AreCoordsValid(xy)
}
func (m *TerrainTypeMap) ShowUnitAt(xy UnitCoords) {
	m.units[m.terrainMap.CoordsToIndex(xy.ToMapCoords())] = true
}
func (m *TerrainTypeMap) HideUnitAt(xy UnitCoords) {
	m.units[m.terrainMap.CoordsToIndex(xy.ToMapCoords())] = false
}
func (m *TerrainTypeMap) ContainsUnit(xy UnitCoords) bool {
	return m.units[m.terrainMap.CoordsToIndex(xy.ToMapCoords())]
}
