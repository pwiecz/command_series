package lib

import "fmt"

type Opcode interface {
	String() string
	StackEffect() (int, int)
	HasSideEffects() bool
}

type Byte struct {
	b byte
}

func (b Byte) String() string          { return fmt.Sprintf("%d", int(b.b)) }
func (b Byte) StackEffect() (int, int) { return 0, 1 }
func (b Byte) HasSideEffects() bool    { return false }

type Atom struct {
	s string
}

func NewAtom(s string) *Atom {
	return &Atom{s: s}
}
func (a *Atom) String() string          { return a.s }
func (a *Atom) StackEffect() (int, int) { return 0, 1 }
func (a *Atom) HasSideEffects() bool    { return false }

type Return struct{}

func (r Return) String() string          { return "RETURN" }
func (r Return) StackEffect() (int, int) { return 0, 0 }
func (r Return) HasSideEffects() bool    { return true }

type Exit struct{}

func (e Exit) String() string          { return "EXIT" }
func (e Exit) StackEffect() (int, int) { return 0, 0 }
func (e Exit) HasSideEffects() bool    { return true }

type Add struct{}

func (a Add) String() string          { return "ADD" }
func (a Add) StackEffect() (int, int) { return 2, 1 }
func (a Add) HasSideEffects() bool    { return false }

type Increment struct{}

func (i Increment) String() string          { return "INC" }
func (i Increment) StackEffect() (int, int) { return 1, 1 }
func (i Increment) HasSideEffects() bool    { return false }

type Subtract struct{}

func (s Subtract) String() string          { return "SUB" }
func (s Subtract) StackEffect() (int, int) { return 2, 1 }
func (s Subtract) HasSideEffects() bool    { return false }

type AdditiveInverse struct{}

func (i AdditiveInverse) String() string          { return "0_MINUS" }
func (i AdditiveInverse) StackEffect() (int, int) { return 1, 1 }
func (i AdditiveInverse) HasSideEffects() bool    { return false }

type Decrement struct{}

func (d Decrement) String() string          { return "DEC" }
func (d Decrement) StackEffect() (int, int) { return 1, 1 }
func (d Decrement) HasSideEffects() bool    { return false }

type Multiply struct{}

func (m Multiply) String() string          { return "MUL" }
func (m Multiply) StackEffect() (int, int) { return 2, 1 }
func (m Multiply) HasSideEffects() bool    { return false }

type MultiplyShiftRight struct{ b byte }

func (m MultiplyShiftRight) String() string          { return fmt.Sprintf("MUL_SHIFT_RIGHT[%d]", m.b) }
func (m MultiplyShiftRight) StackEffect() (int, int) { return 2, 1 } // TODO: check it
func (m MultiplyShiftRight) HasSideEffects() bool    { return false }

type Divide struct{}

func (d Divide) String() string          { return "DIV" }
func (d Divide) StackEffect() (int, int) { return 2, 1 }
func (d Divide) HasSideEffects() bool    { return false }

type IfGreaterThenZero struct{}

func (i IfGreaterThenZero) String() string          { return "IF_GREATER_THAN_ZERO" }
func (i IfGreaterThenZero) StackEffect() (int, int) { return 1, 0 }
func (i IfGreaterThenZero) HasSideEffects() bool    { return true }

type IfZero struct{}

func (i IfZero) String() string          { return "IF_ZERO" }
func (i IfZero) StackEffect() (int, int) { return 1, 0 }
func (i IfZero) HasSideEffects() bool    { return true }

type And_0xFF struct{}

func (a And_0xFF) String() string          { return "AND_OxFF" }
func (a And_0xFF) StackEffect() (int, int) { return 1, 1 }
func (a And_0xFF) HasSideEffects() bool    { return false }

type BinaryAnd struct{}

func (a BinaryAnd) String() string          { return "BAND" }
func (a BinaryAnd) StackEffect() (int, int) { return 2, 1 }
func (a BinaryAnd) HasSideEffects() bool    { return false }

type BinaryOr struct{}

func (o BinaryOr) String() string          { return "BOR" }
func (o BinaryOr) StackEffect() (int, int) { return 2, 1 }
func (o BinaryOr) HasSideEffects() bool    { return false }

type BinaryXor struct{}

func (x BinaryXor) String() string          { return "BXOR" }
func (x BinaryXor) StackEffect() (int, int) { return 2, 1 }
func (x BinaryXor) HasSideEffects() bool    { return false }

type ShiftLeft struct {
	shift byte
}

func (s ShiftLeft) String() string          { return fmt.Sprintf("SHIFT_LEFT[%d]", int(s.shift)) }
func (s ShiftLeft) StackEffect() (int, int) { return 1, 1 }
func (s ShiftLeft) HasSideEffects() bool    { return false }

type ShiftRight struct {
	shift byte
}

func (s ShiftRight) String() string          { return fmt.Sprintf("SHIFT_RIGHT[%d]", int(s.shift)) }
func (s ShiftRight) StackEffect() (int, int) { return 1, 1 }
func (s ShiftRight) HasSideEffects() bool    { return false }

type ScnDtaUnitTypeOffset struct {
	offset byte
}

func (o ScnDtaUnitTypeOffset) String() string {
	return fmt.Sprintf("[&SCN_DTA+%d+UNIT.TYPE]", o.offset)
}
func (o ScnDtaUnitTypeOffset) StackEffect() (int, int) { return 0, 1 }
func (o ScnDtaUnitTypeOffset) HasSideEffects() bool    { return false }

type ReadByte struct{}

func (r ReadByte) String() string          { return "READ_BYTE" }
func (r ReadByte) StackEffect() (int, int) { return 1, 1 }
func (r ReadByte) HasSideEffects() bool    { return false }

type Read struct{}

func (r Read) String() string          { return "READ" }
func (r Read) StackEffect() (int, int) { return 1, 1 }
func (r Read) HasSideEffects() bool    { return false }

type ReadByteWithOffset struct {
	offset byte
}

func (r ReadByteWithOffset) String() string          { return fmt.Sprintf("READ_BYTE[%d]", int(r.offset)) }
func (r ReadByteWithOffset) StackEffect() (int, int) { return 1, 1 }
func (r ReadByteWithOffset) HasSideEffects() bool    { return false }

type MulRandShiftRight8 struct{}

func (m MulRandShiftRight8) String() string          { return "MUL_RAND_SHR8" }
func (m MulRandShiftRight8) StackEffect() (int, int) { return 1, 1 }
func (m MulRandShiftRight8) HasSideEffects() bool    { return false }

type Abs struct{}

func (a Abs) String() string          { return "ABS" }
func (a Abs) StackEffect() (int, int) { return 1, 1 }
func (a Abs) HasSideEffects() bool    { return false }

type Sign struct{}

func (s Sign) String() string          { return "SIGN" }
func (s Sign) StackEffect() (int, int) { return 1, 1 }
func (s Sign) HasSideEffects() bool    { return false }

type Swap struct{}

func (s Swap) String() string          { return "SWAP" }
func (s Swap) StackEffect() (int, int) { return 2, 2 }
func (s Swap) HasSideEffects() bool    { return true }

type Dup struct{}

func (d Dup) String() string          { return "DUP" }
func (d Dup) StackEffect() (int, int) { return 1, 2 }
func (d Dup) HasSideEffects() bool    { return false }

type Drop struct{}

func (d Drop) String() string          { return "DROP" }
func (d Drop) StackEffect() (int, int) { return 1, 0 }
func (d Drop) HasSideEffects() bool    { return false }

type SignExtend struct{}

func (e SignExtend) String() string          { return "SIGN_EXTEND" }
func (e SignExtend) StackEffect() (int, int) { return 1, 1 }
func (e SignExtend) HasSideEffects() bool    { return false }

type Clamp struct{}

func (c Clamp) String() string          { return "CLAMP" }
func (c Clamp) StackEffect() (int, int) { return 3, 1 }
func (c Clamp) HasSideEffects() bool    { return false }

type WriteToA200Plus struct{}

func (w WriteToA200Plus) String() string          { return "WRITE_TO_A200_PLUS" }
func (w WriteToA200Plus) StackEffect() (int, int) { return 2, 0 }
func (w WriteToA200Plus) HasSideEffects() bool    { return true }

type PopToD4 struct{}

func (p PopToD4) String() string          { return "POP_TO_D4" }
func (p PopToD4) StackEffect() (int, int) { return 1, 0 }
func (p PopToD4) HasSideEffects() bool    { return true }

type FetchByteAt struct{}

func (f FetchByteAt) String() string          { return "FETCH_BYTE_AT" }
func (f FetchByteAt) StackEffect() (int, int) { return 1, 1 }
func (f FetchByteAt) HasSideEffects() bool    { return false }

type FetchAt struct{}

func (f FetchAt) String() string          { return "FETCH_AT" }
func (f FetchAt) StackEffect() (int, int) { return 1, 1 }
func (f FetchAt) HasSideEffects() bool    { return false }

type FindObject struct{}

func (f FindObject) String() string          { return "FIND_OBJECT" }
func (f FindObject) StackEffect() (int, int) { return 3, 1 }
func (f FindObject) HasSideEffects() bool    { return false }

type IfNotEqual struct{}

func (i IfNotEqual) String() string          { return "IF_NOT_EQUAL" }
func (i IfNotEqual) StackEffect() (int, int) { return 1, 0 }
func (i IfNotEqual) HasSideEffects() bool    { return true }

type CountNeighbourObjects struct{}

func (c CountNeighbourObjects) String() string          { return "COUNT_NEIGHBOUR_OBJECTS" }
func (c CountNeighbourObjects) StackEffect() (int, int) { return 3, 1 }
func (c CountNeighbourObjects) HasSideEffects() bool    { return false }

type MagicNumber struct{}

func (m MagicNumber) String() string          { return "MAGIC_NUMBER" }
func (m MagicNumber) StackEffect() (int, int) { return 4, 1 }
func (m MagicNumber) HasSideEffects() bool    { return false }

type Else struct{}

func (e Else) String() string          { return "ELSE" }
func (e Else) StackEffect() (int, int) { return 0, 0 }
func (e Else) HasSideEffects() bool    { return true }

type FiAll struct{}

func (f FiAll) String() string          { return "FI_ALL" }
func (f FiAll) StackEffect() (int, int) { return 0, 0 }
func (f FiAll) HasSideEffects() bool    { return true }

type Fi struct{}

func (f Fi) String() string          { return "FI" }
func (f Fi) StackEffect() (int, int) { return 0, 0 }
func (f Fi) HasSideEffects() bool    { return true }

type Done struct{ b byte }

func (d Done) String() string          { return fmt.Sprintf("DONE[%d]", int(d.b)) }
func (d Done) StackEffect() (int, int) { return 0, 0 }
func (d Done) HasSideEffects() bool    { return true }

type Gosub struct{ b byte }

func (g Gosub) String() string          { return fmt.Sprintf("GOSUB L%d", int(g.b)) }
func (g Gosub) StackEffect() (int, int) { return 0, 0 }
func (g Gosub) HasSideEffects() bool    { return true }

/*type AfterSignedMulShiftRight struct{ b byte }

func (s AfterSignedMulShiftRight) String() string {
	return fmt.Sprintf("AFTER_SIGNED_MUL_SHIFT_RIGHT[%d]", s.b)
}
func (s AfterSignedMulShiftRight) StackEffect() (int, int) { return 1, 1 }*/

type AndNum struct{ b byte }

func (a AndNum) String() string          { return fmt.Sprintf("AND[%d]", a.b) }
func (a AndNum) StackEffect() (int, int) { return 1, 1 }
func (a AndNum) HasSideEffects() bool    { return false }

type OrNum struct{ b byte }

func (o OrNum) String() string          { return fmt.Sprintf("OR[%d]", o.b) }
func (o OrNum) StackEffect() (int, int) { return 1, 1 }
func (o OrNum) HasSideEffects() bool    { return false }

type XorNum struct{ b byte }

func (x XorNum) String() string          { return fmt.Sprintf("XOR[%d]", x.b) }
func (x XorNum) StackEffect() (int, int) { return 1, 1 }
func (x XorNum) HasSideEffects() bool    { return false }

type GoTo struct{ b byte }

func (g GoTo) String() string          { return fmt.Sprintf("GOTO %d", g.b) }
func (g GoTo) StackEffect() (int, int) { return 0, 0 }
func (g GoTo) HasSideEffects() bool    { return true }

type RotateRight struct{ b byte }

func (r RotateRight) String() string          { return fmt.Sprintf("ROTATE_RIGHT[%d]", r.b) }
func (r RotateRight) StackEffect() (int, int) { return 1, 1 }
func (r RotateRight) HasSideEffects() bool    { return false }

type Label struct{ b byte }

func (l Label) String() string          { return fmt.Sprintf("L%d:", l.b) }
func (l Label) StackEffect() (int, int) { return 0, 0 }
func (l Label) HasSideEffects() bool    { return true }

type PushSigned struct{ n int8 }

func (p PushSigned) String() string          { return fmt.Sprintf("PUSH_SIGNED[%d]", p.n) }
func (p PushSigned) StackEffect() (int, int) { return 0, 1 }
func (p PushSigned) HasSideEffects() bool    { return false }

type Push2Byte struct{ n uint16 }

func (p Push2Byte) String() string          { return fmt.Sprintf("PUSH[0x%X]", p.n) }
func (p Push2Byte) StackEffect() (int, int) { return 0, 1 }
func (p Push2Byte) HasSideEffects() bool    { return false }

type Push struct{ b byte }

func (p Push) String() string          { return fmt.Sprintf("PUSH[%d]", p.b) }
func (p Push) StackEffect() (int, int) { return 0, 1 }
func (p Push) HasSideEffects() bool    { return false }

type IfSignEq struct{ b byte }

func (i IfSignEq) String() string          { return fmt.Sprintf("IF_SIGN_EQ[%d]", i.b) }
func (i IfSignEq) StackEffect() (int, int) { return 0, 0 }
func (i IfSignEq) HasSideEffects() bool    { return true }

type CoordsToMapAddress struct{ b byte }

func (c CoordsToMapAddress) String() string          { return fmt.Sprintf("COORDS_TO_MAP_ADDRESS[%d]", c.b) }
func (c CoordsToMapAddress) StackEffect() (int, int) { return 2, 1 }
func (c CoordsToMapAddress) HasSideEffects() bool    { return false }

type LoadUnit struct{ b byte }

func (l LoadUnit) String() string {
	if l.b == 0 {
		return "LOAD_UNIT1"
	} else if l.b == 1 {
		return "LOAD_UNIT2"
	} else {
		return fmt.Sprintf("LOAD_UNIT[%d]", l.b)
	}
}
func (l LoadUnit) StackEffect() (int, int) { return 0, 1 }
func (l LoadUnit) HasSideEffects() bool    { return false }

type SaveUnit struct{ b byte }

func (s SaveUnit) String() string {
	if s.b == 0 {
		return "SAVE_UNIT1"
	} else if s.b == 1 {
		return "SAVE_UNIT2"
	} else {
		return fmt.Sprintf("SAVE_UNIT[%d]", s.b)
	}

}
func (s SaveUnit) StackEffect() (int, int) { return 0, 0 }
func (s SaveUnit) HasSideEffects() bool    { return true }

type For struct{ b byte }

func (f For) String() string          { return fmt.Sprintf("FOR[%d]", f.b) }
func (f For) StackEffect() (int, int) { return 2, 0 }
func (f For) HasSideEffects() bool    { return true }

type IfCmp struct{ b byte }

func (i IfCmp) String() string {
	if i.b == 255 {
		return "IF[<]"
	} else if i.b == 0 {
		return "IF[==]"
	} else if i.b == 1 {
		return "IF[>]"
	} else {
		return fmt.Sprintf("IF_CMP_IS[%d]", i.b)
	}
}
func (i IfCmp) StackEffect() (int, int) { return 0, 0 }
func (i IfCmp) HasSideEffects() bool    { return true }

type IfNotBetweenSet struct{ b byte }

func (s IfNotBetweenSet) String() string          { return fmt.Sprintf("IF_NO_BETWEEN_SET[%d]", s.b) }
func (s IfNotBetweenSet) StackEffect() (int, int) { return 3, 1 }
func (s IfNotBetweenSet) HasSideEffects() bool    { return false }

type Fill struct{ b byte }

func (f Fill) String() string          { return fmt.Sprintf("FILL[%d]", f.b) }
func (f Fill) StackEffect() (int, int) { return 2, 0 }
func (f Fill) HasSideEffects() bool    { return true }

type Unknown struct{ opcode byte }

func (u Unknown) String() string          { return fmt.Sprintf("0x%X", u.opcode) }
func (u Unknown) StackEffect() (int, int) { return 0, 0 }
func (u Unknown) HasSideEffects() bool    { return true }

type UnknownOneArg struct{ opcode, b byte }

func (u UnknownOneArg) String() string          { return fmt.Sprintf("0x%X[%d]", u.opcode, u.b) }
func (u UnknownOneArg) StackEffect() (int, int) { return 0, 0 }
func (u UnknownOneArg) HasSideEffects() bool    { return true }

type PopTo struct{ b byte }

func (p PopTo) String() string          { return fmt.Sprintf("POP_TO(%s)", numToUnitField(p.b)) }
func (p PopTo) StackEffect() (int, int) { return 1, 0 }
func (p PopTo) HasSideEffects() bool    { return true }

type PushFrom struct{ b byte }

func (p PushFrom) String() string {
	var locStr string
	switch p.b {
	case 1:
		locStr = "&SCN_UNI"
	case 2:
		locStr = "&CRUSADE_MAP"
	case 3:
		locStr = "&SCN_TER"
	case 4:
		locStr = "&GENERIC_DTA"
	case 5:
		locStr = "&SCN_DTA"
	case 17:
		locStr = "&SCN_DTA_STRINGS"
	case 24:
		locStr = "&SCN_TER_2"
	case 8:
		locStr = "&HEXES_DTA"
	case 22:
		locStr = "29927"
	case 31:
		locStr = "&V10[8]"
	case 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63:
		locStr = numToUnitField(p.b)
	default:
		locStr = fmt.Sprintf("V%d", p.b)
	}
	return fmt.Sprintf("PUSH_FROM(%s)", locStr)
}
func (p PushFrom) StackEffect() (int, int) { return 0, 1 }
func (p PushFrom) HasSideEffects() bool    { return false }

func numToUnitField(num byte) string {
	if num < 32 || num > 63 {
		return fmt.Sprintf("V%d", num)
	}
	unitName := "UNIT"
	if num >= 48 {
		unitName = "UNIT2"
	}
	unitField := fmt.Sprintf("%d", (num-32)%16)
	switch (num - 32) % 16 {
	case 0:
		unitField = "STATE"
	case 1:
		unitField = "X"
	case 2:
		unitField = "Y"
	case 3:
		unitField = "MEN"
	case 4:
		unitField = "TANKS"
	case 5:
		unitField = "_FORMATION"
	case 6:
		unitField = "FATIGUE"
	case 7:
		unitField = "SIDE_AND_TYPE"
	case 8:
		unitField = "NAME"
	case 9:
		unitField = "_ORDER"
	case 10:
		unitField = "GENERAL"
	case 11:
		unitField = "OBJ_X"
	case 12:
		unitField = "OBJ_Y"
	case 13:
		unitField = "TERRAIN"
	case 14:
		unitField = "SUPPLY"
	case 15:
		unitField = "MORALE"
	}
	return fmt.Sprintf("%s.%s", unitName, unitField)
}
