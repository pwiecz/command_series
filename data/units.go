package data

import "bytes"
import "fmt"
import "io"
import "os"

type OrderType int

func (o OrderType) String() string {
	switch o {
	case Reserve:
		return "RESERVE"
	case Defend:
		return "DEFEND"
	case Attack:
		return "ATTACK"
	case Move:
		return "MOVE"
	default:
		return fmt.Sprintf("OrderType(%d)", int(o))
	}
}

const (
	Reserve OrderType = 0
	Defend  OrderType = 1
	Attack  OrderType = 2
	Move    OrderType = 3
)

type Unit struct {
	Side                 int  // 0 or 1
	InContactWithEnemy   bool // &1 != 0
	IsUnderAttack        bool // &2 != 0
	State2               bool // &4 != 0
	HasSupplyLine        bool // &8 == 0
	State4               bool // &16 != 0
	HasLocalCommand      bool // &32 != 0
	SeenByEnemy          bool // &64 != 0
	IsInGame             bool // &128 != 0
	X, Y                 int
	MenCount, EquipCount int
	Formation            int
	SupplyUnit           int // Index of this unit's supply unit
	FormationTopBit      bool
	Type                 int
	ColorPalette         int
	Name                 string
	TargetFormation      int
	OrderBit4            bool
	Order                OrderType
	GeneralIndex         int
	General              General
	SupplyLevel          int
	Morale               int
	Terrain              byte

	VariantBitmap        byte
	HalfDaysUntilAppear  int
	InvAppearProbability int

	Fatigue                int
	ObjectiveX, ObjectiveY int

	Index int
}

func (u *Unit) ClearState() {
	u.InContactWithEnemy = false
	u.IsUnderAttack = false
	u.State2 = false
	u.HasSupplyLine = true
	u.State4 = false
	u.HasLocalCommand = false
	u.SeenByEnemy = false
	u.IsInGame = false

}

type FlashbackUnit struct {
	X, Y         int
	ColorPalette int
	Type         int
}

func ReadUnits(filename string, game Game, unitNames [2][]string, generals [2][]General) ([2][]Unit, error) {
	file, err := os.Open(filename)
	if err != nil {
		return [2][]Unit{}, fmt.Errorf("Cannot open units file %s, %v", filename, err)
	}
	defer file.Close()
	var reader io.Reader
	if game == Conflict {
		decoded, err := UnpackFile(file)
		if err != nil {
			return [2][]Unit{}, err
		}
		reader = bytes.NewReader(decoded)
	} else {
		// Skip first two bytes of the file (they are all zeroes).
		var header [2]byte
		if _, err := io.ReadFull(file, header[:]); err != nil {
			return [2][]Unit{}, err
		}
		reader = file
	}
	units, err := ParseUnits(reader, unitNames, generals)
	if err != nil {
		return [2][]Unit{}, fmt.Errorf("Cannot parse units file %s, %v", filename, err)
	}
	return units, nil
}

func ParseUnit(data [16]byte, unitNames []string, generals []General) (Unit, error) {
	var unit Unit
	state := data[0]
	unit.InContactWithEnemy = state&1 != 0
	unit.IsUnderAttack = state&2 != 0
	unit.State2 = state&4 != 0
	unit.HasSupplyLine = state&8 == 0
	unit.State4 = state&16 != 0
	unit.HasLocalCommand = state&32 != 0
	unit.SeenByEnemy = state&64 != 0
	unit.IsInGame = state&128 != 0
	unit.X = int(data[1])
	unit.Y = int(data[2])
	unit.MenCount = int(data[3])
	unit.EquipCount = int(data[4])
	unit.Formation = int(data[5] & 7) // formation's bit 4 seems unused
	unit.SupplyUnit = int((data[5] / 16) & 7)
	unit.FormationTopBit = data[5]&128 != 0
	unit.VariantBitmap = data[6]
	unit.Type = int(data[7] & 15)
	unit.ColorPalette = int(data[7] / 16)
	nameIndex := int(data[8] & 127)
	// E.g. one Sidi unit have name index equal to the number of names.
	// It's a supply depot outside of map bounds, so maybe it's done on purpose.
	if nameIndex < len(unitNames) {
		unit.Name = unitNames[nameIndex]
	}

	unit.TargetFormation = int(data[9] & 7)
	unit.OrderBit4 = data[9]&8 != 0
	order := data[9] & 48
	switch order {
	case 0:
		unit.Order = Reserve
	case 16:
		unit.Order = Defend
	case 32:
		unit.Order = Attack
	default:
		unit.Order = Move
	}
	if order&0b11000000 != 0 {
		panic(order)
	}
	generalIndex := int(data[10])
	if generalIndex >= len(generals) {
		return Unit{}, fmt.Errorf("Too large general index. Expected <%d, got %d, %d %v %v", len(generals), generalIndex, state, unit, data)
	}
	unit.GeneralIndex = generalIndex
	unit.General = generals[generalIndex]
	unit.HalfDaysUntilAppear = int(data[11])
	unit.InvAppearProbability = int(data[12])
	unit.SupplyLevel = int(data[14])
	unit.Morale = int(data[15])
	return unit, nil
}

func ParseUnits(data io.Reader, unitNames [2][]string, generals [2][]General) ([2][]Unit, error) {
	var units [2][]Unit
	for i := 0; i < 128; i++ {
		var unitData [16]byte
		numRead, err := io.ReadFull(data, unitData[:])
		if numRead < 16 {
			if i != 127 || numRead != 15 {
				return [2][]Unit{}, fmt.Errorf("Too short unit %d, %d", i, numRead)
			}
			unitData[15] = 100
		}
		side := i / 64
		unit, err := ParseUnit(unitData, unitNames[side], generals[side])
		if err != nil {
			return [2][]Unit{}, fmt.Errorf("Error parsing unit %d, %v", i, err)
		}
		unit.Side = side
		unit.Index = len(units[i/64])
		units[i/64] = append(units[i/64], unit)
		if numRead < 16 {
			break
		}
	}
	return units, nil
}
