package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
)

// Representation of data parsed from {scenario}.DTA files.
type Data struct {
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
	MenMultiplier                  int        // Data[170] (one man stored in unit data correspond to that many actual men)
	TanksMultiplier                int        // Data[171] (same as above but for tanks)
	Data173                        int        // Data[173] (a fatigue increase)
	Data174                        int        // Data[174]
	Data175                        int        // Data[175]
	Data176                        [4][4]int  // Data[176:190] four bytes per order (numbers 0-5)
	Data192                        [8]int     // Data[192:200] move cost per formation
	Data200Low                     [16]int    // Data[200:216] lower three bits per type
	UnitResupplyPerType            [16]int    // Data[200:216] top four bits div 2
	FormationChangeSpeed           [2][8]int  // Data[216:232]
	ResupplyRate                   [2]int     // Data[232,233]
	MenReplacementRate             [2]int     // Data[234,235]
	TankReplacementRate            [2]int     // Data[236,237]
	SideColor                      [2]int     // Data[248,249] the value*16 is the hue corresponding to the given side
	Data252                        [2]int     // Data[252:254] per side
	MoveSpeedPerTerrainTypeAndUnit [8][16]int // Data[255:383]
	// Every chunk of four bytes list possible weather for a year's quarter.
	PossibleWeather [16]byte       // Data[384:400]
	DaytimePalette  [8]byte        // Data[400:408]
	NightPalette    [8]byte        // Data[408:416]
	MenCountLimit   [16]int        // Data[416:432]
	TankCountLimit  [16]int        // Data[432:448]
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

func (d *Data) function10(order OrderType, offset int) int {
	if !InRange(offset, 0, 4) {
		panic(offset)
	}
	return d.Data176[int(order)][offset]
}

// At Day change byte at Offset of the scenario data to Value.
type DataUpdate struct {
	Day    int
	Offset int
	Value  byte
}

// ReadData reads and parses given {scenario}.DTA.
func ReadData(fsys fs.FS, filename string) (*Data, error) {
	fileData, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read data file %s (%v)", filename, err)
	}
	return ParseData(fileData)
}

// ParseData parses data from a {scenario.DTA file.
func ParseData(data []byte) (*Data, error) {
	if len(data) < 512 {
		return nil, fmt.Errorf("unexpected data file expecting >512, got %d", len(data))
	}
	scenario := &Data{}
	for i, value := range data[0:383] {
		scenario.UpdateData(i, value)
	}
	copy(scenario.PossibleWeather[:], data[384:])
	copy(scenario.DaytimePalette[:], data[400:])
	copy(scenario.NightPalette[:], data[408:])
	for i, limit := range data[416:432] {
		scenario.MenCountLimit[i] = int(limit)
	}
	for i, limit := range data[432:448] {
		scenario.TankCountLimit[i] = int(limit)
	}
	for i := 0; i < 21; i++ {
		scenario.DataUpdates[i].Day = int(data[448+i*3])
		scenario.DataUpdates[i].Offset = int(data[448+1+i*3])
		scenario.DataUpdates[i].Value = data[448+2+i*3]
	}

	reader := bytes.NewReader(data[512:])
	// There are 32 header bytes, but only 14 string lists.
	// Also offsets count from the start of the header, so subtract the header size
	// (32 bytes)
	stringListOffsets := make([]int, 16)
	for i := 0; i < 16; i++ {
		var offset [2]byte
		if _, err := io.ReadFull(reader, offset[:]); err != nil {
			return scenario, err
		}
		stringListOffsets[i] = int(offset[0]) + 256*int(offset[1]) - 32
	}
	for i := 0; i < 14; i++ {
		if stringListOffsets[i+1] < stringListOffsets[i] {
			return scenario, fmt.Errorf("invalid scenario file. Non-monotonic string offsets num %d, %d (%d, %d)", i, i+1, stringListOffsets[i], stringListOffsets[i+1])
		}
		stringData := make([]byte, stringListOffsets[i+1]-stringListOffsets[i])
		if _, err := io.ReadFull(reader, stringData); err != nil {
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

func (s *Data) UpdateData(offset int, value byte) {
	if offset >= 383 {
		panic(fmt.Errorf("invalid offset %d", offset))
	}
	switch {
	case InRange(offset, 0, 16):
		s.Data0Low[offset] = int(int8(value*16)) / 16
		s.Data0High[offset] = int(int8(value&240)) / 16
	case InRange(offset, 16, 32):
		s.Data16Low[offset-16] = int(value & 15)
		s.Data16High[offset-16] = int(value / 16)
	case InRange(offset, 32, 48):
		s.Data32[offset-32] = int(value)
	case InRange(offset, 48, 64):
		s.UnitScores[offset-48] = int(value)
	case InRange(offset, 64, 80):
		s.RecoveryRate[offset-64] = int(value)
	case InRange(offset, 80, 96):
		s.UnitMask[offset-80] = value
		s.UnitUsesSupplies[offset-80] = value&8 == 0
		s.UnitCanMove[offset-80] = value&64 == 0
	case InRange(offset, 96, 104):
		s.TerrainMenAttack[offset-96] = int(value)
	case InRange(offset, 104, 112):
		s.TerrainTankAttack[offset-104] = int(value)
	case InRange(offset, 112, 120):
		s.TerrainMenDefence[offset-112] = int(value)
	case InRange(offset, 120, 128):
		s.TerrainTankDefence[offset-120] = int(value)
	case InRange(offset, 128, 136):
		s.FormationMenAttack[offset-128] = int(value)
	case InRange(offset, 136, 144):
		s.FormationTankAttack[offset-136] = int(value)
	case InRange(offset, 144, 152):
		s.FormationMenDefence[offset-144] = int(value)
	case InRange(offset, 152, 160):
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
	case InRange(offset, 176, 190):
		s.Data176[(offset-176)/4][(offset-176)%4] = int(value)
	case InRange(offset, 192, 200):
		s.Data192[offset-192] = int(value)
	case InRange(offset, 200, 216):
		s.Data200Low[offset-200] = int(value & 7)
		s.UnitResupplyPerType[offset-200] = int((value & 240) >> 1)
	case InRange(offset, 216, 232):
		s.FormationChangeSpeed[(offset-216)/8][(offset-216)%8] = int(value)
	case offset == 232:
		s.ResupplyRate[0] = int(value)
	case offset == 233:
		s.ResupplyRate[1] = int(value)
	case offset == 234:
		s.MenReplacementRate[0] = int(value)
	case offset == 235:
		s.MenReplacementRate[1] = int(value)
	case offset == 236:
		s.TankReplacementRate[0] = int(value)
	case offset == 237:
		s.TankReplacementRate[1] = int(value)
	case InRange(offset, 248, 250):
		s.SideColor[offset-248] = int(value)
	case offset == 252:
		s.Data252[0] = int(value)
	case offset == 253:
		s.Data252[1] = int(value)
	case offset >= 255:
		s.MoveSpeedPerTerrainTypeAndUnit[(offset-255)/16][(offset-255)%16] = int(value)
	default:
	}
}

func (d *Data) ReadFirst255Bytes(reader io.Reader) error {
	var data [255]byte
	if _, err := io.ReadFull(reader, data[:]); err != nil {
		return err
	}
	for i, v := range data {
		d.UpdateData(i, v)
	}
	return nil
}

func (d *Data) WriteFirst255Bytes(writer io.Writer) error {
	var data [255]byte
	for i := 0; i < 16; i++ {
		data[i] = byte(d.Data0Low[i])&15 + (byte(d.Data0High[i]) << 4)
		data[16+i] = byte(d.Data16Low[i])&15 + (byte(d.Data16High[i]) << 4)
		data[32+i] = byte(d.Data32[i])
		data[48+i] = byte(d.UnitScores[i])
		data[64+i] = byte(d.RecoveryRate[i])
		data[80+i] = byte(d.UnitMask[i])
		data[200+i] = byte(d.Data200Low[i] + d.UnitResupplyPerType[i]*2)
	}
	for i := 0; i < 8; i++ {
		data[96+i] = byte(d.TerrainMenAttack[i])
		data[104+i] = byte(d.TerrainTankAttack[i])
		data[112+i] = byte(d.TerrainMenDefence[i])
		data[120+i] = byte(d.TerrainTankDefence[i])
		data[128+i] = byte(d.FormationMenAttack[i])
		data[136+i] = byte(d.FormationTankAttack[i])
		data[144+i] = byte(d.FormationMenDefence[i])
		data[152+i] = byte(d.FormationTankDefence[i])
		data[192+i] = byte(d.Data192[i])
	}
	data[160] = byte(d.MinSupplyType)
	data[161] = byte(d.HexSizeInMiles)
	data[162] = byte(d.Data162)
	data[163] = byte(d.Data163)
	data[164] = byte(d.MaxResupplyAmount)
	data[165] = byte(d.MaxSupplyTransportCost)
	data[166] = byte(d.AvgDailySupplyUse)
	data[167] = byte(d.Data167)
	data[168] = byte(d.MinutesPerTick)
	data[169] = byte(d.UnitUpdatesPerTimeIncrement)
	data[170] = byte(d.MenMultiplier)
	data[171] = byte(d.TanksMultiplier)
	data[173] = byte(d.Data173)
	data[174] = byte(d.Data174)
	data[175] = byte(d.Data175)
	for order := 0; order < 4; order++ {
		for i := 0; i < 4; i++ {
			data[176+order*4+i] = byte(d.Data176[order][i])
		}
	}
	for dir := 0; dir <= 1; dir++ {
		for formation := 0; formation < 8; formation++ {
			data[216+dir*8+formation] = byte(d.FormationChangeSpeed[dir][formation])
		}
	}
	for i := 0; i < 2; i++ {
		data[232+i] = byte(d.ResupplyRate[i])
		data[234+i] = byte(d.MenReplacementRate[i])
		data[236+i] = byte(d.TankReplacementRate[i])
		data[248+i] = byte(d.SideColor[i])
		data[252+i] = byte(d.Data252[i])
	}
	if _, err := writer.Write(data[:]); err != nil {
		return err
	}
	return nil
}
