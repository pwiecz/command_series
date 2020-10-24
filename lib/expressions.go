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
	BaseMemoryAddress() (byte, bool)
	ReadsFromMemoryAddress(addr byte) bool
	ReadsFromVariable(v byte) bool
}

type Variable struct {
	id byte
}

func (v Variable) String() string                        { return varName(v.id) }
func (v Variable) Priority() int                         { return 20 }
func (v Variable) Type() Type                            { return varType(v.id) }
func (v Variable) BaseMemoryAddress() (byte, bool)       { return v.id, true }
func (v Variable) ReadsFromMemoryAddress(addr byte) bool { return false }
func (v Variable) ReadsFromVariable(v_ byte) bool        { return v_ == v.id }

type Num struct {
	n int
}

func (n Num) String() string                        { return fmt.Sprintf("%d", n.n) }
func (n Num) Priority() int                         { return 20 }
func (n Num) Type() Type                            { return NUMBER }
func (n Num) BaseMemoryAddress() (byte, bool)       { return 0, false }
func (n Num) ReadsFromMemoryAddress(addr byte) bool { return false }
func (n Num) ReadsFromVariable(v byte) bool         { return false }

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
func (s Sum) Priority() int { return 10 }
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
func (s Sum) BaseMemoryAddress() (byte, bool) {
	if s.arg0.Type() == ADDRESS {
		return s.arg0.BaseMemoryAddress()
	}
	if s.arg1.Type() == ADDRESS {
		return s.arg1.BaseMemoryAddress()
	}
	panic("Don't know yet")
}
func (s Sum) ReadsFromMemoryAddress(addr byte) bool {
	return s.arg0.ReadsFromMemoryAddress(addr) ||
		s.arg1.ReadsFromMemoryAddress(addr)
}
func (s Sum) ReadsFromVariable(v byte) bool {
	return s.arg0.ReadsFromVariable(v) ||
		s.arg1.ReadsFromVariable(v)
}

type ExclusiveOr struct {
	arg0, arg1 Expression
}

func (x ExclusiveOr) String() string {
	return fmt.Sprintf("%s ^ %s",
		inParensIfPriorityLessThan(x.arg0, x.Priority()),
		inParensIfPriorityLessThan(x.arg1, x.Priority()))
}
func (x ExclusiveOr) Priority() int {
	return 4
}
func (x ExclusiveOr) Type() Type {
	if x.arg0.Type() == ADDRESS {
		if x.arg1.Type() == ADDRESS {
			panic("Xor of two addresses")
		}
		return ADDRESS
	}
	if x.arg1.Type() == ADDRESS {
		return ADDRESS
	}
	return NUMBER
}
func (x ExclusiveOr) BaseMemoryAddress() (byte, bool) {
	if x.arg0.Type() == ADDRESS {
		return x.arg0.BaseMemoryAddress()
	}
	if x.arg1.Type() == ADDRESS {
		return x.arg1.BaseMemoryAddress()
	}
	panic("Don't know yet")
}
func (x ExclusiveOr) ReadsFromMemoryAddress(addr byte) bool {
	return x.arg0.ReadsFromMemoryAddress(addr) ||
		x.arg1.ReadsFromMemoryAddress(addr)
}
func (x ExclusiveOr) ReadsFromVariable(v byte) bool {
	return x.arg0.ReadsFromVariable(v) ||
		x.arg1.ReadsFromVariable(v)
}

type Difference struct {
	arg0, arg1 Expression
}

func (d Difference) String() string {
	return fmt.Sprintf("%s - %s",
		inParensIfPriorityLessThan(d.arg0, d.Priority()),
		inParensIfPriorityLessThan(d.arg1, d.Priority()+1))
}
func (d Difference) Priority() int { return 10 }
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
func (d Difference) BaseMemoryAddress() (byte, bool) {
	return d.arg0.BaseMemoryAddress()
}
func (d Difference) ReadsFromMemoryAddress(addr byte) bool {
	return d.arg0.ReadsFromMemoryAddress(addr) ||
		d.arg1.ReadsFromMemoryAddress(addr)
}
func (d Difference) ReadsFromVariable(v byte) bool {
	return d.arg0.ReadsFromVariable(v) ||
		d.arg1.ReadsFromVariable(v)
}

type UnaryPrefixOp struct {
	op       string
	arg      Expression
	priority int
}

func (o UnaryPrefixOp) String() string {
	return fmt.Sprintf("%s%s", o.op, inParensIfPriorityLessThan(o.arg, o.priority+1))
}
func (o UnaryPrefixOp) Priority() int { return o.priority }
func (o UnaryPrefixOp) Type() Type {
	if o.arg.Type() != NUMBER && o.arg.Type() != REFERENCE {
		panic(fmt.Sprintf("Invalid parameter to operator %s:\"%s\"(%s)", o.op, o.arg, o.arg.Type()))
	}
	return NUMBER
}
func (o UnaryPrefixOp) BaseMemoryAddress() (byte, bool) {
	return 0, false
}
func (o UnaryPrefixOp) ReadsFromMemoryAddress(addr byte) bool {
	return o.arg.ReadsFromMemoryAddress(addr)
}
func (o UnaryPrefixOp) ReadsFromVariable(v byte) bool {
	return o.arg.ReadsFromVariable(v)
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
func (o NonCommutativeBinaryOp) Priority() int { return o.priority }
func (o NonCommutativeBinaryOp) Type() Type {
	if (o.arg0.Type() != NUMBER && o.arg0.Type() != REFERENCE && o.arg0.Type() != UNDEFINED) ||
		(o.arg1.Type() != NUMBER && o.arg1.Type() != REFERENCE && o.arg1.Type() != UNDEFINED) {
		panic(fmt.Sprintf("Invalid parameters to operator %s:\"%s\"(%s) and \"%s\"(%s)",
			o.op, o.arg0, o.arg0.Type(), o.arg1, o.arg1.Type()))
	}
	return NUMBER
}
func (o NonCommutativeBinaryOp) BaseMemoryAddress() (byte, bool) {
	panic("Unexpected call")
}
func (o NonCommutativeBinaryOp) ReadsFromMemoryAddress(addr byte) bool {
	return o.arg0.ReadsFromMemoryAddress(addr) ||
		o.arg1.ReadsFromMemoryAddress(addr)
}
func (o NonCommutativeBinaryOp) ReadsFromVariable(v byte) bool {
	return o.arg0.ReadsFromVariable(v) ||
		o.arg1.ReadsFromVariable(v)
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
func (o CommutativeBinaryOp) Priority() int { return o.priority }
func (o CommutativeBinaryOp) Type() Type {
	if (o.arg0.Type() != NUMBER && o.arg0.Type() != REFERENCE && o.arg0.Type() != UNDEFINED) ||
		(o.arg1.Type() != NUMBER && o.arg1.Type() != REFERENCE && o.arg1.Type() != UNDEFINED) {
		panic(fmt.Sprintf("Invalid parameters to operator %s:\"%s\"(%s) and \"%s\"(%s)",
			o.op, o.arg0, o.arg0.Type(), o.arg1, o.arg1.Type()))
	}
	return NUMBER
}
func (o CommutativeBinaryOp) BaseMemoryAddress() (byte, bool) {
	panic(fmt.Sprintf("Unexpected call op:%s(%s) %s", o.op, o.Type(), o))
}
func (o CommutativeBinaryOp) ReadsFromMemoryAddress(addr byte) bool {
	return o.arg0.ReadsFromMemoryAddress(addr) ||
		o.arg1.ReadsFromMemoryAddress(addr)
}
func (o CommutativeBinaryOp) ReadsFromVariable(v byte) bool {
	return o.arg0.ReadsFromVariable(v) ||
		o.arg1.ReadsFromMemoryAddress(v)
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
func (m MultiplyShiftRightExpr) Priority() int                   { return 9 }
func (m MultiplyShiftRightExpr) Type() Type                      { return NUMBER }
func (m MultiplyShiftRightExpr) BaseMemoryAddress() (byte, bool) { panic("Unexpected call") }
func (m MultiplyShiftRightExpr) ReadsFromMemoryAddress(addr byte) bool {
	return m.arg0.ReadsFromMemoryAddress(addr) ||
		m.arg1.ReadsFromMemoryAddress(addr)
}
func (m MultiplyShiftRightExpr) ReadsFromVariable(v byte) bool {
	return m.arg0.ReadsFromVariable(v) ||
		m.arg1.ReadsFromVariable(v)
}

type FuncCall struct {
	name string
	args []Expression
	typ  Type
}

func (c FuncCall) String() string {
	argStrs := make([]string, 0, len(c.args))
	for _, arg := range c.args {
		argStrs = append(argStrs, arg.String())
	}
	return fmt.Sprintf("%s(%s)", c.name, strings.Join(argStrs, ", "))
}
func (c FuncCall) Priority() int                   { return 20 }
func (c FuncCall) Type() Type                      { return c.typ }
func (c FuncCall) BaseMemoryAddress() (byte, bool) { panic("Unexpected call") }
func (c FuncCall) ReadsFromMemoryAddress(addr byte) bool {
	for _, arg := range c.args {
		if arg.ReadsFromMemoryAddress(addr) {
			return true
		}
	}
	return false
}
func (c FuncCall) ReadsFromVariable(v byte) bool {
	for _, arg := range c.args {
		if arg.ReadsFromVariable(v) {
			return true
		}
	}
	return false

}

type ReadByteExpr struct {
	arg Expression
}

func (r ReadByteExpr) String() string                  { return fmt.Sprintf("[%s]", r.arg) }
func (r ReadByteExpr) Priority() int                   { return 20 }
func (r ReadByteExpr) Type() Type                      { return NUMBER }
func (r ReadByteExpr) BaseMemoryAddress() (byte, bool) { panic("Unexpected call") }
func (r ReadByteExpr) ReadsFromMemoryAddress(addr byte) bool {
	return r.arg.ReadsFromVariable(addr)
}
func (r ReadByteExpr) ReadsFromVariable(v byte) bool {
	return r.arg.ReadsFromVariable(v)
}

type ReadExpr struct {
	arg Expression
}

func (r ReadExpr) String() string                  { return fmt.Sprintf("[%s:]", r.arg) }
func (r ReadExpr) Priority() int                   { return 20 }
func (r ReadExpr) Type() Type                      { return NUMBER }
func (r ReadExpr) BaseMemoryAddress() (byte, bool) { panic("Unexpected call") }
func (r ReadExpr) ReadsFromMemoryAddress(addr byte) bool {
	return r.arg.ReadsFromVariable(addr)
}
func (r ReadExpr) ReadsFromVariable(addr byte) bool {
	return r.arg.ReadsFromVariable(addr)
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

func (f *FoldingDecoder) funcCall(o Opcode, name string, numArgs int, typ Type) {
	// Make sure the args slice is a copy of a piece of the stack, and it does not point
	// to the original stack array. Otherwise modifying stack modifies args as well.
	args := append(make([]Expression, 0, numArgs), f.stack[len(f.stack)-numArgs:]...)
	fc := FuncCall{name, args, typ}
	f.popNAndPush(numArgs, fc)
}

type Statement interface {
	fmt.Stringer
	AffectsExpressionValue(e Expression, stack []Expression) bool
}
type IfGreaterThanZeroStmt struct {
	arg Expression
}

func (i IfGreaterThanZeroStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	return true
}
func (i IfGreaterThanZeroStmt) String() string {
	return fmt.Sprintf("IF %s > 0 THEN", i.arg)
}

type IfZeroStmt struct {
	arg Expression
}

func (i IfZeroStmt) AffectsExpressionValue(e Expression, stack []Expression) bool { return true }
func (i IfZeroStmt) String() string {
	return fmt.Sprintf("IF %s == 0 THEN", i.arg)
}

type IfNotEqualStmt struct {
	arg0, arg1 Expression
}

func (i IfNotEqualStmt) AffectsExpressionValue(e Expression, stack []Expression) bool { return true }
func (i IfNotEqualStmt) String() string {
	return fmt.Sprintf("IF %s != %s THEN", i.arg0, i.arg1)
}

type IfSignEqStmt struct {
	arg Expression
	v   byte
}

func (i IfSignEqStmt) AffectsExpressionValue(e Expression, stack []Expression) bool { return true }
func (i IfSignEqStmt) String() string {
	if i.v == 255 {
		return fmt.Sprintf("IF %s < 0 THEN", i.arg)
	} else if i.v == 0 {
		return fmt.Sprintf("IF %s == 0 THEN", i.arg)
	} else if i.v == 1 {
		return fmt.Sprintf("IF %s > 0 THEN", i.arg)
	} else {
		panic("Unexpected sign value")
	}
}

type IfCmpStmt struct {
	arg0, arg1 Expression
	v          byte
}

func (i IfCmpStmt) AffectsExpressionValue(e Expression, stack []Expression) bool { return true }
func (i IfCmpStmt) String() string {
	if i.v == 255 {
		return fmt.Sprintf("IF %s < %s THEN", i.arg0, i.arg1)
	} else if i.v == 0 {
		return fmt.Sprintf("IF %s == %s THEN", i.arg0, i.arg1)
	} else if i.v == 1 {
		return fmt.Sprintf("IF %s > %s THEN", i.arg0, i.arg1)
	} else {
		panic("Unexpected cmp value")
	}
}

type WriteToA200PlusStmt struct {
	offset, value Expression
}

func (w WriteToA200PlusStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	return false
}
func (w WriteToA200PlusStmt) String() string {
	return fmt.Sprintf("[A200+%s] = %s", inParensIfPriorityLessThan(w.offset, 10), w.value)
}

type PopToD4Stmt struct {
	arg Expression
}

func (p PopToD4Stmt) AffectsExpressionValue(e Expression, stack []Expression) bool { return false }
func (p PopToD4Stmt) String() string {
	return fmt.Sprintf("D4 = %s", p.arg)
}

type StoreByteStmt struct {
	location, value Expression
}

func (s StoreByteStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	addr, ok := s.location.BaseMemoryAddress()
	return ok && e.ReadsFromMemoryAddress(addr)
}
func (s StoreByteStmt) String() string {
	if s.location.Type() == REFERENCE {
		return fmt.Sprintf("%s = %s", s.location, s.value)
	} else {
		return fmt.Sprintf("[%s] = %s", s.location, s.value)
	}

}

type StoreStmt struct {
	location, value Expression
}

func (s StoreStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	addr, ok := s.location.BaseMemoryAddress()
	return ok && e.ReadsFromMemoryAddress(addr)
}
func (s StoreStmt) String() string {
	if s.location.Type() == REFERENCE {
		//panic("Storing two-byte value in one-byte variable")
	}
	return fmt.Sprintf("[%s:] = %s", s.location, s.value)
}

type ForStmt struct {
	v        byte
	from, to Expression
}

func (f ForStmt) AffectsExpressionValue(e Expression, stack []Expression) bool { return true }
func (f ForStmt) String() string {
	return fmt.Sprintf("FOR %s = %s TO %s DO", varName(f.v), f.from, f.to)
}

type PopToStmt struct {
	v     byte
	value Expression
}

func (p PopToStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	return e.ReadsFromVariable(p.v)
}
func (p PopToStmt) String() string {
	if varType(p.v) != ADDRESS && varType(p.v) != UNDEFINED && p.value.Type() == ADDRESS {
		panic("Writing address value " + p.value.String() + " into " + varName(p.v))
	}
	// We don't panic if a number is writting into a variable of type ADDRESS.
	// In program 17 a hardcoded address of flashback data is being used.
	return fmt.Sprintf("%s = %s", varName(p.v), p.value)
}

type FillStmt struct {
	address, size Expression
	val           byte
}

func (f FillStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	addr, ok := f.address.BaseMemoryAddress()
	return ok && e.ReadsFromMemoryAddress(addr)
}
func (f FillStmt) String() string {
	return fmt.Sprintf("FILL(%s, %s, %d)", f.address, f.size, f.val)
}

type LoadUnitStmt struct {
	v   byte
	arg Expression
}

func (l LoadUnitStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	for i := 0; i < 16; i++ {
		if e.ReadsFromVariable(byte(int(l.v) + 17 + i)) {
			return true
		}
	}
	return false
}
func (l LoadUnitStmt) String() string {
	if l.v == 15 {
		return fmt.Sprintf("LOAD_UNIT1(%s)", l.arg)
	} else if l.v == 31 {
		return fmt.Sprintf("LOAD_UNIT2(%s)", l.arg)
	} else {
		return fmt.Sprintf("LOAD_UNIT[%d](%s)", l.v, l.arg)
	}

}

type SaveUnitStmt struct {
	v   byte
	arg Expression
}

func (l SaveUnitStmt) AffectsExpressionValue(e Expression, stack []Expression) bool {
	a, ok := l.arg.BaseMemoryAddress()
	return ok && e.ReadsFromMemoryAddress(a)
}
func (l SaveUnitStmt) String() string {
	if l.v == 15 {
		return fmt.Sprintf("SAVE_UNIT1(%s)", l.arg)
	} else if l.v == 31 {
		return fmt.Sprintf("SAVE_UNIT2(%s)", l.arg)
	} else {
		return fmt.Sprintf("SAVE_UNIT[%d](%s)", l.v, l.arg)
	}

}

func (f *FoldingDecoder) Apply(o Opcode) {
	toPop, toPush := o.StackEffect()
	if !o.HasSideEffects() && toPop <= len(f.stack) {
		stackLen := len(f.stack)
		switch v := o.(type) {
		case Byte:
			f.push(Num{int(v.b)})
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
			i := Sum{f.top(), Num{1}}
			f.popNAndPush(1, i)
		case Decrement:
			d := Difference{f.top(), Num{1}}
			f.popNAndPush(1, d)
		case AdditiveInverse:
			if f.top().Type() != NUMBER && f.top().Type() != REFERENCE && f.top().Type() != UNDEFINED {
				panic("Invalid argument to unary -")
			}
			i := UnaryPrefixOp{"-", f.top(), 11 /* ? TODO: check */}
			f.popNAndPush(1, i)
		case And0xFF:
			a := CommutativeBinaryOp{"&", f.top(), Num{255}, 5}
			f.popNAndPush(1, a)
		case BinaryAnd:
			f.commutativeBinaryOp("&", 5)
		case BinaryOr:
			f.commutativeBinaryOp("|", 3)
		case BinaryXor:
			x := ExclusiveOr{f.belowTop(), f.top()}
			f.popNAndPush(2, x)
		case ShiftLeft:
			s := CommutativeBinaryOp{"<<", f.top(), Num{int(v.shift)}, 9}
			f.popNAndPush(1, s)
		case ArithmeticShiftRight:
			s := CommutativeBinaryOp{">>", f.top(), Num{int(v.shift)}, 9}
			f.popNAndPush(1, s)
		case ScnDtaUnitTypeOffset:
			unitType := CommutativeBinaryOp{"&", Variable{39}, Num{15}, 5}
			if v.offset > 0 {
				a := ReadByteExpr{Sum{Variable{5}, Sum{Num{int(v.offset)}, unitType}}}
				f.push(a)
			} else {
				a := ReadByteExpr{Sum{Variable{5}, unitType}}
				f.push(a)
			}
		case ReadByte:
			r := ReadByteExpr{f.top()}
			f.popNAndPush(1, r)
		case Read:
			r := ReadExpr{f.top()}
			f.popNAndPush(1, r)
		case ReadByteWithOffset:
			a := Sum{f.top(), Num{int(v.offset)}}
			r := ReadByteExpr{a}
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
			x := ExclusiveOr{f.top(), Num{int(v.b)}}
			f.popNAndPush(1, x)
		case LogicalShiftRight:
			s := CommutativeBinaryOp{">>>", f.top(), Num{int(v.shift)}, 9}
			f.popNAndPush(1, s)
		case PushSigned:
			a := Num{int(v.n)}
			f.push(a)
		case Push2Byte:
			a := Num{int(v.n)}
			f.push(a)
		case Push:
			a := Num{int(v.b)}
			f.push(a)
		case CoordsToMapAddress:
			f.funcCall(o, fmt.Sprintf("COORDS_TO_MAP_ADDRESS[%d]", v.b), 2, ADDRESS)
		case IfNotBetweenSet:
			f.funcCall(o, fmt.Sprintf("IF_NOT_BETWEEN_SET[%d]", v.b), 3, NUMBER)
		case PushFrom:
			f.push(Variable{v.b})
		default:
			panic(fmt.Sprintf("Unexpected opcode type %s", o.String()))
		}
		if len(f.stack)-stackLen != toPush-toPop {
			panic(fmt.Sprintf("Stack effect mismatch for opcode %s %d->%d vs %d->%d", o.String(), stackLen, len(f.stack), toPush, toPop))
		}
		return
	} else if o.HasSideEffects() && toPop > 0 && toPop <= len(f.stack) {
		var stmt Statement
		switch v := o.(type) {
		case IfGreaterThanZero:
			stmt = IfGreaterThanZeroStmt{f.top()}
		case IfZero:
			stmt = IfZeroStmt{f.top()}
		case IfNotEqual:
			stmt = IfNotEqualStmt{f.belowTop(), f.top()}
		case IfSignEq:
			stmt = IfSignEqStmt{f.top(), v.b}
		case IfCmp:
			stmt = IfCmpStmt{f.belowTop(), f.top(), v.b}
		case WriteToA200Plus:
			stmt = WriteToA200PlusStmt{f.top(), f.belowTop()}
		case PopToD4:
			stmt = PopToD4Stmt{f.top()}
		case StoreByte:
			stmt = StoreByteStmt{f.top(), f.belowTop()}
		case Store:
			stmt = StoreStmt{f.top(), f.belowTop()}
		case For:
			stmt = ForStmt{v.b, f.belowTop(), f.top()}
		case PopTo:
			stmt = PopToStmt{v.b, f.top()}
		case Fill:
			stmt = FillStmt{f.belowTop(), f.top(), v.b}
		case LoadUnit:
			stmt = LoadUnitStmt{v.b, f.top()}
		case SaveUnit:
			stmt = SaveUnitStmt{v.b, f.top()}
		default:
			panic(fmt.Sprintf("Unhandled opcode %s", o.String()))
		}
		f.popN(toPop)
		lastAffectedExpression := 0
		for i, expr := range f.stack {
			if stmt.AffectsExpressionValue(expr, f.stack) {
				lastAffectedExpression = i + 1
			}
		}
		for _, expr := range f.stack[:lastAffectedExpression] {
			f.printIndent()
			fmt.Printf("PUSH(%s)\n", expr.String())
		}
		f.stack = f.stack[lastAffectedExpression:]
		f.printIndent()
		fmt.Println(stmt)
		switch o.(type) {
		case IfGreaterThanZero:
			f.scopes = append(f.scopes, IF)
		case IfZero:
			f.scopes = append(f.scopes, IF)
		case IfNotEqual:
			f.scopes = append(f.scopes, IF)
		case IfSignEq:
			f.scopes = append(f.scopes, IF)
		case IfCmp:
			f.scopes = append(f.scopes, IF)
		case For:
			f.scopes = append(f.scopes, FOR)
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
