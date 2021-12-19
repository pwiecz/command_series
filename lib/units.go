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
	Side                int  // 0 or 1
	InContactWithEnemy  bool // &1 != 0
	IsUnderAttack       bool // &2 != 0
	State2              bool // &4 != 0
	HasSupplyLine       bool // &8 == 0
	State4              bool // &16 != 0
	HasLocalCommand     bool // &32 != 0
	SeenByEnemy         bool // &64 != 0
	IsInGame            bool // &128 != 0
	XY                  UnitCoords
	MenCount, TankCount int
	Formation           int
	SupplyUnit          int // Index of this unit's supply unit
	LongRangeAttack     bool
	Type                int
	TypeName            string
	ColorPalette        int
	nameIndex           int
	Name                string
	TargetFormation     int
	OrderBit4           bool
	Order               OrderType
	generalIndex        int
	General             General
	SupplyLevel         int
	Morale              int

	VariantBitmap        byte
	HalfDaysUntilAppear  int
	InvAppearProbability int

	Fatigue   int
	Objective UnitCoords

	Index int
}
type Units [2][]Unit

func (u Units) IsUnitAt(xy UnitCoords) bool {
	return u.IsUnitOfSideAt(xy, 0) || u.IsUnitOfSideAt(xy, 1)
}
func (u Units) IsUnitOfSideAt(xy UnitCoords, side int) bool {
	sideUnits := u[side]
	for i := range sideUnits {
		if sideUnits[i].IsInGame && sideUnits[i].XY == xy {
			return true
		}
	}
	return false
}
func (u Units) FindUnitAt(xy UnitCoords) (Unit, bool) {
	for _, sideUnits := range u {
		for i := range sideUnits {
			if sideUnits[i].IsInGame && sideUnits[i].XY == xy {
				return sideUnits[i], true
			}
		}
	}
	return Unit{}, false
}
func (u Units) FindUnitOfSideAt(xy UnitCoords, side int) (Unit, bool) {
	sideUnits := u[side]
	for i := range sideUnits {
		if sideUnits[i].IsInGame && sideUnits[i].XY == xy {
			return sideUnits[i], true
		}
	}
	return Unit{}, false
}
func (u Units) NeighbourUnitCount(xy UnitCoords, side int) int {
	num := 0
	for _, unit := range u[side] {
		if !unit.IsInGame {
			continue
		}
		if Abs(unit.XY.X-xy.X)+Abs(2*(unit.XY.Y-xy.Y)) < 4 {
			num++
		}
	}
	return num
}

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
func (u Unit) Function15_distanceToObjective() int {
	return hexDistance(u.Objective.X-u.XY.X, u.Objective.Y-u.XY.Y)
}
func (u Unit) IsVisible() bool {
	return u.IsInGame && (u.InContactWithEnemy || u.SeenByEnemy)
}

type FlashbackUnit struct {
	XY           UnitCoords
	ColorPalette int
	Type         int
}
type FlashbackUnits []FlashbackUnit
type FlashbackHistory []FlashbackUnits

func ReadUnits(fsys fs.FS, filename string, game Game, unitTypeNames []string, unitNames [2][]string, generals *Generals) (*Units, error) {
	fileData, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read units file %s (%v)", filename, err)
	}
	var reader io.Reader
	if game == Conflict {
		decoded, err := UnpackFile(bytes.NewReader(fileData))
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(decoded)
	} else {
		// Skip first two bytes of the file (they are all zeroes).
		reader = bytes.NewReader(fileData[2:])
	}
	units, err := ParseUnits(reader, unitTypeNames, unitNames, generals)
	if err != nil {
		return nil, fmt.Errorf("cannot parse units file %s (%v)", filename, err)
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
	unit.XY.X = int(data[1])
	unit.XY.Y = int(data[2])
	unit.MenCount = int(data[3])
	unit.TankCount = int(data[4])
	unit.Formation = int(data[5] & 7) // formation's bit 4 seems unused
	unit.SupplyUnit = int((data[5] / 16) & 7)
	unit.LongRangeAttack = data[5]&128 != 0
	unit.VariantBitmap = data[6]
	unit.Fatigue = int(data[6])
	unit.Type = int(data[7] & 15)
	if unit.Type >= len(unitTypeNames) {
		return unit, fmt.Errorf("invalid unit type number: %d", unit.Type)
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
		unit.Objective.X = int(data[11])
		unit.Objective.Y = int(data[12])
	}
	unit.SupplyLevel = int(data[14])
	unit.Morale = int(data[15])
	return unit, nil
}

func ParseUnits(data io.Reader, unitTypeNames []string, unitNames [2][]string, generals *Generals) (*Units, error) {
	var units Units
	for i := 0; i < 128; i++ {
		var unitData [16]byte
		numRead, _ := io.ReadFull(data, unitData[:])
		if numRead < 16 {
			if i != 127 || numRead != 15 {
				return nil, fmt.Errorf("too short unit %d data, %d bytes", i, numRead)
			}
			unitData[15] = 100
		}
		side := i / 64
		unit, err := ParseUnit(unitData, unitTypeNames, unitNames[side], generals[side])
		if err != nil {
			return nil, fmt.Errorf("error parsing unit %d (%v)", i, err)
		}
		unit.Side = side
		unit.Index = len(units[i/64])
		units[i/64] = append(units[i/64], unit)
		if numRead < 16 {
			break
		}
	}
	return &units, nil
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
	data[1] = byte(u.XY.X)
	data[2] = byte(u.XY.Y)
	data[3] = byte(u.MenCount)
	data[4] = byte(u.TankCount)
	data[5] = byte(u.Formation) + byte(u.SupplyUnit<<4)
	if u.LongRangeAttack {
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
		data[11] = byte(u.Objective.X)
		data[12] = byte(u.Objective.Y)
	} else {
		data[11] = byte(u.HalfDaysUntilAppear)
		data[12] = byte(u.InvAppearProbability)
	}
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
		data[0] = byte(unit.XY.X)
		data[1] = byte(unit.XY.Y)
		data[2] = byte(unit.Type) + byte(unit.ColorPalette<<4)
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
			XY:           UnitCoords{int(data[0]), int(data[1])},
			Type:         int(data[2] & 15),
			ColorPalette: int(data[2] / 16)})
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

func (u Unit) FullString() string {
	return fmt.Sprintf(`Side: %d
In contact with enemy: %t
Is under attack: %t
State2: %t
Has supply line: %t
State4: %t
Has local command: %t
Seen by enemy: %t
Is in game: %t
X,Y: %v
Formation: %d
Supply unit: %d
Long range attack: %t
Type: %s (%d)
ColorPalette: %d
Name: %s (%d)
Target formation: %d
Order bit4: %t
Order: %v
General: %s (%d)
Supply level: %d
Morale: %d
Variants: %08b
Half-days until appear: %d
Inv appear probability: %d
Fatigue: %d
ObjectiveX,ObjectiveY: %v`,
		u.Side, u.InContactWithEnemy, u.IsUnderAttack, u.State2, u.HasSupplyLine, u.State4, u.HasLocalCommand, u.SeenByEnemy, u.IsInGame, u.XY, u.Formation, u.SupplyUnit, u.LongRangeAttack, u.TypeName, u.Type, u.ColorPalette, u.Name, u.nameIndex, u.TargetFormation, u.OrderBit4, u.Order, u.General.Name, u.generalIndex, u.SupplyLevel, u.Morale, u.VariantBitmap, u.HalfDaysUntilAppear, u.InvAppearProbability, u.Fatigue, u.Objective)
}
