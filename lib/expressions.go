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
	switch v := o.(type) {
	case Byte:
		f.push(Atom{o.String()})
	case Add:
		if f.binaryOp(o, "+") {
			return
		}
	case Subtract:
		if f.binaryOp(o, "-") {
			return
		}
	case Multiply:
		if f.binaryOp(o, "*") {
			return
		}
	case Divide:
		if f.binaryOp(o, "/") {
			return
		}
	case MultiplyShiftRight:
		if len(f.stack) >= 2 {
			a := Atom{fmt.Sprintf("(%s)*(%s)>>%d", f.belowTop(), f.top(), v.b)}
			f.popNAndPush(2, a)
			return
		}
	case Increment:
		if len(f.stack) >= 1 {
			a := Atom{"(" + f.top().String() + ")+1"}
			f.popNAndPush(1, a)
			return
		}
	case Decrement:
		if len(f.stack) >= 1 {
			a := Atom{"(" + f.top().String() + ")-1"}
			f.popNAndPush(1, a)
			return
		}
	case AdditiveInverse:
		if len(f.stack) >= 1 {
			a := Atom{"-(" + f.top().String() + ")"}
			f.popNAndPush(1, a)
			return
		}
	case And_0xFF:
		if len(f.stack) >= 1 {
			a := Atom{"(" + f.top().String() + ")&0xFF"}
			f.popNAndPush(1, a)
			return
		}
	case BinaryAnd:
		if f.binaryOp(o, "&") {
			return
		}
	case BinaryOr:
		if f.binaryOp(o, "|") {
			return
		}
	case BinaryXor:
		if f.binaryOp(o, "^") {
			return
		}
	case ShiftLeft:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("(%s)<<%d", f.top(), v.shift)}
			f.popNAndPush(1, a)
			return
		}
	case ShiftRight:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("(%s)>>%d", f.top(), v.shift)}
			f.popNAndPush(1, a)
			return
		}
	case ScnDtaUnitTypeOffset:
		a := Atom{fmt.Sprintf("[&SCN_DTA+%d+UNIT.TYPE]", v.offset)}
		f.push(a)
		return
	case ReadByte:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("[%s]", f.top())}
			f.popNAndPush(1, a)
			return
		}
	case Read:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("[%s:]", f.top())}
			f.popNAndPush(1, a)
			return
		}
	case ReadByteWithOffset:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("[(%s)+%d]", f.top(), v.offset)}
			f.popNAndPush(1, a)
			return
		}
	case MulRandShiftRight8:
		if f.funcCall(o, "MUL_RAND_SHR8", 1) {
			return
		}
	case Abs:
		if f.funcCall(o, "ABS", 1) {
			return
		}
	case Sign:
		if f.funcCall(o, "SIGN", 1) {
			return
		}
	case Swap:
		if len(f.stack) >= 2 {
			f.stack[len(f.stack)-1], f.stack[len(f.stack)-2] = f.stack[len(f.stack)-2], f.stack[len(f.stack)-1]
			return
		}
	case Dup:
		if len(f.stack) >= 1 {
			f.push(f.stack[len(f.stack)-1])
			return
		}
	case Drop:
		if len(f.stack) >= 1 {
			f.popN(1)
			return
		}
	case SignExtend:
		if f.funcCall(o, "SIGN_EXTEND", 1) {
			return
		}
	case Clamp:
		if f.funcCall(o, "CLAMP", 3) {
			return
		}
	case FindObject:
		if f.funcCall(o, "FIND_OBJECT", 3) {
			return
		}
	case CountNeighbourObjects:
		if f.funcCall(o, "COUNT_NEIGHBOUR_OBJECTS", 3) {
			return
		}
	case MagicNumber:
		if f.funcCall(o, "MAGIC_NUMBER", 4) {
			return
		}
	case AndNum:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("(%s)&%d", f.top(), v.b)}
			f.popNAndPush(1, a)
			return
		}
	case OrNum:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("(%s)|%d", f.top(), v.b)}
			f.popNAndPush(1, a)
			return
		}
	case XorNum:
		if len(f.stack) >= 1 {
			a := Atom{fmt.Sprintf("(%s)^%d", f.top(), v.b)}
			f.popNAndPush(1, a)
		}
	case RotateRight:
		if f.funcCall(o, fmt.Sprintf("ROT[%d]", v.b), 1) {
			return
		}
	case PushSigned:
		a := Atom{fmt.Sprintf("%d", v.n)}
		f.push(a)
		return
	case Push2Byte:
		a := Atom{fmt.Sprintf("0x%X", v.n)}
		f.push(a)
		return
	case Push:
		a := Atom{fmt.Sprintf("%d", v.b)}
		f.push(a)
		return
	case CoordsToMapAddress:
		if f.funcCall(o, fmt.Sprintf("COORDS_TO_MAP_ADDRESS[%d]", v.b), 2) {
			return
		}
	case IfNotBetweenSet:
		if f.funcCall(o, fmt.Sprintf("IF_NOT_BETWEEN_SET[%d]", v.b), 3) {
			return
		}
	case PushFrom:
		f.push(Atom{pushFromArgString(v.b)})
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
		if f.scopes[len(f.scopes)-1] != IF {
//			panic("FI not in if statement")
		}
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
		if f.scopes[len(f.scopes) - 1] != FOR {
//			panic("DONE not in for loop")
		}
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
	//for i := len(f.stack) - 1; i >= 0; i-- {
	//	f.printIndent()
	//	fmt.Println(f.stack[i].String())
	//}
	f.stack = nil
}
