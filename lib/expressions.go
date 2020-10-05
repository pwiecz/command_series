package lib

import "fmt"
import "strings"

type Type int

const NUMBER = Type(0)
const ADDRESS = Type(1)
const REFERENCE = Type(2)
const UNDEFINED = Type(3)

func (t Type) String() string {
	switch t {
	case NUMBER:
		return "NUMBER"
	case ADDRESS:
		return "ADDRESS"
	case REFERENCE:
		return "REFERENCE"
	case UNDEFINED:
		return "UNDEFINED"
	}
	return fmt.Sprintf("UNKNOWN<%d>", t)
}

type Expression interface {
	fmt.Stringer
	Priority() int
	Type() Type
}

type Atom struct {
	s   string
	typ Type
}

func (a Atom) String() string { return a.s }
func (a Atom) Priority() int  { return 20 }
func (a Atom) Type() Type     { return a.typ }

type Num struct {
	n int
}

func (n Num) String() string { return fmt.Sprintf("%d", n.n) }
func (n Num) Priority() int  { return 10 }
func (n Num) Type() Type     { return NUMBER }

func inParensIfPriorityLessThan(e Expression, priority int) string {
	if e.Priority() < priority {
		return fmt.Sprintf("(%s)", e)
	}
	return e.String()
}

type Sum struct {
	arg0, arg1 Expression
}

func (s Sum) String() string {
	return fmt.Sprintf("%s + %s",
		inParensIfPriorityLessThan(s.arg0, s.Priority()),
		inParensIfPriorityLessThan(s.arg1, s.Priority()))
}
func (s Sum) Priority() int {
	return 10
}
func (s Sum) Type() Type {
	if s.arg0.Type() == ADDRESS {
		if s.arg1.Type() == ADDRESS {
			panic("Sum of two addresses")
		}
		return ADDRESS
	}
	if s.arg1.Type() == ADDRESS {
		return ADDRESS
	}
	return NUMBER
}

type Difference struct {
	arg0, arg1 Expression
}

func (d Difference) String() string {
	return fmt.Sprintf("%s - %s",
		inParensIfPriorityLessThan(d.arg0, d.Priority()),
		inParensIfPriorityLessThan(d.arg1, d.Priority()+1))
}
func (d Difference) Priority() int {
	return 10
}
func (d Difference) Type() Type {
	if d.arg0.Type() == ADDRESS {
		if d.arg1.Type() == ADDRESS {
			panic("Difference of two addresses")
		}
		return ADDRESS
	}
	if d.arg1.Type() == ADDRESS {
		panic("Subtracting address from number or reference")
	}
	return NUMBER
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
func (o NonCommutativeBinaryOp) Type() Type {
	if (o.arg0.Type() != NUMBER && o.arg0.Type() != REFERENCE && o.arg0.Type() != UNDEFINED) ||
		(o.arg1.Type() != NUMBER && o.arg1.Type() != REFERENCE && o.arg1.Type() != UNDEFINED) {
		panic(fmt.Sprintf("Invalid parameters to operator %s:\"%s\"(%s) and \"%s\"(%s)",
			o.op, o.arg0, o.arg0.Type(), o.arg1, o.arg1.Type()))
	}
	return NUMBER
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
func (o CommutativeBinaryOp) Type() Type {
	if (o.arg0.Type() != NUMBER && o.arg0.Type() != REFERENCE && o.arg0.Type() != UNDEFINED) ||
		(o.arg1.Type() != NUMBER && o.arg1.Type() != REFERENCE && o.arg1.Type() != UNDEFINED) {
		panic(fmt.Sprintf("Invalid parameters to operator %s:\"%s\"(%s) and \"%s\"(%s)",
			o.op, o.arg0, o.arg0.Type(), o.arg1, o.arg1.Type()))
	}
	return NUMBER
}

type MultiplyShiftRightExpr struct {
	arg0, arg1 Expression
	shift      int
}

func (m MultiplyShiftRightExpr) String() string {
	return fmt.Sprintf("(%s) ** (%s) >> %d",
		inParensIfPriorityLessThan(m.arg0, 11),
		inParensIfPriorityLessThan(m.arg1, 11),
		m.shift)
}
func (m MultiplyShiftRightExpr) Priority() int {
	return 9
}
func (m MultiplyShiftRightExpr) Type() Type { return NUMBER }

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

func (f *FoldingDecoder) funcCall(o Opcode, name string, numArgs int, typ Type) {
	args := make([]string, numArgs)
	for i, expr := range f.stack[len(f.stack)-numArgs:] {
		args[i] = expr.String()
	}
	a := Atom{fmt.Sprintf("%s(%s)", name, strings.Join(args, ", ")), typ}
	f.popNAndPush(numArgs, a)
}

func (f *FoldingDecoder) Apply(o Opcode) {
	toPop, toPush := o.StackEffect()
	if !o.HasSideEffects() && toPop <= len(f.stack) {
		stackLen := len(f.stack)
		switch v := o.(type) {
		case Byte:
			f.push(Atom{o.String(), NUMBER})
		case Add:
			s := Sum{f.belowTop(), f.top()}
			f.popNAndPush(2, s)
		case Subtract:
			d := Difference{f.belowTop(), f.top()}
			f.popNAndPush(2, d)
		case Multiply:
			f.commutativeBinaryOp("*", 11)
		case Divide:
			f.nonCommutativeBinaryOp("/", 11)
		case MultiplyShiftRight:
			f.multiplyShiftRight(int(v.b))
		case Increment:
			i := Sum{f.top(), Atom{"1", NUMBER}}
			f.popNAndPush(1, i)
		case Decrement:
			d := Difference{f.top(), Atom{"1", NUMBER}}
			f.popNAndPush(1, d)
		case AdditiveInverse:
			if f.top().Type() != NUMBER && f.top().Type() != REFERENCE && f.top().Type() != UNDEFINED {
				panic("Invalid argument to unary -")
			}
			i := Atom{"-(" + f.top().String() + ")", NUMBER}
			f.popNAndPush(1, i)
		case And0xFF:
			a := CommutativeBinaryOp{"&", f.top(), Atom{"0xFF", NUMBER}, 5}
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
		case ArithmeticShiftRight:
			s := CommutativeBinaryOp{">>", f.top(), Num{int(v.shift)}, 9}
			f.popNAndPush(1, s)
		case ScnDtaUnitTypeOffset:
			a := Atom{v.String(), NUMBER}
			f.push(a)
		case ReadByte:
			a := Atom{fmt.Sprintf("[%s]", f.top()), NUMBER}
			f.popNAndPush(1, a)
		case Read:
			a := Atom{fmt.Sprintf("[%s:]", f.top()), NUMBER}
			f.popNAndPush(1, a)
		case ReadByteWithOffset:
			a := CommutativeBinaryOp{"+", f.top(), Num{int(v.offset)}, 10}
			r := Atom{fmt.Sprintf("[%s]", a), NUMBER}
			f.popNAndPush(1, r)
		case MulRandShiftRight8:
			f.funcCall(o, "MUL_RAND_SHR8", 1, NUMBER)
		case Abs:
			f.funcCall(o, "ABS", 1, NUMBER)
		case Sign:
			f.funcCall(o, "SIGN", 1, NUMBER)
		case Swap:
			f.stack[len(f.stack)-1], f.stack[len(f.stack)-2] = f.stack[len(f.stack)-2], f.stack[len(f.stack)-1]
		case Dup:
			f.push(f.stack[len(f.stack)-1])
		case Drop:
			f.popN(1)
		case SignExtend:
			f.funcCall(o, "SIGN_EXTEND", 1, NUMBER)
		case Clamp:
			f.funcCall(o, "CLAMP", 3, NUMBER)
		case FindObject:
			f.funcCall(o, "FIND_OBJECT", 3, ADDRESS)
		case CountNeighbourObjects:
			f.funcCall(o, "COUNT_NEIGHBOUR_OBJECTS", 3, NUMBER)
		case MagicNumber:
			f.funcCall(o, "MAGIC_NUMBER", 4, NUMBER)
		case AndNum:
			a := CommutativeBinaryOp{"&", f.top(), Num{int(v.b)}, 5}
			f.popNAndPush(1, a)
		case OrNum:
			a := CommutativeBinaryOp{"|", f.top(), Num{int(v.b)}, 3}
			f.popNAndPush(1, a)
		case XorNum:
			a := CommutativeBinaryOp{"^", f.top(), Num{int(v.b)}, 4}
			f.popNAndPush(1, a)
		case LogicalShiftRight:
			s := CommutativeBinaryOp{">>>", f.top(), Num{int(v.shift)}, 9}
			f.popNAndPush(1, s)
		case PushSigned:
			a := Atom{fmt.Sprintf("%d", v.n), NUMBER}
			f.push(a)
		case Push2Byte:
			a := Atom{fmt.Sprintf("0x%X", v.n), NUMBER}
			f.push(a)
		case Push:
			a := Atom{fmt.Sprintf("%d", v.b), NUMBER}
			f.push(a)
		case CoordsToMapAddress:
			f.funcCall(o, fmt.Sprintf("COORDS_TO_MAP_ADDRESS[%d]", v.b), 2, ADDRESS)
		case IfNotBetweenSet:
			f.funcCall(o, fmt.Sprintf("IF_NOT_BETWEEN_SET[%d]", v.b), 3, NUMBER)
		case PushFrom:
			f.push(Atom{varName(v.b), varType(v.b)})
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
			if f.top().Type() == REFERENCE {
				fmt.Printf("%s = %s\n", f.top(), f.belowTop())
			} else {
				fmt.Printf("[%s] = %s\n", f.top(), f.belowTop())
			}
		case Store:
			if f.top().Type() == REFERENCE {
				//panic("Storing two-byte value in one-byte variable")
			} else {
				fmt.Printf("[%s] = %s\n", f.top(), f.belowTop())
			}
		case For:
			fmt.Printf("FOR %s = %s TO %s DO\n", varName(v.b), f.belowTop(), f.top())
			f.scopes = append(f.scopes, FOR)
		case PopTo:
			if varType(v.b) != ADDRESS && varType(v.b) != UNDEFINED && f.top().Type() == ADDRESS {
				panic("Writing address value " + f.top().String() + " into " + varName(v.b))
			}
			// We don't panic if a number is writting into a variable of type ADDRESS.
			// In program 17 a hardcoded address of flashback data is being used.
			fmt.Printf("%s = %s\n", varName(v.b), f.top())
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
		} else {
			panic("FI not in an if statement")
		}
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
		} //else {
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
func varType(b byte) Type {
	name := varName(b)
	// Those variables are being used as both numbers and addresses
	if name == "TEMP" || name == "TEMP2" || name == "ARG1" || name == "ARG2" || name == "INDEX" || name == "INDEX2" || name == "UNIT2.SUPPLY" || name == "RANGE" {
		return UNDEFINED
	}
	if name[0] == '&' {
		return ADDRESS
	}
	return REFERENCE
}
