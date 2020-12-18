package lib

import "fmt"

type IntelligenceType struct{ i int }

func (i IntelligenceType) String() string {
	switch i {
	case Full:
		return "FULL"
	case Limited:
		return "LIMITED"
	}
	panic(fmt.Errorf("Unknown intelligence type: %d", i.i))
}
func (i IntelligenceType) Other() IntelligenceType {
	return IntelligenceType{1 - i.i}
}

var Full = IntelligenceType{0}
var Limited = IntelligenceType{1}

type CommanderType struct{ c int }

func (c CommanderType) Int() int { return c.c }
func (c CommanderType) String() string {
	switch c {
	case Player:
		return "PLAYER"
	case Computer:
		return "COMPUTER"
	}
	panic(fmt.Errorf("Unknown commander type: %d", c.c))
}
func (c CommanderType) Other() CommanderType {
	return CommanderType{1 - c.c}
}

var Player = CommanderType{0}
var Computer = CommanderType{1}

type UnitDisplayType int

func (u UnitDisplayType) String() string {
	switch u {
	case ShowAsSymbols:
		return "SYMBOLS"
	case ShowAsIcons:
		return "ICONS"
	}
	panic(fmt.Errorf("Unknown unit display type: %d", int(u)))
}

const (
	ShowAsSymbols UnitDisplayType = 0
	ShowAsIcons   UnitDisplayType = 1
)

type SpeedType int

func (s SpeedType) String() string {
	switch s {
	case Fast:
		return "FAST"
	case Medium:
		return "MEDIUM"
	case Slow:
		return "SLOW"
	}
	panic(fmt.Errorf("Unknown speed: %d", int(s)))
}
func (s SpeedType) DelayTicks() int {
	return 60 * int(s)
}
func (s SpeedType) Faster() SpeedType {
	switch s {
	case Fast:
		return Fast
	case Medium:
		return Fast
	case Slow:
		return Medium
	}
	panic(fmt.Errorf("Unknown speed: %d", int(s)))
}
func (s SpeedType) Slower() SpeedType {
	switch s {
	case Fast:
		return Medium
	case Medium:
		return Slow
	case Slow:
		return Slow
	}
	panic(fmt.Errorf("Unknown speed: %d", int(s)))
}

const (
	Fast   SpeedType = 1
	Medium SpeedType = 2
	Slow   SpeedType = 3
)

type Options struct {
	AlliedCommander CommanderType // [0..1]
	GermanCommander CommanderType // [0..1]
	Intelligence    IntelligenceType
	UnitDisplay     UnitDisplayType // [0..1]
	GameBalance     int             // [0..4]
	Speed           SpeedType       // [1..3]
}

func DefaultOptions() Options {
	return Options{
		AlliedCommander: Player,
		GermanCommander: Computer,
		Intelligence:    Limited,
		UnitDisplay:     ShowAsSymbols,
		GameBalance:     2,
		Speed:           Medium}
}

