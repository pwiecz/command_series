package lib

import "io"
import "io/ioutil"
import "fmt"

type decoder struct {
	buf    []byte
	offset int
}

func NewDecoder(buf []byte) decoder {
	return decoder{
		buf:    buf,
		offset: 0,
	}
}

func (d *decoder) skipByteAndZero() {
	// First skipped byte after an IF statement can be anything, only the seconds one
	// is being checked for being zero.
	d.offset++
	if d.offset >= len(d.buf) {
		panic("EOF while skipping zeroes")
	}
	if d.buf[d.offset] != 0x0 {
		panic(fmt.Sprintf("Expected 0x0 got 0x%X at position %d", d.buf[d.offset], d.offset))
	}
	d.offset++
	if d.offset >= len(d.buf) {
		panic("EOF while skipping zeroes")
	}
}

func (d *decoder) noArgOpCode(opcode byte) Opcode {
	switch opcode {
	case 0x00:
		return Unknown{opcode}
	case 0x02:
		return Return{}
	case 0x04:
		return AdditiveInverse{}
	case 0x06:
		return Exit{}
	case 0x08, 0x0A:
		return Increment{}
	case 0x0C:
		return Decrement{}
	case 0x0E:
		return And0xFF{}
	case 0x10:
		return Rand{}
	case 0x12:
		return Abs{}
	case 0x14:
		return Sign{}
	case 0x16:
		return ReadByte{}
	case 0x18:
		return Exit{}
	case 0x1a:
		return Read{}
	case 0x1C:
		return Swap{}
	case 0x1E:
		return SignExtend{}
	case 0x22:
		return Dup{}
	case 0x40:
		return Add{}
	case 0x42:
		return Subtract{}
	case 0x44:
		return Multiply{}
	case 0x46:
		return Divide{}
	case 0x48:
		shiftOpcode := d.buf[d.offset]
		if shiftOpcode != 0x45 {
			panic("Unexpected opcode after MUL2")
		}
		d.offset++
		if d.offset >= len(d.buf) {
			panic(fmt.Errorf("EOF decoding opcode 0x48"))
		}
		arg := d.buf[d.offset]
		d.offset++
		if d.offset >= len(d.buf) {
			panic(fmt.Errorf("EOF decoding opcode 0x48"))
		}
		return MultiplyShiftRight{arg}
	case 0x4A:
		d.skipByteAndZero()
		return IfGreaterThanZero{}
	case 0x4C:
		d.skipByteAndZero()
		return IfZero{}
	case 0x4e:
		return Exit{}
	case 0x54:
		return BinaryAnd{}
	case 0x56:
		return Drop{}
	case 0x5C:
		return PopToD4{}
	case 0x60:
		return StoreByte{}
	case 0x68:
		return Store{}
	case 0x6A:
		return Clamp{}
	case 0x6C:
		return WriteToA200Plus{}
	case 0x6E:
		return FindObject{}
	case 0x70:
		d.skipByteAndZero()
		return IfNotEqual{}
	case 0x72:
		return CountNeighbourObjects{}
	case 0x74:
		return MagicNumber{}
	case 0xF6:
		d.skipByteAndZero()
		return Else{}
	case 0xF8:
		return FiAll{}
	case 0xFA:
		return Fi{}
	default:
		return Unknown{opcode}
	}
}
func (d *decoder) oneArgOpCode(opcode byte) Opcode {
	if d.offset >= len(d.buf) {
		panic(fmt.Errorf("EOF decoding opcode"))
	}
	arg := d.buf[d.offset]
	d.offset++
	switch opcode {
	case 0x80:
		return Done{arg}
	case 0x82:
		return ArithmeticShiftRight{arg}
	case 0x84:
		return ShiftLeft{arg}
	case 0x86:
		return ReadByteWithOffset{arg}
	case 0x88:
		return Gosub{arg}
	case 0x8A:
		panic(fmt.Sprintf("Unexpected opcode AFTER_SIGNED_MUL_SHIFT_RIGHT[%d]", arg))
		//return AfterSignedMulShiftRight{arg}
	case 0x90:
		return AndNum{arg}
	case 0x92:
		return OrNum{arg}
	case 0x94:
		return XorNum{arg}
	case 0x96:
		return GoTo{arg}
	case 0x98:
		return LogicalShiftRight{arg}
	case 0x9E:
		return Label{arg}
	case 0xA0:
		return PushSigned{int8(arg)}
	case 0xA2:
		if d.offset >= len(d.buf) {
			panic(fmt.Errorf("EOF decoding opcode 0xA2"))
		}
		arg2 := d.buf[d.offset]
		d.offset++
		return Push2Byte{256*uint16(arg2) + uint16(arg)}
	case 0xA4:
		return Push{arg}
	case 0xA6:
		return ScnDtaUnitTypeOffset{arg}
	case 0xC2:
		d.skipByteAndZero()
		return IfSignEq{arg}
	case 0xC4:
		return CoordsToMapAddress{arg}
	case 0xC8:
		return LoadUnit{arg}
	case 0xCA:
		return SaveUnit{arg}
	case 0xE0:
		return For{arg}
	case 0xE2:
		d.skipByteAndZero()
		return IfCmp{arg}
	case 0xE4:
		return IfNotBetweenSet{arg}
	case 0xE6:
		return Fill{arg}
	default:
		return UnknownOneArg{opcode, arg}
	}
}

func (d *decoder) Decode() (Opcode, bool) {
	if d.offset >= len(d.buf) {
		return nil, false
	}
	opcode := d.buf[d.offset]
	d.offset++
	if opcode > 0x7f {
		opcode <<= 1
		if opcode > 0x7f {
			opcode &= 0x7f
			return PopTo{opcode / 2}, true
		}
		return PushFrom{opcode / 2}, true
	}
	opcode <<= 1
	if opcode <= 0x7f || opcode > 0xF4 {
		return d.noArgOpCode(opcode), true
	}
	return d.oneArgOpCode(opcode), true
}

func ReadOpcodes(reader io.Reader) ([]Opcode, error) {
	buf, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	decoder := NewDecoder(buf)
	opcodes := []Opcode(nil)
	for {
		opcode, cont := decoder.Decode()
		if !cont {
			return opcodes, nil
		}
		opcodes = append(opcodes, opcode)
	}
}
