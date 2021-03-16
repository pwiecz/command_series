package lib

type TerrainTypeMap struct {
	terrainMap *Map
	units      []bool
	generic    *Generic
}

func newTerrainTypeMap(terrainMap *Map, generic *Generic) *TerrainTypeMap {
	return &TerrainTypeMap{
		terrainMap: terrainMap,
		units:      make([]bool, len(terrainMap.terrain)),
		generic:    generic,
	}
}

func (m *TerrainTypeMap) terrainOrUnitTypeAt(xy UnitCoords) int {
	return m.terrainOrUnitTypeAtIndex(m.terrainMap.CoordsToIndex(xy.ToMapCoords()))
}
func (m *TerrainTypeMap) terrainOrUnitTypeAtIndex(ix int) int {
	if ix < 0 || ix >= len(m.units) || m.units[ix] {
		return 7
	}
	return m.generic.TerrainTypes[m.terrainMap.getTileAtIndex(ix)&63]
}
func (m *TerrainTypeMap) terrainTypeAt(xy UnitCoords) int {
	return m.terrainTypeAtIndex(m.terrainMap.CoordsToIndex(xy.ToMapCoords()))
}
func (m *TerrainTypeMap) terrainTypeAtIndex(ix int) int {
	terrain := m.terrainMap.getTileAtIndex(ix)
	if terrain&63 >= 48 {
		panic(terrain)
	}
	return m.generic.TerrainTypes[terrain&63]
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
