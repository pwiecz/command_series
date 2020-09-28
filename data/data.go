package data

import "fmt"
import "io"
import "os"

// Representation of data parsed from {scenario}.DTA files.
type ScenarioData struct {
	Data [512]byte
	// Score gained by destroying enemy unit of this type
	UnitScores [16]int // Data[48:64]
	// Various bits concerning unit types... not all clear yet
	UnitMask         [16]byte // Data[80:96]
	UnitUsesSupplies [16]bool // bits 3 of bytes Data[80:96]
	UnitCanMove      [16]bool // bits 6 of bytes Data[80:96]
	// Units with type >=MinSupplyType can provide supply to other units.
	// Such units can receive supplies only from units with larger type numbers.
	MinSupplyType          int // Data[160]
	MaxResupplyAmount      int // Data[164]
	MaxSupplyTransportCost int // Data[165]
	Data112 [8]int // Data[112:118] sth per terrain type
	Data144 [8]int // Data[144:152] sth per formation
	// On average the many supplies will be used by each unit every day.
	ProbabilityOfUnitsUsingSupplies int        // Data[166]
	MinutesPerTick                  int        // Data[168]
	UnitUpdatesPerTimeIncrement     int        // Data[169]
	UnitResupplyPerType             [16]int    // Data[200:216] top four bytes div 2
	ResupplyRate                    [2]int     // Data[232,233]
	MenReplacementRate              [2]int     // Data[234,235]
	EquipReplacementRate            [2]int     // Data[236,237]
	MoveCostPerTerrainTypesAndUnit  [8][16]int // Data[255:263]
	// Every chunk of four bytes list possible weather for a year's quarter.
	PossibleWeather [16]byte // Data[384:400]
	DaytimePalette  [8]byte  // Data[400:408]
	NightPalette    [8]byte  // Data[408:416]
	MenCountLimit   [16]int  // Data[416:432]
	EquipCountLimit [16]int  // Data[432:448]
	UnitTypes       []string
	Strings1        []string
	Formations      []string
	Experience      []string
	Strings4        []string
	Equipments      []string
	UnitNames       [2][]string
	Strings7        []string
	Strings9        []string
	Months          []string
	Sides           []string
	Weather         []string
	Colors          []string
}

// ReadScenarioData reads and parses given {scenario}.DTA.
func ReadScenarioData(filename string) (ScenarioData, error) {
	var scenarioData ScenarioData
	file, err := os.Open(filename)
	if err != nil {
		return scenarioData, fmt.Errorf("Cannot open data file %s, %v", filename, err)
	}
	defer file.Close()
	return ParseScenarioData(file)
}

// ReadScenarioData parses data from a {scenario.DTA file.
func ParseScenarioData(data io.Reader) (ScenarioData, error) {
	var scenario ScenarioData
	_, err := io.ReadFull(data, scenario.Data[:])
	if err != nil {
		return scenario, err
	}
	for i, v := range scenario.Data[48:64] {
		scenario.UnitScores[i] = int(v)
	}
	copy(scenario.UnitMask[:], scenario.Data[80:])
	for i, v := range scenario.UnitMask {
		scenario.UnitUsesSupplies[i] = v&8 == 0
		scenario.UnitCanMove[i] = v&64 == 0
	}
	scenario.MinSupplyType = int(scenario.Data[160])
	scenario.MaxResupplyAmount = int(scenario.Data[164])
	scenario.MaxSupplyTransportCost = int(scenario.Data[165])
	scenario.ProbabilityOfUnitsUsingSupplies = int(scenario.Data[166])
	scenario.MinutesPerTick = int(scenario.Data[168])
	scenario.UnitUpdatesPerTimeIncrement = int(scenario.Data[169])
	for i, resupply := range scenario.Data[200:216] {
		scenario.UnitResupplyPerType[i] = (int((resupply & 240) >> 1))
	}
	scenario.ResupplyRate[0] = int(scenario.Data[232]) * 2
	scenario.ResupplyRate[1] = int(scenario.Data[233]) * 2
	scenario.MenReplacementRate[0] = int(scenario.Data[234])
	scenario.MenReplacementRate[1] = int(scenario.Data[235])
	scenario.EquipReplacementRate[0] = int(scenario.Data[236])
	scenario.EquipReplacementRate[1] = int(scenario.Data[237])
	for terrainType := 0; terrainType < 8; terrainType++ {
		for unitType, cost := range scenario.Data[255+16*terrainType : 255+16*(terrainType+1)] {
			scenario.MoveCostPerTerrainTypesAndUnit[terrainType][unitType] = int(cost)
		}
	}
	copy(scenario.PossibleWeather[:], scenario.Data[384:])
	copy(scenario.DaytimePalette[:], scenario.Data[400:])
	copy(scenario.NightPalette[:], scenario.Data[408:])
	for i, limit := range scenario.Data[416:432] {
		scenario.MenCountLimit[i] = int(limit)
	}
	for i, limit := range scenario.Data[432:448] {
		scenario.EquipCountLimit[i] = int(limit)
	}
	// There are 32 header bytes, but only 14 string lists.
	// Also offsets count from the start of the header, so subtract the header size
	// (32 bytes)
	stringListOffsets := make([]int, 16)
	for i := 0; i < 16; i++ {
		var offset [2]byte
		_, err = io.ReadFull(data, offset[:])
		if err != nil {
			return scenario, err
		}
		stringListOffsets[i] = int(offset[0]) + 256*int(offset[1]) - 32
	}
	for i := 0; i < 14; i++ {
		if stringListOffsets[i+1] < stringListOffsets[i] {
			return scenario, fmt.Errorf("Invalid scenario file. Non-monotonic string offsets num %d, %d (%d, %d)", i, i+1, stringListOffsets[i], stringListOffsets[i+1])
		}
		stringData := make([]byte, stringListOffsets[i+1]-stringListOffsets[i])
		_, err = io.ReadFull(data, stringData)
		if err != nil {
			return scenario, err
		}
		strings := []string{}
		for {
			byteString := []byte(nil)
			for j, b := range stringData {
				if b > 0x7f {
					byteString = stringData[0 : j+1]
					byteString[j] -= 0x80
					stringData = stringData[j+1:]
					break
				}
			}
			if byteString == nil {
				break
			}
			strings = append(strings, string(byteString))
		}
		switch i {
		case 0:
			scenario.UnitTypes = strings
		case 1:
			scenario.Strings1 = strings
		case 2:
			scenario.Formations = strings
		case 3:
			scenario.Experience = strings
		case 4:
			scenario.Strings4 = strings
		case 5:
			scenario.Equipments = strings
		case 6:
			scenario.UnitNames[0] = strings
		case 7:
			scenario.Strings7 = strings
		case 8:
			scenario.UnitNames[1] = strings
		case 9:
			scenario.Strings9 = strings
		case 10:
			scenario.Months = strings
		case 11:
			scenario.Sides = strings
		case 12:
			scenario.Weather = strings
		case 13:
			scenario.Colors = strings
		}
	}
	return scenario, nil
}