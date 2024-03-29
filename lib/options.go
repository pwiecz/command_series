package lib

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Intelligence struct{ i int }

func (i Intelligence) Int() int { return i.i }
func (i Intelligence) String() string {
	switch i {
	case Full:
		return "FULL"
	case Limited:
		return "LIMITED"
	}
	panic(fmt.Errorf("unknown intelligence type: %d", i.i))
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
	panic(fmt.Errorf("unknown commander type: %d", c.c))
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
	panic(fmt.Errorf("unknown unit display type: %d", int(u)))
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
	panic(fmt.Errorf("unknown speed: %d", int(s)))
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
	panic(fmt.Errorf("unknown speed: %d", int(s)))
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
	panic(fmt.Errorf("unknown speed: %d", int(s)))
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

type CommanderFlags struct {
	PlayerControlled      [2]bool
	PlayerCanSeeUnits     [2]bool
	PlayerHasIntelligence [2]bool
}

func newCommanderFlags(o *Options) *CommanderFlags {
	commanderFlags := &CommanderFlags{}
	commanderFlags.PlayerControlled[0] = o.AlliedCommander == Player
	commanderFlags.PlayerControlled[1] = o.GermanCommander == Player
	if o.Intelligence == Limited {
		commanderFlags.PlayerCanSeeUnits[0] = o.AlliedCommander == Player || (o.AlliedCommander == Computer && o.GermanCommander == Computer)
		commanderFlags.PlayerCanSeeUnits[1] = o.AlliedCommander == Computer
		commanderFlags.PlayerHasIntelligence[0] = false
		commanderFlags.PlayerHasIntelligence[1] = false
	} else {
		commanderFlags.PlayerCanSeeUnits[0] = true
		commanderFlags.PlayerCanSeeUnits[1] = true
		commanderFlags.PlayerHasIntelligence[0] = true
		commanderFlags.PlayerHasIntelligence[1] = true
	}
	return commanderFlags
}
func (c *CommanderFlags) SwitchSides() {
	c.PlayerControlled[0], c.PlayerControlled[1] = c.PlayerControlled[1], c.PlayerControlled[0]
	c.PlayerCanSeeUnits[0], c.PlayerCanSeeUnits[1] = c.PlayerCanSeeUnits[1], c.PlayerCanSeeUnits[0]
	c.PlayerHasIntelligence[0], c.PlayerHasIntelligence[1] = c.PlayerHasIntelligence[1], c.PlayerHasIntelligence[0]
}

func (c *CommanderFlags) Serialize() (result uint8) {
	if !c.PlayerControlled[0] {
		result |= 0b1
	}
	if !c.PlayerControlled[1] {
		result |= 0b01
	}
	if !c.PlayerCanSeeUnits[0] {
		result |= 0b001
	}
	if !c.PlayerCanSeeUnits[1] {
		result |= 0b0001
	}
	if !c.PlayerHasIntelligence[0] {
		result |= 0b00001
	}
	if !c.PlayerHasIntelligence[1] {
		result |= 0b000001
	}
	return
}
func (c *CommanderFlags) Deserialize(value uint8) {
	c.PlayerControlled[0] = (value & 0b1) == 0
	c.PlayerControlled[1] = (value & 0b01) == 0
	c.PlayerCanSeeUnits[0] = (value & 0b001) == 0
	c.PlayerCanSeeUnits[1] = (value & 0b0001) == 0
	c.PlayerHasIntelligence[0] = (value & 0b00001) == 0
	c.PlayerHasIntelligence[1] = (value & 0b000001) == 0
}
