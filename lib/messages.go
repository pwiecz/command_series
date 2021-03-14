package lib

import "fmt"

type MessageFromUnit interface {
	Unit() Unit
	Icon() IconType
	String() string
}

type WeAreAttacking struct { // MSG = 1
	unit           Unit
	enemy          Unit
	outcome        int
	formationNames []string
}

func (a WeAreAttacking) Unit() Unit     { return a.unit }
func (a WeAreAttacking) Enemy() Unit    { return a.enemy }
func (a WeAreAttacking) Icon() IconType { return FightingUnit }
func (a WeAreAttacking) String() string {
	losses := []string{"HEAVY", "MODERATE", "LIGHT", "VERY LIGHT"}
	return fmt.Sprintf("WE ARE ATTACKING.\nENEMY IS IN %s FORMATION.\nOUR LOSSES ARE %s.",
		a.formationNames[a.enemy.Formation], losses[Min(a.outcome/11, 3)])
}
func (a WeAreAttacking) EnemyMessage() WeAreUnderFire { return WeAreUnderFire{a.enemy} }

type WeHaveMetStrongResistance struct { // MSG = 2
	unit Unit
}

func (m WeHaveMetStrongResistance) Unit() Unit     { return m.unit }
func (a WeHaveMetStrongResistance) Icon() IconType { return UnitOnKnees }
func (m WeHaveMetStrongResistance) String() string {
	return "WE HAVE MET STRONG RESISTANCE\nHEAVY LOSSES, ATTACK MUST BE HALTED."
}

type WeMustSurrender struct { // MSG = 3
	unit Unit
}

func (m WeMustSurrender) Unit() Unit     { return m.unit }
func (a WeMustSurrender) Icon() IconType { return SurrenderingUnit }
func (m WeMustSurrender) String() string {
	return "WE MUST SURRENDER"
}

type WeAreInContactWithEnemy struct { // MSG = 4
	unit Unit
}

func (c WeAreInContactWithEnemy) Unit() Unit     { return c.unit }
func (c WeAreInContactWithEnemy) Icon() IconType { return ExclamationMark }
func (c WeAreInContactWithEnemy) String() string {
	return "WE ARE IN CONTACT WITH ENEMY FORCES."
}

type WeHaveCaptured struct { // MSG = 5
	unit Unit
	city City
}

func (c WeHaveCaptured) Unit() Unit     { return c.unit }
func (c WeHaveCaptured) Icon() IconType { return SmilingFace }
func (c WeHaveCaptured) String() string {
	return fmt.Sprintf("WE HAVE CAPTURED %s", c.city.Name)
}

type WeHaveReachedOurObjective struct { // MSG = 6
	unit Unit
}

func (r WeHaveReachedOurObjective) Unit() Unit     { return r.unit }
func (r WeHaveReachedOurObjective) Icon() IconType { return QuestionMark }
func (r WeHaveReachedOurObjective) String() string {
	return "WE HAVE REACHED OUR OBJECTIVE.\nAWAITING FURTHER ORDERS."
}

type WeHaveExhaustedSupplies struct { // MSG = 7
	unit Unit
}

func (e WeHaveExhaustedSupplies) Unit() Unit     { return e.unit }
func (e WeHaveExhaustedSupplies) Icon() IconType { return SupplyTruck }
func (e WeHaveExhaustedSupplies) String() string {
	return "WE HAVE EXHAUSTED OUR SUPPLIES."
}

type WeAreRetreating struct { // MSG = 9
	unit Unit
}

func (r WeAreRetreating) Unit() Unit     { return r.unit }
func (r WeAreRetreating) Icon() IconType { return MovingUnit }
func (r WeAreRetreating) String() string {
	return "WE ARE RETREATING."
}

type WeHaveBeenOverrun struct { // MSG = 10
	unit Unit
}

func (o WeHaveBeenOverrun) Unit() Unit     { return o.unit }
func (o WeHaveBeenOverrun) Icon() IconType { return UnitOnKnees }
func (o WeHaveBeenOverrun) String() string {
	return "WE HAVE BEEN OVERRUN."
}

type WeAreUnderFire struct { // MSG = 11
	unit Unit
}

func (u WeAreUnderFire) Unit() Unit     { return u.unit }
func (u WeAreUnderFire) Icon() IconType { return ExclamationMark }
func (u WeAreUnderFire) String() string {
	return "WE ARE UNDER FIRE!"
}

type Initialized struct{}

type Reinforcements struct{ Sides [2]bool }

type GameOver struct{ Results string }

type UnitAttack struct {
	XY      UnitCoords
	Outcome int
}

type UnitMove struct {
	Unit     Unit
	XY0, XY1 MapCoords
}

type SupplyTruckMove struct {
	XY0, XY1 MapCoords
}

type WeatherForecast struct{ Weather int }

type SupplyDistributionStart struct{}
type SupplyDistributionEnd struct{}

type DailyUpdate struct {
	DaysRemaining int
	SupplyLevel   int
}
type TimeChanged struct{}
