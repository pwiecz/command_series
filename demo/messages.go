package main

import "fmt"
import "github.com/pwiecz/command_series/data"

type Message interface {
	Unit() data.Unit
	String() string
}

type WeAreAttacking struct {
	unit           data.Unit
	enemy          data.Unit // todo: it may not to up to date, should it be pointer(index) of a unit?
	outcome        int
	formationNames []string
}

func (a WeAreAttacking) Unit() data.Unit { return a.unit }
func (a WeAreAttacking) String() string {
	losses := []string{"HEAVY", "MODERATE", "LIGHT", "VERY LIGHT"}
	return fmt.Sprintf("WE ARE ATTACKING.\nENEMY IS IN %s FORMATION.\nOUT LOSSES ARE %s.",
		a.formationNames[a.enemy.Formation], losses[Min(a.outcome/11, 3)])
}

type WeHaveMetStrongResistance struct {
	unit data.Unit
}

func (m WeHaveMetStrongResistance) Unit() data.Unit { return m.unit }
func (m WeHaveMetStrongResistance) String() string {
	return "WE HAVE MET STRONG RESISTANCE\nHEAVY LOSSES, ATTACK MUST BE HALTED."
}

type WeMustSurrender struct {
	unit data.Unit
}

func (m WeMustSurrender) Unit() data.Unit { return m.unit }
func (m WeMustSurrender) String() string {
	return "WE MUST SURRENDER"
}

type WeAreInContactWithEnemy struct {
	unit data.Unit
}

func (c WeAreInContactWithEnemy) Unit() data.Unit { return c.unit }
func (c WeAreInContactWithEnemy) String() string {
	return "WE ARE IN CONTACT WITH ENEMY FORCES."
}

type WeHaveCaptured struct {
	unit data.Unit
	city data.City
}

func (c WeHaveCaptured) Unit() data.Unit { return c.unit }
func (c WeHaveCaptured) String() string {
	return fmt.Sprintf("WE HAVE CAPTURED %s", c.city.Name)
}

type WeHaveReachedOurObjective struct {
	unit data.Unit
}

func (r WeHaveReachedOurObjective) Unit() data.Unit { return r.unit }
func (r WeHaveReachedOurObjective) String() string {
	return "WE HAVE REACHED OUR OBJECTIVE.\nAWAITING FURTHER ORDERS."
}

type WeHaveExhaustedSupplies struct {
	unit data.Unit
}

func (e WeHaveExhaustedSupplies) Unit() data.Unit { return e.unit }
func (e WeHaveExhaustedSupplies) String() string {
	return "WE HAVE EXHAUSTED OUR SUPPLIES."
}

type WeAreRetreating struct {
	unit data.Unit
}

func (r WeAreRetreating) Unit() data.Unit { return r.unit }
func (r WeAreRetreating) String() string {
	return "WE ARE RETREATING."
}

type WeHaveBeenOverrun struct {
	unit data.Unit
}

func (o WeHaveBeenOverrun) Unit() data.Unit { return o.unit }
func (o WeHaveBeenOverrun) String() string {
	return "WE HAVE BEEN OVERRUN."
}
