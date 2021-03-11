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

func (m *TerrainTypeMap) terrainOrUnitTypeAt(x, y int) int {
	return m.terrainOrUnitTypeAtIndex(m.terrainMap.CoordsToIndex(x, y))
}
func (m *TerrainTypeMap) terrainOrUnitTypeAtIndex(ix int) int {
	if ix < 0 || ix >= len(m.units) || m.units[ix] {
		return 7
	}
	return m.generic.TerrainTypes[m.terrainMap.getTileAtIndex(ix)&63]
}
func (m *TerrainTypeMap) terrainTypeAt(x, y int) int {
	return m.terrainTypeAtIndex(m.terrainMap.CoordsToIndex(x, y))
}
func (m *TerrainTypeMap) terrainTypeAtIndex(ix int) int {
	terrain := m.terrainMap.getTileAtIndex(ix)
	if terrain & 63 >= 48 {
		panic(terrain)
	}
	return m.generic.TerrainTypes[terrain & 63]
}
func (m *TerrainTypeMap) showUnit(unit Unit) {
	m.ShowUnitAt(unit.X, unit.Y)
}
func (m *TerrainTypeMap) hideUnit(unit Unit) {
	m.HideUnitAt(unit.X, unit.Y)
}
func (m *TerrainTypeMap) AreCoordsValid(x, y int) bool {
	return m.terrainMap.AreCoordsValid(x, y)
}
func (m *TerrainTypeMap) ShowUnitAt(x, y int) {
	m.units[m.terrainMap.CoordsToIndex(x/2, y)] = true
}
func (m *TerrainTypeMap) HideUnitAt(x, y int) {
	m.units[m.terrainMap.CoordsToIndex(x/2, y)] = false
}
func (m *TerrainTypeMap) ContainsUnit(x, y int) bool {
	return m.units[m.terrainMap.CoordsToIndex(x/2, y)]
}
