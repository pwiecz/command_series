package lib

import (
	"fmt"

	"github.com/pwiecz/command_series/atr"
)

type GameData struct {
	Game      Game
	Scenarios []Scenario
	Sprites   Sprites
	Icons     Icons
	Map       Map
	Generic   Generic
	Hexes     Hexes
}
type ScenarioData struct {
	Variants []Variant
	Generals Generals
	Terrain  Terrain
	Data     Data
	Units    Units
}

func LoadGameData(diskImage atr.SectorReader) (*GameData, error) {
	game, err := DetectGame(diskImage)
	if err != nil {
		return nil, fmt.Errorf("Error detecting game, %v", err)
	}
	scenarios, err := ReadScenarios(diskImage)
	if err != nil {
		return nil, fmt.Errorf("Error loading scenarios, %v", err)
	}
	sprites, err := ReadSprites(diskImage)
	if err != nil {
		return nil, fmt.Errorf("Error loading sprites, %v", err)
	}
	icons, err := ReadIcons(diskImage)
	if err != nil {
		return nil, fmt.Errorf("Error loading icons, %v", err)
	}
	terrainMap, err := ReadMap(diskImage, game)
	if err != nil {
		return nil, fmt.Errorf("Error loading map, %v", err)
	}
	generic, err := ReadGeneric(diskImage)
	if err != nil {
		return nil, fmt.Errorf("Error loading generic, %v", err)
	}
	hexes, err := ReadHexes(diskImage)
	if err != nil {
		return nil, fmt.Errorf("Error loading hexes, %v", err)
	}
	gameData := &GameData{
		Game:      game,
		Scenarios: scenarios,
		Sprites:   sprites,
		Icons:     icons,
		Map:       terrainMap,
		Generic:   generic,
		Hexes:     hexes}
	return gameData, nil

}

func LoadScenarioData(diskImage atr.SectorReader, filePrefix string) (*ScenarioData, error) {
	game, err := FilePrefixToGame(filePrefix)
	if err != nil {
		return nil, err
	}
	variantsFilename := filePrefix + ".VAR"
	variants, err := ReadVariants(diskImage, variantsFilename)
	if err != nil {
		return nil, err
	}

	generalsFilename := filePrefix + ".GEN"
	generals, err := ReadGenerals(diskImage, generalsFilename)
	if err != nil {
		return nil, err
	}

	terrainFilename := filePrefix + ".TER"
	terrain, err := ReadTerrain(diskImage, terrainFilename, game)
	if err != nil {
		return nil, err
	}

	scenarioDataFilename := filePrefix + ".DTA"
	dta, err := ReadData(diskImage, scenarioDataFilename)
	if err != nil {
		return nil, err
	}

	unitsFilename := filePrefix + ".UNI"
	units, err := ReadUnits(diskImage, unitsFilename, game, dta.UnitTypes, dta.UnitNames, generals)
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
