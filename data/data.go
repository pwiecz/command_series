package data

import "bytes"
import "fmt"
import "io"

import "github.com/pwiecz/command_series/atr"

// Representation of data parsed from {scenario}.DTA files.
type ScenarioData struct {
	Data       [512]byte
	Data0Low   [16]int // Data[0:16] per unit type (lower 4 bits)
	Data0High  [16]int // Data[0:16] per unit type (higher 4 bits)
	Data16Low  [16]int // Data[16:32] per unit type (lower 4 bits)
	Data16High [16]int // Data[16:32] per unit type (higher 4 bits)
	Data32     [16]int // Data[32:48] per unit type (&31 attack range)
	// Score gained by destroying enemy unit of this type
	// Units with score >= 4 are high importance units which are priority targets.
	UnitScores   [16]int // Data[48:64]
	RecoveryRate [16]int // Data[64:80]
	// Various bits concerning unit types... not all clear yet (&4 weather has no impact?)
	UnitMask             [16]byte // Data[80:96] (per unit type)
	UnitUsesSupplies     [16]bool // !bit 3(&8) of bytes Data[80:96]
	UnitCanMove          [16]bool // !bit 6(&64) of bytes Data[80:96]
	TerrainMenAttack     [8]int   // Data[96:104]
	TerrainTankAttack    [8]int   // Data[104:112]
	TerrainMenDefence    [8]int   // Data[112:120]
	TerrainTankDefence   [8]int   // Data[120:128]
	FormationMenAttack   [8]int   // Data[128:136]
	FormationTankAttack  [8]int   // Data[136:144]
	FormationMenDefence  [8]int   // Data[144:152]
	FormationTankDefence [8]int   // Data[152:160]
	// Units with type >=MinSupplyType can provide supply to other units.
	// Such units can receive supplies only from units with larger type numbers.
	MinSupplyType          int // Data[160]
	HexSizeInMiles         int // Data[161]
	Data162                int // Data[162] some generic supply use (while attacking?)
	Data163                int // Data[163] some generic supply use (while being attacked?)
	MaxResupplyAmount      int // Data[164]
	MaxSupplyTransportCost int // Data[165] in half-miles
	// On average that many supplies will be used by each unit every day.
	AvgDailySupplyUse              int        // Data[166]
	Data167                        int        // Data[167]
	MinutesPerTick                 int        // Data[168]
	UnitUpdatesPerTimeIncrement    int        // Data[169]
	MenMultiplier                  int        // Data[170] (one man store in unit data correspond to that many actual men)
	TanksMultiplier                int        // Data[171] (same as above but for tanks)
	Data173                        int        // Data[173] (a fatigue increase)
	Data174                        int        // Data[174]
	Data175                        int        // Data[175]
	Data176                        [4][4]int  // Data[176:190] four bytes per order (numbers 0-5)
	Data192                        [8]int     // Data[192:200] move cost per formation
	Data200Low                     [16]int    // Data[200:216] lower three bits per type
	UnitResupplyPerType            [16]int    // Data[200:216] top four bits div 2
	Data216                        [16]int    // Data[216:232]
	ResupplyRate                   [2]int     // Data[232,233]
	MenReplacementRate             [2]int     // Data[234,235]
	EquipReplacementRate           [2]int     // Data[236,237]
	SideColor                      [2]int     // Data[248,249] the value*16 is the hue corresponding to the given side
	Data252                        [2]int     // Data[252:254] per side
	MoveCostPerTerrainTypesAndUnit [8][16]int // Data[255:383]
	// Every chunk of four bytes list possible weather for a year's quarter.
	PossibleWeather [16]byte       // Data[384:400]
	DaytimePalette  [8]byte        // Data[400:408]
	NightPalette    [8]byte        // Data[408:416]
	MenCountLimit   [16]int        // Data[416:432]
	EquipCountLimit [16]int        // Data[432:448]
	DataUpdates     [21]DataUpdate // Data[448:511]
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
	Offset byte
	Value  byte
}

// ReadScenarioData reads and parses given {scenario}.DTA.
func ReadScenarioData(diskimage atr.SectorReader, filename string) (ScenarioData, error) {
	var scenarioData ScenarioData
	fileData, err := atr.ReadFile(diskimage, filename)
	if err != nil {
		return scenarioData, fmt.Errorf("Cannot read data file %s (%v)", filename, err)
	}
	return ParseScenarioData(bytes.NewReader(fileData))
}

// ReadScenarioData parses data from a {scenario.DTA file.
func ParseScenarioData(data io.Reader) (ScenarioData, error) {
	var scenario ScenarioData
	_, err := io.ReadFull(data, scenario.Data[:])
	if err != nil {
		return scenario, err
	}
	for i, v := range scenario.Data[0:16] {
		scenario.Data0Low[i] = int(int8(v*16)) / 16
		scenario.Data0High[i] = int(int8(v&240)) / 16
	}
	for i, v := range scenario.Data[16:32] {
		scenario.Data16Low[i] = int(v & 15)
		scenario.Data16High[i] = int(v / 16)
	}
	for i, v := range scenario.Data[32:48] {
		scenario.Data32[i] = int(v)
	}
	for i, v := range scenario.Data[48:64] {
		scenario.UnitScores[i] = int(v)
	}
	for i, v := range scenario.Data[64:80] {
		scenario.RecoveryRate[i] = int(v)
	}
	copy(scenario.UnitMask[:], scenario.Data[80:])
	for i, v := range scenario.UnitMask {
		scenario.UnitUsesSupplies[i] = v&8 == 0
		scenario.UnitCanMove[i] = v&64 == 0
	}
	for i, v := range scenario.Data[96:104] {
		scenario.TerrainMenAttack[i] = int(v)
	}
	for i, v := range scenario.Data[104:112] {
		scenario.TerrainTankAttack[i] = int(v)
	}
	for i, v := range scenario.Data[112:120] {
		scenario.TerrainMenDefence[i] = int(v)
	}
	for i, v := range scenario.Data[120:128] {
		scenario.TerrainTankDefence[i] = int(v)
	}
	for i, v := range scenario.Data[128:136] {
		scenario.FormationMenAttack[i] = int(v)
	}
	for i, v := range scenario.Data[136:144] {
		scenario.FormationTankAttack[i] = int(v)
	}
	for i, v := range scenario.Data[144:152] {
		scenario.FormationMenDefence[i] = int(v)
	}
	for i, v := range scenario.Data[152:160] {
		scenario.FormationTankDefence[i] = int(v)
	}
	scenario.MinSupplyType = int(scenario.Data[160])
	scenario.HexSizeInMiles = int(scenario.Data[161])
	scenario.Data162 = int(scenario.Data[162])
	scenario.Data163 = int(scenario.Data[163])
	scenario.MaxResupplyAmount = int(scenario.Data[164])
	scenario.MaxSupplyTransportCost = int(scenario.Data[165])
	scenario.AvgDailySupplyUse = int(scenario.Data[166])
	scenario.Data167 = int(scenario.Data[167])
	scenario.MinutesPerTick = int(scenario.Data[168])
	scenario.UnitUpdatesPerTimeIncrement = int(scenario.Data[169])
	scenario.MenMultiplier = int(scenario.Data[170])
	scenario.TanksMultiplier = int(scenario.Data[171])
	scenario.Data173 = int(scenario.Data[173])
	scenario.Data174 = int(scenario.Data[174])
	scenario.Data175 = int(scenario.Data[175])
	for i, v := range scenario.Data[176:190] {
		scenario.Data176[i/4][i%4] = int(v)
	}
	for i, v := range scenario.Data[192:200] {
		scenario.Data192[i] = int(v)
	}
	for i, v := range scenario.Data[200:216] {
		scenario.Data200Low[i] = int(v & 7)
		scenario.UnitResupplyPerType[i] = (int((v & 240) >> 1))
	}
	for i, v := range scenario.Data[216:232] {
		scenario.Data216[i] = int(v)
	}
	scenario.ResupplyRate[0] = int(scenario.Data[232])
	scenario.ResupplyRate[1] = int(scenario.Data[233])
	scenario.MenReplacementRate[0] = int(scenario.Data[234])
	scenario.MenReplacementRate[1] = int(scenario.Data[235])
	scenario.EquipReplacementRate[0] = int(scenario.Data[236])
	scenario.EquipReplacementRate[1] = int(scenario.Data[237])
	scenario.SideColor[0] = int(scenario.Data[248])
	scenario.SideColor[1] = int(scenario.Data[249])
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
		scenario.DataUpdates[i].Offset = scenario.Data[448+1+i*3]
		scenario.DataUpdates[i].Value = scenario.Data[448+2+i*3]
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

func inRange(v, min, max byte) bool {
	if v < min || v >= max {
		return false
	}
	return true
}

func (s *ScenarioData) UpdateData(offset, value byte) {
	switch {
	case inRange(offset, 0, 16):
		s.Data0Low[offset] = int(int8(value*16)) / 16
		s.Data0High[offset] = int(int8(value&240)) / 16
	case inRange(offset, 16, 32):
		s.Data16Low[offset-16] = int(value & 15)
		s.Data16High[offset-16] = int(value / 16)
	case inRange(offset, 32, 48):
		s.Data32[offset-32] = int(value)
	case inRange(offset, 48, 64):
		s.UnitScores[offset-48] = int(value)
	case inRange(offset, 64, 80):
		s.RecoveryRate[offset-64] = int(value)
	case inRange(offset, 80, 96):
		s.UnitMask[offset-80] = value
		s.UnitUsesSupplies[offset-80] = value&8 == 0
		s.UnitCanMove[offset-80] = value&64 == 0
	case inRange(offset, 96, 104):
		s.TerrainMenAttack[offset-96] = int(value)
	case inRange(offset, 104, 112):
		s.TerrainTankAttack[offset-104] = int(value)
	case inRange(offset, 112, 120):
		s.TerrainMenDefence[offset-112] = int(value)
	case inRange(offset, 120, 128):
		s.TerrainTankDefence[offset-120] = int(value)
	case inRange(offset, 128, 136):
		s.FormationMenAttack[offset-128] = int(value)
	case inRange(offset, 136, 144):
		s.FormationTankAttack[offset-136] = int(value)
	case inRange(offset, 144, 152):
		s.FormationMenDefence[offset-144] = int(value)
	case inRange(offset, 152, 160):
		s.FormationTankDefence[offset-152] = int(value)
	case offset == 160:
		s.MinSupplyType = int(value)
	case offset == 161:
		s.HexSizeInMiles = int(value)
	case offset == 162:
		s.Data162 = int(value)
	case offset == 163:
		s.Data163 = int(value)
	case offset == 164:
		s.MaxResupplyAmount = int(value)
	case offset == 165:
		s.MaxSupplyTransportCost = int(value)
	case offset == 166:
		s.AvgDailySupplyUse = int(value)
	case offset == 167:
		s.Data167 = int(value)
	case offset == 168:
		s.MinutesPerTick = int(value)
	case offset == 169:
		s.UnitUpdatesPerTimeIncrement = int(value)
	case offset == 170:
		s.MenMultiplier = int(value)
	case offset == 171:
		s.TanksMultiplier = int(value)
	case offset == 173:
		s.Data173 = int(value)
	case offset == 174:
		s.Data174 = int(value)
	case offset == 175:
		s.Data175 = int(value)
	case inRange(offset, 176, 190):
		s.Data176[(offset-176)/4][(offset-176)%4] = int(value)
	case inRange(offset, 192, 200):
		s.Data192[offset-192] = int(value)
	case inRange(offset, 200, 216):
		s.Data200Low[offset-200] = int(value & 7)
		s.UnitResupplyPerType[offset-200] = int((value & 240) >> 1)
	case inRange(offset, 216, 232):
		s.Data216[offset-216] = int(value)
	case offset == 232:
		s.ResupplyRate[0] = int(value)
	case offset == 233:
		s.ResupplyRate[1] = int(value)
	case offset == 234:
		s.MenReplacementRate[0] = int(value)
	case offset == 235:
		s.MenReplacementRate[1] = int(value)
	case offset == 236:
		s.EquipReplacementRate[0] = int(value)
	case offset == 237:
		s.EquipReplacementRate[1] = int(value)
	case offset == 252:
		s.Data252[0] = int(value)
	case offset == 253:
		s.Data252[1] = int(value)
	default:
		panic(fmt.Errorf("Unhandled update offset %d", int(offset)))
	}
}
