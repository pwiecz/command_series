package lib

import "fmt"
import "strings"

type Expression interface {
	String() string
}

type Atom struct {
	s string
}

func (a Atom) String() string { return a.s }

type scopeType int

const (
	IF  scopeType = 0
	FOR scopeType = 1
)

type FoldingDecoder struct {
	stack  []Expression
	scopes []scopeType
}

func (f *FoldingDecoder) top() Expression {
	return f.stack[len(f.stack)-1]
}
func (f *FoldingDecoder) belowTop() Expression {
	return f.stack[len(f.stack)-2]
}
func (f *FoldingDecoder) popN(n int) {
	f.stack = f.stack[:len(f.stack)-n]
}
func (f *FoldingDecoder) push(e Expression) {
	f.stack = append(f.stack, e)
}
func (f *FoldingDecoder) popNAndPush(n int, e Expression) {
	f.stack = append(f.stack[:len(f.stack)-n], e)
}

func (f *FoldingDecoder) popScope() {
	f.scopes = f.scopes[:len(f.scopes)-1]
}
func (f *FoldingDecoder) pushScope(t scopeType) {
	f.scopes = append(f.scopes, t)
}

func (f *FoldingDecoder) binaryOp(o Opcode, op string) bool {
	if len(f.stack) >= 2 {
		a := Atom{"(" + f.belowTop().String() + ")" + op + "(" + f.top().String() + ")"}
		f.popNAndPush(2, a)
		return true
	}
	return false
}
func (f *FoldingDecoder) funcCall(o Opcode, name string, numArgs int) bool {
	if len(f.stack) >= numArgs {
		args := make([]string, numArgs)
		for i, expr := range f.stack[len(f.stack)-numArgs:] {
			args[i] = expr.String()
		}
		a := Atom{fmt.Sprintf("%s(%s)", name, strings.Join(args, ","))}
		f.popNAndPush(numArgs, a)
		return true
	}
	return false
}

func (f *FoldingDecoder) Apply(o Opcode) {
	toPop, toPush := o.StackEffect()
	if !o.HasSideEffects() && toPop <= len(f.stack) {
		stackLen := len(f.stack)
		switch v := o.(type) {
		case Byte:
			f.push(Atom{o.String()})
		case Add:
			f.binaryOp(o, "+")
		case Subtract:
			f.binaryOp(o, "-")
		case Multiply:
			f.binaryOp(o, "*")
		case Divide:
			f.binaryOp(o, "/")
		case MultiplyShiftRight:
			a := Atom{fmt.Sprintf("(%s)*(%s)>>%d", f.belowTop(), f.top(), v.b)}
			f.popNAndPush(2, a)
		case Increment:
			a := Atom{"(" + f.top().String() + ")+1"}
			f.popNAndPush(1, a)
		case Decrement:
			a := Atom{"(" + f.top().String() + ")-1"}
			f.popNAndPush(1, a)
		case AdditiveInverse:
			a := Atom{"-(" + f.top().String() + ")"}
			f.popNAndPush(1, a)
		case And_0xFF:
			a := Atom{"(" + f.top().String() + ")&0xFF"}
			f.popNAndPush(1, a)
		case BinaryAnd:
			f.binaryOp(o, "&")
		case BinaryOr:
			f.binaryOp(o, "|")
		case BinaryXor:
			f.binaryOp(o, "^")
		case ShiftLeft:
			a := Atom{fmt.Sprintf("(%s)<<%d", f.top(), v.shift)}
			f.popNAndPush(1, a)
		case ShiftRight:
			a := Atom{fmt.Sprintf("(%s)>>%d", f.top(), v.shift)}
			f.popNAndPush(1, a)
		case ScnDtaUnitTypeOffset:
			a := Atom{fmt.Sprintf("[&SCN_DTA+%d+UNIT.TYPE]", v.offset)}
			f.push(a)
		case ReadByte:
			a := Atom{fmt.Sprintf("[%s]", f.top())}
			f.popNAndPush(1, a)
		case Read:
			a := Atom{fmt.Sprintf("[%s:]", f.top())}
			f.popNAndPush(1, a)
		case ReadByteWithOffset:
			a := Atom{fmt.Sprintf("[(%s)+%d]", f.top(), v.offset)}
			f.popNAndPush(1, a)
		case MulRandShiftRight8:
			f.funcCall(o, "MUL_RAND_SHR8", 1)
		case Abs:
			f.funcCall(o, "ABS", 1)
		case Sign:
			f.funcCall(o, "SIGN", 1)
		case Swap:
			f.stack[len(f.stack)-1], f.stack[len(f.stack)-2] = f.stack[len(f.stack)-2], f.stack[len(f.stack)-1]
		case Dup:
			f.push(f.stack[len(f.stack)-1])
		case Drop:
			f.popN(1)
		case SignExtend:
			f.funcCall(o, "SIGN_EXTEND", 1)
		case Clamp:
			f.funcCall(o, "CLAMP", 3)
		case FindObject:
			f.funcCall(o, "FIND_OBJECT", 3)
		case CountNeighbourObjects:
			f.funcCall(o, "COUNT_NEIGHBOUR_OBJECTS", 3)
		case MagicNumber:
			f.funcCall(o, "MAGIC_NUMBER", 4)
		case AndNum:
			a := Atom{fmt.Sprintf("(%s)&%d", f.top(), v.b)}
			f.popNAndPush(1, a)
		case OrNum:
			a := Atom{fmt.Sprintf("(%s)|%d", f.top(), v.b)}
			f.popNAndPush(1, a)
		case XorNum:
			a := Atom{fmt.Sprintf("(%s)^%d", f.top(), v.b)}
			f.popNAndPush(1, a)
		case RotateRight:
			f.funcCall(o, fmt.Sprintf("ROT[%d]", v.b), 1)
		case PushSigned:
			a := Atom{fmt.Sprintf("%d", v.n)}
			f.push(a)
		case Push2Byte:
			a := Atom{fmt.Sprintf("0x%X", v.n)}
			f.push(a)
		case Push:
			a := Atom{fmt.Sprintf("%d", v.b)}
			f.push(a)
		case CoordsToMapAddress:
			f.funcCall(o, fmt.Sprintf("COORDS_TO_MAP_ADDRESS[%d]", v.b), 2)
		case IfNotBetweenSet:
			f.funcCall(o, fmt.Sprintf("IF_NOT_BETWEEN_SET[%d]", v.b), 3)
		case PushFrom:
			f.push(Atom{pushFromArgString(v.b)})
		default:
			panic(fmt.Sprintf("Unexpected opcode type %s", o.String()))
		}
		if stackLen-len(f.stack) != toPush-toPop {
			panic(fmt.Sprintf("Stack effect mismatch %d->%d vs %d->%d", stackLen, len(f.stack), toPush, toPop))
		}
		return
	}
	f.DumpStack()

	switch o.(type) {
	case IfGreaterThanZero:
		f.printIndent()
		f.scopes = append(f.scopes, IF)
	case IfZero:
		f.printIndent()
		f.scopes = append(f.scopes, IF)
	case IfNotEqual:
		f.printIndent()
		f.scopes = append(f.scopes, IF)
	case IfSignEq:
		f.printIndent()
		f.scopes = append(f.scopes, IF)
	case IfCmp:
		f.printIndent()
		f.scopes = append(f.scopes, IF)
	case Fi:
		//if f.scopes[len(f.scopes)-1] != IF {
		//	panic("FI not in an if statement")
		//}
		f.scopes = f.scopes[:len(f.scopes)-1]
		f.printIndent()
	case Else:
		f.scopes = f.scopes[:len(f.scopes)-1]
		f.printIndent()
		f.scopes = append(f.scopes, IF)
	case FiAll:
		for i := 0; i < len(f.scopes); i++ {
			if f.scopes[i] == IF {
				f.scopes = f.scopes[:i]
				break
			}
		}
		f.printIndent()
	case For:
		f.printIndent()
		f.scopes = append(f.scopes, FOR)
	case Done:
		//if f.scopes[len(f.scopes)-1] != FOR {
		//	panic("DONE not in a for loop")
		//}
		f.scopes = f.scopes[:len(f.scopes)-1]
		f.printIndent()
	default:
		f.printIndent()
	}

	fmt.Println(o.String())
	return
}

func (f *FoldingDecoder) printIndent() {
	for i := 0; i < len(f.scopes); i++ {
		fmt.Print("  ")
	}
}
func (f *FoldingDecoder) DumpStack() {
	for _, expr := range f.stack {
		f.printIndent()
		fmt.Println(expr.String())
	}
	f.stack = nil
}
