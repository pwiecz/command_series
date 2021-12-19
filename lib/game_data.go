package lib

import (
	"fmt"
	"io/fs"
)

type GameData struct {
	Game           Game
	Scenarios      []Scenario
	Sprites        *Sprites
	Icons          *Icons
	Map            *Map
	Generic        *Generic
	TerrainTypeMap *TerrainTypeMap
	Hexes          *Hexes
}
type ScenarioData struct {
	Variants []Variant
	Generals *Generals
	Terrain  *Terrain
	Data     *Data
	Units    *Units
}

func LoadGameData(fsys fs.FS) (*GameData, error) {
	game, err := DetectGame(fsys)
	if err != nil {
		return nil, fmt.Errorf("error detecting game, %v", err)
	}
	scenarios, err := ReadScenarios(fsys)
	if err != nil {
		return nil, fmt.Errorf("error loading scenarios, %v", err)
	}
	sprites, err := ReadSprites(fsys)
	if err != nil {
		return nil, fmt.Errorf("error loading sprites, %v", err)
	}
	icons, err := ReadIcons(fsys)
	if err != nil {
		return nil, fmt.Errorf("error loading icons, %v", err)
	}
	terrainMap, err := ReadMap(fsys, game)
	if err != nil {
		return nil, fmt.Errorf("error loading map, %v", err)
	}
	generic, err := ReadGeneric(fsys, game)
	if err != nil {
		return nil, fmt.Errorf("error loading generic, %v", err)
	}
	hexes, err := ReadHexes(fsys)
	if err != nil {
		return nil, fmt.Errorf("error loading hexes, %v", err)
	}
	gameData := &GameData{
		Game:           game,
		Scenarios:      scenarios,
		Sprites:        sprites,
		Icons:          icons,
		Map:            terrainMap,
		Generic:        generic,
		TerrainTypeMap: newTerrainTypeMap(terrainMap, generic),
		Hexes:          hexes}
	return gameData, nil

}

func LoadScenarioData(fsys fs.FS, filePrefix string) (*ScenarioData, error) {
	game, err := FilePrefixToGame(filePrefix)
	if err != nil {
		return nil, err
	}
	variantsFilename := filePrefix + ".VAR"
	variants, err := ReadVariants(fsys, variantsFilename)
	if err != nil {
		return nil, err
	}

	generalsFilename := filePrefix + ".GEN"
	generals, err := ReadGenerals(fsys, generalsFilename)
	if err != nil {
		return nil, err
	}

	terrainFilename := filePrefix + ".TER"
	terrain, err := ReadTerrain(fsys, terrainFilename, game)
	if err != nil {
		return nil, err
	}

	scenarioDataFilename := filePrefix + ".DTA"
	dta, err := ReadData(fsys, scenarioDataFilename)
	if err != nil {
		return nil, err
	}

	unitsFilename := filePrefix + ".UNI"
	units, err := ReadUnits(fsys, unitsFilename, game, dta.UnitTypes, dta.UnitNames, generals)
	if err != nil {
		return nil, err
	}

	scenarioData := &ScenarioData{
		Variants: variants,
		Generals: generals,
		Terrain:  terrain,
		Data:     dta,
		Units:    units}
	return scenarioData, nil
}
