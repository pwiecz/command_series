package lib

import "fmt"
import "strings"

type Expression interface {
	fmt.Stringer
	Priority() int
}

type Atom struct {
	s string
}

func (a Atom) String() string { return a.s }
func (a Atom) Priority() int  { return 20 }

type Num struct {
	n int
}

func (n Num) String() string { return fmt.Sprintf("%d", n.n) }
func (n Num) Priority() int  { return 10 }

func inParensIfPriorityLessThan(e Expression, priority int) string {
	if e.Priority() < priority {
		return fmt.Sprintf("(%s)", e)
	}
	return e.String()
}

type NonCommutativeBinaryOp struct {
	op         string
	arg0, arg1 Expression
	priority   int
}

func (o NonCommutativeBinaryOp) String() string {
	return fmt.Sprintf("%s %s %s",
		inParensIfPriorityLessThan(o.arg0, o.priority),
		o.op,
		inParensIfPriorityLessThan(o.arg1, o.priority+1))
}
func (o NonCommutativeBinaryOp) Priority() int {
	return o.priority
}

type CommutativeBinaryOp struct {
	op         string
	arg0, arg1 Expression
	priority   int
}

func (o CommutativeBinaryOp) String() string {
	return fmt.Sprintf("%s %s %s",
		inParensIfPriorityLessThan(o.arg0, o.priority),
		o.op,
		inParensIfPriorityLessThan(o.arg1, o.priority))
}
func (o CommutativeBinaryOp) Priority() int {
	return o.priority
}

type MultiplyShiftRightExpr struct {
	arg0, arg1 Expression
	shift      int
}

func (m MultiplyShiftRightExpr) String() string {
	return fmt.Sprintf("%s * %s >> %d",
		inParensIfPriorityLessThan(m.arg0, 11),
		inParensIfPriorityLessThan(m.arg1, 11),
		m.shift)
}
func (m MultiplyShiftRightExpr) Priority() int {
	return 9
}

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

func (f *FoldingDecoder) commutativeBinaryOp(op string, priority int) {
	o := CommutativeBinaryOp{op, f.belowTop(), f.top(), priority}
	f.popNAndPush(2, o)
}
func (f *FoldingDecoder) nonCommutativeBinaryOp(op string, priority int) {
	o := NonCommutativeBinaryOp{op, f.belowTop(), f.top(), priority}
	f.popNAndPush(2, o)
}
func (f *FoldingDecoder) multiplyShiftRight(shift int) {
	m := MultiplyShiftRightExpr{f.belowTop(), f.top(), shift}
	f.popNAndPush(2, m)
}

func (f *FoldingDecoder) funcCall(o Opcode, name string, numArgs int) {
	args := make([]string, numArgs)
	for i, expr := range f.stack[len(f.stack)-numArgs:] {
		args[i] = expr.String()
	}
	a := Atom{fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))}
	f.popNAndPush(numArgs, a)
}

func (f *FoldingDecoder) Apply(o Opcode) {
	toPop, toPush := o.StackEffect()
	if !o.HasSideEffects() && toPop <= len(f.stack) {
		stackLen := len(f.stack)
		switch v := o.(type) {
		case Byte:
			f.push(Atom{o.String()})
		case Add:
			f.commutativeBinaryOp("+", 10)
		case Subtract:
			f.nonCommutativeBinaryOp("-", 10)
		case Multiply:
			f.commutativeBinaryOp("*", 11)
		case Divide:
			f.nonCommutativeBinaryOp("/", 11)
		case MultiplyShiftRight:
			f.multiplyShiftRight(int(v.b))
		case Increment:
			i := CommutativeBinaryOp{"+", f.top(), Atom{"1"}, 10}
			f.popNAndPush(1, i)
		case Decrement:
			d := NonCommutativeBinaryOp{"-", f.top(), Atom{"1"}, 10}
			f.popNAndPush(1, d)
		case AdditiveInverse:
			i := Atom{"-(" + f.top().String() + ")"}
			f.popNAndPush(1, i)
		case And0xFF:
			a := CommutativeBinaryOp{"&", f.top(), Atom{"0xFF"}, 5}
			f.popNAndPush(1, a)
		case BinaryAnd:
			f.commutativeBinaryOp("&", 5)
		case BinaryOr:
			f.commutativeBinaryOp("|", 3)
		case BinaryXor:
			f.commutativeBinaryOp("^", 4)
		case ShiftLeft:
			s := CommutativeBinaryOp{"<<", f.top(), Num{int(v.shift)}, 9}
			f.popNAndPush(1, s)
		case ShiftRight:
			s := CommutativeBinaryOp{">>", f.top(), Num{int(v.shift)}, 9}
			f.popNAndPush(1, s)
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
			a := CommutativeBinaryOp{"+", f.top(), Num{int(v.offset)}, 10}
			r := Atom{fmt.Sprintf("[%s]", a)}
			f.popNAndPush(1, r)
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
			a := CommutativeBinaryOp{"&", f.top(), Num{int(v.b)}, 5}
			f.popNAndPush(1, a)
		case OrNum:
			a := CommutativeBinaryOp{"|", f.top(), Num{int(v.b)}, 3}
			f.popNAndPush(1, a)
		case XorNum:
			a := CommutativeBinaryOp{"^", f.top(), Num{int(v.b)}, 4}
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
		if len(f.stack)-stackLen != toPush-toPop {
			panic(fmt.Sprintf("Stack effect mismatch for opcode %s %d->%d vs %d->%d", o.String(), stackLen, len(f.stack), toPush, toPop))
		}
		return
	} else if o.HasSideEffects() && toPop > 0 && toPop <= len(f.stack) {
		for _, expr := range f.stack[:len(f.stack)-toPop] {
			f.printIndent()
			fmt.Printf("PUSH(%s)\n", expr.String())
		}
		f.stack = f.stack[len(f.stack)-toPop:]
		f.printIndent()
		switch v := o.(type) {
		case IfGreaterThanZero:
			fmt.Printf("IF %s > 0 THEN\n", f.top())
			f.scopes = append(f.scopes, IF)
		case IfZero:
			fmt.Printf("IF %s == 0 THEN\n", f.top())
			f.scopes = append(f.scopes, IF)
		case IfNotEqual:
			fmt.Printf("IF %s != %s THEN\n", f.belowTop(), f.top())
			f.scopes = append(f.scopes, IF)
		case IfSignEq:
			if v.b == 255 {
				fmt.Printf("IF %s < 0 THEN\n", f.top())
			} else if v.b == 0 {
				fmt.Printf("IF %s == 0 THEN\n", f.top())
			} else if v.b == 1 {
				fmt.Printf("IF %s > 0 THEN\n", f.top())
			} else {
				panic("Unexpected sign value")
			}
			f.scopes = append(f.scopes, IF)
		case IfCmp:
			if v.b == 255 {
				fmt.Printf("IF %s < %s THEN\n", f.belowTop(), f.top())
			} else if v.b == 0 {
				fmt.Printf("IF %s == %s THEN\n", f.belowTop(), f.top())
			} else if v.b == 1 {
				fmt.Printf("IF %s > %s THEN\n", f.belowTop(), f.top())
			} else {
				panic("Unexpected cmp value")
			}
			f.scopes = append(f.scopes, IF)
		case WriteToA200Plus:
			fmt.Printf("[A200+%s] = %s\n", inParensIfPriorityLessThan(f.top(), 10), f.belowTop())
		case PopToD4:
			fmt.Printf("D4 = %s\n", f.top())
		case StoreByte:
			fmt.Printf("[%s] = %s\n", f.top(), f.belowTop())
		case Store:
			fmt.Printf("[%s:] = %s\n", f.top(), f.belowTop())
		case For:
			fmt.Printf("FOR V%d = %s TO %s DO\n", v.b, f.belowTop(), f.top())
			f.scopes = append(f.scopes, FOR)
		case PopTo:
			fmt.Printf("[%s] = %s\n", numToUnitField(v.b), f.top())
		case Fill:
			fmt.Printf("FILL(%s, %s, %d)\n", f.belowTop(), f.top(), v.b)
		case LoadUnit:
			if v.b == 15 {
				fmt.Printf("LOAD_UNIT1(%s)\n", f.top())
			} else if v.b == 31 {
				fmt.Printf("LOAD_UNIT2(%s)\n", f.top())
			} else {
				fmt.Printf("LOAD_UNIT[%d](%s)\n", v.b, f.top())
			}
		default:
			if toPop > 0 {
				panic(fmt.Sprintf("Unhandled opcode %s", o.String()))
			}
		}
		f.stack = nil
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
		if f.scopes[len(f.scopes)-1] == IF {
			f.scopes = f.scopes[:len(f.scopes)-1]
		} // else {
		//	panic("FI not in an if statement")
		//}
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
		if f.scopes[len(f.scopes)-1] == FOR {
			f.scopes = f.scopes[:len(f.scopes)-1]
		} // else {
		//	panic("DONE not in a for loop")
		//}
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
		fmt.Printf("PUSH(%s)\n", expr.String())
	}
	f.stack = nil
}
