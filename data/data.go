package data

import "fmt"
import "io"
import "os"

// Representation of data parsed from {scenario}.DTA files.
type ScenarioData struct {
	Data   [512]byte
	Data0  [16]int // Data[0:16] per unit type
	Data16 [16]int // Data[16:32] per unit type
	Data32 [16]int // Data[32:48] per unit type
	// Score gained by destroying enemy unit of this type
	UnitScores [16]int // Data[48:64]
	// Various bits concerning unit types... not all clear yet
	UnitMask         [16]byte // Data[80:96] (per unit type)
	UnitUsesSupplies [16]bool // bits 3 of bytes Data[80:96]
	UnitCanMove      [16]bool // bits 6 of bytes Data[80:96]
	Data96           [8]int   // Data[96:104] per terrain type
	Data104          [8]int   // Data[104:112] per terrain type
	Data112          [8]int   // Data[112:120] sth per terrain type
	Data120          [8]int   // Data[120:128] per terrain type
	Data128          [8]int   // Data[128:136] per formation&7
	Data136          [8]int   // Data[136:144] per formation&7
	Data144          [8]int   // Data[144:152] sth per formation&7
	Data152          [8]int   // Data[152:160] sth per formation&7
	// Units with type >=MinSupplyType can provide supply to other units.
	// Such units can receive supplies only from units with larger type numbers.
	MinSupplyType          int // Data[160]
	MaxResupplyAmount      int // Data[164]
	MaxSupplyTransportCost int // Data[165]
	// On average that many supplies will be used by each unit every day.
	ProbabilityOfUnitsUsingSupplies int        // Data[166]
	Data167                         int        // Data[167]
	MinutesPerTick                  int        // Data[168]
	UnitUpdatesPerTimeIncrement     int        // Data[169]
	Data173                         int        // Data[173] (a fatigue increase)
	Data178                         int        // Resusing value from array below (some kind of default formation?)
	Data176                         [16]int    // Data[176:190] four bytes per order (numbers 0-5)
	Data192                         [8]int     // Data[192:200] per formation
	UnitResupplyPerType             [16]int    // Data[200:216] top four bytes div 2
	ResupplyRate                    [2]int     // Data[232,233]
	MenReplacementRate              [2]int     // Data[234,235]
	EquipReplacementRate            [2]int     // Data[236,237]
	Data252                         [2]int     // Data[252:254] per side
	MoveCostPerTerrainTypesAndUnit  [8][16]int // Data[255:383]
	// Every chunk of four bytes list possible weather for a year's quarter.
	PossibleWeather [16]byte       // Data[384:400]
	DaytimePalette  [8]byte        // Data[400:408]
	NightPalette    [8]byte        // Data[408:416]
	MenCountLimit   [16]int        // Data[416:432]
	EquipCountLimit [16]int        // Data[432:448]
	DataUpdates     [21]DataUpdate //Data[448:511]
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

// At Day change byte at Offset of the scenario data to Value.
type DataUpdate struct {
	Day    int
	Offset int
	Value  int
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
	for i, v := range scenario.Data[0:16] {
		scenario.Data0[i] = int(v)
	}
	for i, v := range scenario.Data[16:32] {
		scenario.Data16[i] = int(v)
	}
	for i, v := range scenario.Data[32:48] {
		scenario.Data32[i] = int(v)
	}
	for i, v := range scenario.Data[48:64] {
		scenario.UnitScores[i] = int(v)
	}
	copy(scenario.UnitMask[:], scenario.Data[80:])
	for i, v := range scenario.UnitMask {
		scenario.UnitUsesSupplies[i] = v&8 == 0
		scenario.UnitCanMove[i] = v&64 == 0
	}
	for i, v := range scenario.Data[96:104] {
		scenario.Data96[i] = int(v)
	}
	for i, v := range scenario.Data[104:112] {
		scenario.Data104[i] = int(v)
	}
	for i, v := range scenario.Data[112:120] {
		scenario.Data112[i] = int(v)
	}
	for i, v := range scenario.Data[120:128] {
		scenario.Data120[i] = int(v)
	}
	for i, v := range scenario.Data[128:136] {
		scenario.Data128[i] = int(v)
	}
	for i, v := range scenario.Data[136:144] {
		scenario.Data136[i] = int(v)
	}
	for i, v := range scenario.Data[144:152] {
		scenario.Data144[i] = int(v)
	}
	for i, v := range scenario.Data[152:160] {
		scenario.Data152[i] = int(v)
	}
	scenario.MinSupplyType = int(scenario.Data[160])
	scenario.MaxResupplyAmount = int(scenario.Data[164])
	scenario.MaxSupplyTransportCost = int(scenario.Data[165])
	scenario.ProbabilityOfUnitsUsingSupplies = int(scenario.Data[166])
	scenario.Data167 = int(scenario.Data[167])
	scenario.MinutesPerTick = int(scenario.Data[168])
	scenario.UnitUpdatesPerTimeIncrement = int(scenario.Data[169])
	scenario.Data173 = int(scenario.Data[173])
	for i, v := range scenario.Data[176:190] {
		scenario.Data176[i] = int(v)
	}
	scenario.Data178 = int(scenario.Data[178])
	for i, v := range scenario.Data[192:200] {
		scenario.Data192[i] = int(v)
	}
	for i, resupply := range scenario.Data[200:216] {
		scenario.UnitResupplyPerType[i] = (int((resupply & 240) >> 1))
	}
	scenario.ResupplyRate[0] = int(scenario.Data[232]) * 2
	scenario.ResupplyRate[1] = int(scenario.Data[233]) * 2
	scenario.MenReplacementRate[0] = int(scenario.Data[234])
	scenario.MenReplacementRate[1] = int(scenario.Data[235])
	scenario.EquipReplacementRate[0] = int(scenario.Data[236])
	scenario.EquipReplacementRate[1] = int(scenario.Data[237])
	scenario.Data252[0] = int(scenario.Data[252])
	scenario.Data252[1] = int(scenario.Data[253])
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
	for i := 0; i < 21; i++ {
		scenario.DataUpdates[i].Day = int(scenario.Data[448+i*3])
		scenario.DataUpdates[i].Offset = int(scenario.Data[448+1+i*3])
		scenario.DataUpdates[i].Value = int(scenario.Data[448+2+i*3])
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
