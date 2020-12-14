package data

type IntelligenceType int

const (
	Full    IntelligenceType = 0
	Limited IntelligenceType = 1
)

type Options struct {
	AlliedCommander int // [0..1]
	GermanCommander int // [0..1]
	Intelligence    IntelligenceType
	UnitDisplay     int // [0..1]
	GameBalance     int // [0..4]
	Speed           int // [1..3]
}

func (o Options) IsPlayerControlled(side int) bool {
	if side == 0 {
		return o.AlliedCommander == 0
	}
	return o.GermanCommander == 0
}
func (o Options) Num() int {
	n := o.AlliedCommander + 2*o.GermanCommander
	if o.Intelligence == Limited {
		n += 56 - 4*(o.AlliedCommander*o.GermanCommander+o.AlliedCommander)
	}
	return n
}

