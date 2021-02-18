package lib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
)

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
	TypeName             string
	ColorPalette         int
	nameIndex            int
	Name                 string
	TargetFormation      int
	OrderBit4            bool
	Order                OrderType
	generalIndex         int
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
type Units [2][]Unit

func (u Unit) FullName() string {
	return u.Name + " " + u.TypeName
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
	Terrain      byte
}
type FlashbackUnits []FlashbackUnit
type FlashbackHistory []FlashbackUnits

func ReadUnits(fsys fs.FS, filename string, game Game, unitTypeNames []string, unitNames [2][]string, generals Generals) (Units, error) {
	fileData, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return Units{}, fmt.Errorf("Cannot read units file %s (%v)", filename, err)
	}
	var reader io.Reader
	if game == Conflict {
		decoded, err := UnpackFile(bytes.NewReader(fileData))
		if err != nil {
			return Units{}, err
		}
		reader = bytes.NewReader(decoded)
	} else {
		// Skip first two bytes of the file (they are all zeroes).
		reader = bytes.NewReader(fileData[2:])
	}
	units, err := ParseUnits(reader, unitTypeNames, unitNames, generals)
	if err != nil {
		return Units{}, fmt.Errorf("Cannot parse units file %s (%v)", filename, err)
	}
	return units, nil
}

func ParseUnit(data [16]byte, unitTypeNames []string, unitNames []string, generals []General) (Unit, error) {
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
	unit.Fatigue = int(data[6])
	unit.Type = int(data[7] & 15)
	if unit.Type >= len(unitTypeNames) {
		return Unit{}, fmt.Errorf("Invalid unit type number: %d", unit.Type)
	}
	unit.TypeName = unitTypeNames[unit.Type]
	unit.ColorPalette = int(data[7] / 16)
	unit.nameIndex = int(data[8] & 127)
	// E.g. one Sidi unit have name index equal to the number of names.
	// It's a supply depot outside of map bounds, so maybe it's done on purpose.
	if unit.nameIndex < len(unitNames) {
		unit.Name = unitNames[unit.nameIndex]
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
	unit.generalIndex = int(data[10])
	if unit.generalIndex >= len(generals) {
		// One of El-Alamein units have invalid general index set in available
		// disk images.
		fmt.Printf("Too large general index. Expected <%d, got %d\n", len(generals), unit.generalIndex)
		unit.generalIndex = 0
	}
	unit.General = generals[unit.generalIndex]
	if !unit.IsInGame {
		unit.HalfDaysUntilAppear = int(data[11])
		unit.InvAppearProbability = int(data[12])
	} else {
		unit.ObjectiveX = int(data[11])
		unit.ObjectiveY = int(data[12])
	}
	unit.Terrain = data[13]
	unit.SupplyLevel = int(data[14])
	unit.Morale = int(data[15])
	return unit, nil
}

func ParseUnits(data io.Reader, unitTypeNames []string, unitNames [2][]string, generals [2][]General) (Units, error) {
	var units Units
	for i := 0; i < 128; i++ {
		var unitData [16]byte
		numRead, err := io.ReadFull(data, unitData[:])
		if numRead < 16 {
			if i != 127 || numRead != 15 {
				return Units{}, fmt.Errorf("Too short unit %d data, %d bytes", i, numRead)
			}
			unitData[15] = 100
		}
		side := i / 64
		unit, err := ParseUnit(unitData, unitTypeNames, unitNames[side], generals[side])
		if err != nil {
			return Units{}, fmt.Errorf("Error parsing unit %d (%v)", i, err)
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

func (u *Units) Write(writer io.Writer) error {
	for _, sideUnits := range u {
		for _, unit := range sideUnits {
			if err := unit.Write(writer); err != nil {
				return err
			}
		}

		for i := len(sideUnits); i < 64; i++ {
			if _, err := writer.Write(make([]byte, 16)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (u *Unit) Write(writer io.Writer) error {
	var data [16]byte
	if u.InContactWithEnemy {
		data[0] |= 1
	}
	if u.IsUnderAttack {
		data[0] |= 2
	}
	if u.State2 {
		data[0] |= 4
	}
	if !u.HasSupplyLine {
		data[0] |= 8
	}
	if u.State4 {
		data[0] |= 16
	}
	if u.HasLocalCommand {
		data[0] |= 32
	}
	if u.SeenByEnemy {
		data[0] |= 64
	}
	if u.IsInGame {
		data[0] |= 128
	}
	data[1] = byte(u.X)
	data[2] = byte(u.Y)
	data[3] = byte(u.MenCount)
	data[4] = byte(u.EquipCount)
	data[5] = byte(u.Formation) + byte(u.SupplyUnit<<4)
	if u.FormationTopBit {
		data[5] |= 128
	}
	data[6] = byte(u.Fatigue)
	data[7] = byte(u.Type) + byte(u.ColorPalette<<4)
	data[8] = byte(u.nameIndex)
	data[9] = byte(u.TargetFormation) + byte(u.Order<<4)
	if u.OrderBit4 {
		data[9] |= 8
	}
	data[10] = byte(u.generalIndex)
	if u.IsInGame {
		data[11] = byte(u.ObjectiveX)
		data[12] = byte(u.ObjectiveY)
	} else {
		data[11] = byte(u.HalfDaysUntilAppear)
		data[12] = byte(u.InvAppearProbability)
	}
	data[13] = u.Terrain
	data[14] = byte(u.SupplyLevel)
	data[15] = byte(u.Morale)
	if _, err := writer.Write(data[:]); err != nil {
		return err
	}
	return nil
}

func (u FlashbackUnits) Write(writer io.Writer) error {
	size := uint64(len(u))
	if err := binary.Write(writer, binary.LittleEndian, size); err != nil {
		return err
	}
	var data [4]byte
	for _, unit := range u {
		data[0] = byte(unit.X)
		data[1] = byte(unit.Y)
		data[2] = byte(unit.Type) + byte(unit.ColorPalette<<4)
		data[3] = unit.Terrain
		if _, err := writer.Write(data[:]); err != nil {
			return err
		}
	}
	return nil
}

func (u *FlashbackUnits) Read(reader io.Reader) error {
	var size uint64
	if err := binary.Read(reader, binary.LittleEndian, &size); err != nil {
		return err
	}
	units := make([]FlashbackUnit, 0, size)
	var data [4]byte
	for i := 0; i < int(size); i++ {
		if _, err := io.ReadFull(reader, data[:]); err != nil {
			return err
		}
		units = append(units, FlashbackUnit{
			X:            int(data[0]),
			Y:            int(data[1]),
			Type:         int(data[2] & 15),
			ColorPalette: int(data[2] / 16),
			Terrain:      data[3]})
	}
	*u = FlashbackUnits(units)
	return nil
}

func (h FlashbackHistory) Write(writer io.Writer) error {
	size := uint64(len(h))
	if err := binary.Write(writer, binary.LittleEndian, size); err != nil {
		return err
	}
	for _, units := range h {
		if err := units.Write(writer); err != nil {
			return err
		}
	}
	return nil
}

func (h *FlashbackHistory) Read(reader io.Reader) error {
	var size uint64
	if err := binary.Read(reader, binary.LittleEndian, &size); err != nil {
		return err
	}
	history := make([]FlashbackUnits, 0, size)
	for i := 0; i < int(size); i++ {
		units := FlashbackUnits{}
		if err := units.Read(reader); err != nil {
			return err
		}
		history = append(history, units)
	}
	*h = FlashbackHistory(history)
	return nil
}
