package lib

import "encoding/binary"
import "fmt"
import "io"

type Intelligence struct{ i int }

func (i Intelligence) Int() int { return i.i }
func (i Intelligence) String() string {
	switch i {
	case Full:
		return "FULL"
	case Limited:
		return "LIMITED"
	}
	panic(fmt.Errorf("Unknown intelligence type: %d", i.i))
}
func (i Intelligence) Other() Intelligence {
	return Intelligence{1 - i.i}
}

var Full = Intelligence{0}
var Limited = Intelligence{1}

type Commander struct{ c int }

func (c Commander) Int() int { return c.c }
func (c Commander) String() string {
	switch c {
	case Player:
		return "PLAYER"
	case Computer:
		return "COMPUTER"
	}
	panic(fmt.Errorf("Unknown commander type: %d", c.c))
}
func (c Commander) Other() Commander {
	return Commander{1 - c.c}
}

var Player = Commander{0}
var Computer = Commander{1}

type UnitDisplay int

func (u UnitDisplay) String() string {
	switch u {
	case ShowAsSymbols:
		return "SYMBOLS"
	case ShowAsIcons:
		return "ICONS"
	}
	panic(fmt.Errorf("Unknown unit display type: %d", int(u)))
}

const (
	ShowAsSymbols UnitDisplay = 0
	ShowAsIcons   UnitDisplay = 1
)

type Speed int

func (s Speed) String() string {
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
func (s Speed) DelayTicks() int {
	return 60 * int(s)
}
func (s Speed) Faster() Speed {
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
func (s Speed) Slower() Speed {
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
	Fast   Speed = 1
	Medium Speed = 2
	Slow   Speed = 3
)

type Options struct {
	AlliedCommander Commander // [0..1]
	GermanCommander Commander // [0..1]
	Intelligence    Intelligence
	UnitDisplay     UnitDisplay // [0..1]
	GameBalance     int         // [0..4]
	Speed           Speed       // [1..3]
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

func (o Options) Write(writer io.Writer) error {
	if err := binary.Write(writer, binary.LittleEndian, uint8(o.AlliedCommander.Int())); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint8(o.GermanCommander.Int())); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint8(o.Intelligence.Int())); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint8(o.UnitDisplay)); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint8(o.GameBalance)); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, uint8(o.Speed)); err != nil {
		return err
	}
	return nil
}

func (o *Options) Read(reader io.Reader) error {
	var alliedCommander, germanCommander, intelligence, unitDisplay, gameBalance, speed uint8
	if err := binary.Read(reader, binary.LittleEndian, &alliedCommander); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &germanCommander); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &intelligence); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &unitDisplay); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &gameBalance); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &speed); err != nil {
		return err
	}
	o.AlliedCommander = Commander{int(alliedCommander)}
	o.GermanCommander = Commander{int(germanCommander)}
	o.Intelligence = Intelligence{int(intelligence)}
	o.UnitDisplay = UnitDisplay(unitDisplay)
	o.GameBalance = int(gameBalance)
	o.Speed = Speed(speed)
	return nil
}
