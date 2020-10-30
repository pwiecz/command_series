package data

import "fmt"
import "io"
import "os"

type OrderType int

const (
	Reserve OrderType = 0
	Defend  OrderType = 1
	Attack  OrderType = 2
	Move    OrderType = 3
)

type Unit struct {
	Side                 int  // 0 or 1
	State                byte // bit 15 - is added to game, bit 5 - local command, bit 4 - has contact with enemy, bit 3 - is there no supply line to unit, bit 1 - has contact with enemy?
	X, Y                 int
	MenCount, EquipCount int
	Formation            int
	SupplyUnit           int // Index of this unit's supply unit
	FormationTopBit bool
	Type                 int
	ColorPalette         int
	Name                 string
	OrderLower4Bits      byte
	Order                OrderType
	GeneralIndex         int
	General              General
	SupplyLevel          int
	Morale               int

	VariantBitmap        byte
	HalfDaysUntilAppear  int
	InvAppearProbability int

	Fatigue                int
	ObjectiveX, ObjectiveY int

	Index int
}

type FlashbackUnit struct {
	X, Y         int
	ColorPalette int
	Type         int
}

func ReadUnits(filename string, unitNames [2][]string, generals [2][]General) ([2][]Unit, error) {
	file, err := os.Open(filename)
	if err != nil {
		return [2][]Unit{}, fmt.Errorf("Cannot open units file %s, %v", filename, err)
	}
	defer file.Close()
	units, err := ParseUnits(file, unitNames, generals)
	if err != nil {
		return [2][]Unit{}, fmt.Errorf("Cannot parse units file %s, %v", filename, err)
	}
	return units, nil
}

func ParseUnit(data [16]byte, unitNames []string, generals []General) (Unit, error) {
	var unit Unit
	unit.State = data[0]
	unit.X = int(data[1])
	unit.Y = int(data[2])
	unit.MenCount = int(data[3])
	unit.EquipCount = int(data[4])
	unit.Formation = int(data[5] & 15)
	unit.SupplyUnit = int((data[5] / 16) & 7)
	unit.FormationTopBit = data[5] & 128 != 0
	unit.VariantBitmap = data[6]
	unit.Type = int(data[7] & 15)
	unit.ColorPalette = int(data[7] / 64)
	nameIndex := int(data[8] & 127)
	if nameIndex >= len(unitNames) {
		// there's problem with one Sidi units having high name index
		// return unit, fmt.Errorf("Too large unit name index. Expected <%d, got %d", len(unitNames), nameIndex)
		fmt.Printf("Too large unit name index. Expected <%d, got %d\n", len(unitNames), nameIndex)
		nameIndex = 0
	}
	unit.Name = unitNames[nameIndex]
	unit.OrderLower4Bits = data[9] & 15
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
		// there's problem with one El-Alamein and one Bulge unit having high general index
		//return units, fmt.Errorf("Error parsing unit %d, %v", i, err)
		fmt.Printf("Too large general index. Expected <%d, got %d\n", len(generals), generalIndex)
		generalIndex = 0
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
	var header [2]byte
	_, err := io.ReadFull(data, header[:2])
	if err != nil {
		return [2][]Unit{}, err
	}
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

/*func ParseUnitsConflict(data []byte, variantNum int, alliedUnitNames, germanUnitNames []string, generals Generals) (GameUnits, error) {
	var units GameUnits
	if len(data) < 128*16+2-1 {
		return units, fmt.Errorf("Too short units file, expected at least %d bytes, got %d",
			128*16+2, len(data))
	}
	data = data[2:]
	for i := 0; i < 128; i++ {
		var unitData [16]byte
		copy(unitData[:], data)
		if len(data) > 16 {
			data = data[16:]
		}
		if i == 127 {
			unitData[15] = 100
		}
		if unitData[3] == 0 || unitData[4] == 0 {
			continue
		}
		variantBitmap := unitData[6]
		if variantBitmap&(1<<variantNum) != 0 {
			continue
		}
		var unit InactiveUnit
		var err error
		if i < 64 {
			unit, err = ParseUnit(unitData, alliedUnitNames, generals.AlliedGenerals)
			if err != nil {
				return units, fmt.Errorf("Error parsing unit %d, %v", i, err)
			}
			unit.Side = Allied
		} else {
			unit, err = ParseUnit(unitData, germanUnitNames, generals.GermanGenerals)
			if err != nil {
				return units, fmt.Errorf("Error parsing unit %d, %v", i, err)
			}
			unit.Side = German
		}
		if unit.HalfDaysUntilAppear == 0 {
			activeUnit := ActiveUnit{
				unit: unit.unit,
			}
			units.ActiveUnits = append(units.ActiveUnits, activeUnit)
		} else {
			units.InactiveUnits = append(units.InactiveUnits, unit)
		}
	}
	return units, nil
}*/
