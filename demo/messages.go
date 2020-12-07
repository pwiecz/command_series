package main

import "fmt"
import "github.com/pwiecz/command_series/data"

type MessageFromUnit interface {
	Unit() data.Unit
	Icon() data.IconType
	String() string
}

type WeAreAttacking struct { // MSG = 1
	unit           data.Unit
	enemy          data.Unit
	outcome        int
	formationNames []string
}

func (a WeAreAttacking) Unit() data.Unit     { return a.unit }
func (a WeAreAttacking) Icon() data.IconType { return data.FightingUnit }
func (a WeAreAttacking) String() string {
	losses := []string{"HEAVY", "MODERATE", "LIGHT", "VERY LIGHT"}
	return fmt.Sprintf("WE ARE ATTACKING.\nENEMY IS IN %s FORMATION.\nOUR LOSSES ARE %s.",
		a.formationNames[a.enemy.Formation], losses[Min(a.outcome/11, 3)])
}

type WeHaveMetStrongResistance struct { // MSG = 2
	unit data.Unit
}

func (m WeHaveMetStrongResistance) Unit() data.Unit     { return m.unit }
func (a WeHaveMetStrongResistance) Icon() data.IconType { return data.UnitOnKnees }
func (m WeHaveMetStrongResistance) String() string {
	return "WE HAVE MET STRONG RESISTANCE\nHEAVY LOSSES, ATTACK MUST BE HALTED."
}

type WeMustSurrender struct { // MSG = 3
	unit data.Unit
}

func (m WeMustSurrender) Unit() data.Unit     { return m.unit }
func (a WeMustSurrender) Icon() data.IconType { return data.SurrenderingUnit }
func (m WeMustSurrender) String() string {
	return "WE MUST SURRENDER"
}

type WeAreInContactWithEnemy struct { // MSG = 4
	unit data.Unit
}

func (c WeAreInContactWithEnemy) Unit() data.Unit     { return c.unit }
func (c WeAreInContactWithEnemy) Icon() data.IconType { return data.ExclamationMark }
func (c WeAreInContactWithEnemy) String() string {
	return "WE ARE IN CONTACT WITH ENEMY FORCES."
}

type WeHaveCaptured struct { // MSG = 5
	unit data.Unit
	city data.City
}

func (c WeHaveCaptured) Unit() data.Unit     { return c.unit }
func (c WeHaveCaptured) Icon() data.IconType { return data.SmilingFace }
func (c WeHaveCaptured) String() string {
	return fmt.Sprintf("WE HAVE CAPTURED %s", c.city.Name)
}

type WeHaveReachedOurObjective struct { // MSG = 6
	unit data.Unit
}

func (r WeHaveReachedOurObjective) Unit() data.Unit     { return r.unit }
func (r WeHaveReachedOurObjective) Icon() data.IconType { return data.QuestionMark }
func (r WeHaveReachedOurObjective) String() string {
	return "WE HAVE REACHED OUR OBJECTIVE.\nAWAITING FURTHER ORDERS."
}

type WeHaveExhaustedSupplies struct { // MSG = 7
	unit data.Unit
}

func (e WeHaveExhaustedSupplies) Unit() data.Unit     { return e.unit }
func (e WeHaveExhaustedSupplies) Icon() data.IconType { return data.SupplyTruck }
func (e WeHaveExhaustedSupplies) String() string {
	return "WE HAVE EXHAUSTED OUR SUPPLIES."
}

type WeAreRetreating struct { // MSG = 9
	unit data.Unit
}

func (r WeAreRetreating) Unit() data.Unit     { return r.unit }
func (r WeAreRetreating) Icon() data.IconType { return data.MovingUnit }
func (r WeAreRetreating) String() string {
	return "WE ARE RETREATING."
}

type WeHaveBeenOverrun struct { // MSG = 10
	unit data.Unit
}

func (o WeHaveBeenOverrun) Unit() data.Unit     { return o.unit }
func (o WeHaveBeenOverrun) Icon() data.IconType { return data.UnitOnKnees }
func (o WeHaveBeenOverrun) String() string {
	return "WE HAVE BEEN OVERRUN."
}

type WeAreUnderFire struct { // MSG = 11
	unit data.Unit
}

func (u WeAreUnderFire) Unit() data.Unit     { return u.unit }
func (u WeAreUnderFire) Icon() data.IconType { return data.ExclamationMark }
func (u WeAreUnderFire) String() string {
	return "WE ARE UNDER FIRE!"
}

type Initialized struct{}

type Reinforcements struct{ Sides [2]bool }

type GameOver struct{ Results string }

type UnitMove struct {
	Unit           data.Unit
	X0, Y0, X1, Y1 int
}

type SupplyTruckMove struct {
	X0, Y0, X1, Y1 int
}

type WeatherForecast struct{ Weather int }

type SupplyDistributionStart struct{}
type SupplyDistributionEnd struct{}

type DailyUpdate struct {
	DaysRemaining int
	SupplyLevel   int
}
type TimeChanged struct{}
