package main

import "fmt"

import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/ebitenutil"
import "github.com/hajimehoshi/ebiten/inpututil"
import "github.com/pwiecz/command_series/data"

type IntelligenceType int

const (
	Full    IntelligenceType = 0
	Limited IntelligenceType = 1
)

type Options struct {
	AlliedCommander int
	GermanCommander int
	Intelligence    IntelligenceType
	GameBalance     int // [0..4]
}

func (o Options) IsPlayerControlled(side int) bool {
	if side == 0 {
		return o.AlliedCommander > 0
	}
	return o.GermanCommander > 0
}
func (o Options) Num() int {
	n := o.AlliedCommander + 2*o.GermanCommander
	if o.Intelligence == Limited {
		n += 56 - 4*(o.AlliedCommander*o.GermanCommander+o.AlliedCommander)
	}
	return n
}

type ShowMap struct {
	mainGame  *Game
	mapView   *MapView
	animation *Animation
	mapImage  *ebiten.Image
	options   Options
	dx, dy    int

	currentSpeed  int
	idleTicksLeft int
	isFrozen      bool
	unitIconView  bool
	playerSide    int

	orderedUnit *data.Unit

	gameState     *GameState
	commandBuffer *CommandBuffer

	sync    *MessageSync
	started bool
}

func NewShowMap(g *Game) *ShowMap {
	scenario := g.scenarios[g.selectedScenario]
	variant := g.variants[g.selectedVariant]
	for x := scenario.MinX - 1; x <= scenario.MaxX+1; x++ {
		g.terrainMap.SetTile(x, scenario.MinY-1, 12)
		g.terrainMap.SetTile(x, scenario.MaxY+1, 12)
	}
	for y := scenario.MinY; y <= scenario.MaxY; y++ {
		g.terrainMap.SetTile(scenario.MinX-1, y, 10)
		g.terrainMap.SetTile(scenario.MaxX+1, y, 12)
	}
	s := &ShowMap{
		mainGame:      g,
		dx:            0,
		dy:            0,
		currentSpeed:  2,
		idleTicksLeft: 60,
		commandBuffer: NewCommandBuffer(20),
		sync:          NewMessageSync()}
	s.options.AlliedCommander = 0
	s.options.GermanCommander = 0
	s.options.GameBalance = 2
	s.gameState = NewGameState(&scenario, &g.scenarioData, &variant, g.selectedVariant, g.units, &g.terrain, &g.terrainMap, &g.generic, &g.hexes, g.generals, s.options, s.sync)
	s.mapView = NewMapView(
		&g.terrainMap, scenario.MinX, scenario.MinY, scenario.MaxX, scenario.MaxY,
		&g.sprites.TerrainTiles, &g.sprites.UnitSymbolSprites, &g.sprites.UnitIconSprites,
		&g.icons.Sprites, &g.scenarioData.DaytimePalette, &g.scenarioData.NightPalette)
	s.unitIconView = true
	return s
}

func (s *ShowMap) screenCoordsToUnitCoords(screenX, screenY int) (x, y int) {
	return s.mapView.ToUnitCoords(
		screenX+s.dx*int(s.mapView.tileWidth), screenY+s.dy*int(s.mapView.tileHeight))
}

func (s *ShowMap) Update() error {
	if !s.started {
		go func() {
			if !s.sync.Wait() {
				return
			}
			if !s.gameState.Init() {
				return
			}
			for {
				if !s.gameState.Update() {
					return
				}
			}
		}()
		s.started = true
	}

	s.commandBuffer.Update()
	if s.animation != nil {
		s.animation.Update()
		if s.animation.Done() {
			s.animation = nil
		} else {
			return nil
		}
		// Do not handle key events just after finishing animation to let logic
		// update the state - e.g. mark the final location of the unit.
	} else {
		select {
		case cmd := <-s.commandBuffer.Commands:
			switch cmd {
			case Freeze:
				s.isFrozen = !s.isFrozen
				s.idleTicksLeft = 0
				if s.isFrozen {
					fmt.Println("FROZEN")
				} else {
					fmt.Println("UNFROZEN")
				}
			case StatusReport:
				fmt.Println(s.gameState.statusReport())
				s.idleTicksLeft = 60 * s.currentSpeed
			case DecreaseSpeed:
				s.idleTicksLeft = 60 * s.currentSpeed
				s.decreaseGameSpeed()
			case IncreaseSpeed:
				s.idleTicksLeft = 60 * s.currentSpeed
				s.increaseGameSpeed()
			case SwitchUnitDisplay:
				s.idleTicksLeft = 60 * s.currentSpeed
				s.unitIconView = !s.unitIconView
			case Quit:
				s.sync.Stop()
				return fmt.Errorf("QUIT")
			case Reserve:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Reserve)
				s.idleTicksLeft = 60 * s.currentSpeed
			case Defend:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Defend)
				s.idleTicksLeft = 60 * s.currentSpeed
			case Attack:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Attack)
				s.idleTicksLeft = 60 * s.currentSpeed
			case Move:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Move)
				s.idleTicksLeft = 60 * s.currentSpeed
			case SetObjective:
				s.trySetObjective(s.mapView.cursorX, s.mapView.cursorY)
				s.idleTicksLeft = 60 * s.currentSpeed
			case ScrollDown:
				s.idleTicksLeft = 60 * s.currentSpeed
				s.mapView.cursorY++
				s.applyCursorChange()
			case ScrollUp:
				s.idleTicksLeft = 60 * s.currentSpeed
				s.mapView.cursorY--
				s.applyCursorChange()
			case ScrollRight:
				s.idleTicksLeft = 60 * s.currentSpeed
				s.mapView.cursorX++
				s.applyCursorChange()
			case ScrollLeft:
				s.idleTicksLeft = 60 * s.currentSpeed
				s.mapView.cursorX--
				s.applyCursorChange()
			}
		default:
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mouseX, mouseY := ebiten.CursorPosition()
			x, y := s.screenCoordsToUnitCoords(mouseX, mouseY)
			s.mapView.cursorX = x / 2
			s.mapView.cursorY = y
			s.applyCursorChange()
			if unit, ok := s.gameState.FindUnit(x, y); ok {
				fmt.Println()
				fmt.Println(s.gameState.unitInfo(unit))
			} else {
				fmt.Println("NO UNIT")
			}
		}
	}
	if s.isFrozen {
		return nil
	}
	if s.gameState.isInitialized && s.idleTicksLeft > 0 {
		s.idleTicksLeft--
		return nil
	}
	for {
		update := s.sync.GetUpdate()
		if update == nil {
			break
		}
		switch message := update.(type) {
		case MessageFromUnit:
			unit := message.Unit()
			if unit.Side == s.playerSide {
				fmt.Printf("\nMESSAGE FROM ...\n%s %s:\n", unit.Name, s.mainGame.scenarioData.UnitTypes[unit.Type])
				fmt.Printf("'%s'\n", message.String())
				s.idleTicksLeft = 60 * s.currentSpeed
				break
			}
		case Reinforcements:
			if message.Sides[s.playerSide] {
				fmt.Println("\nREINFORCEMENTS!")
				s.idleTicksLeft = 100
				break
			}
		case GameOver:
			fmt.Printf("\n%s\n", message.Results)
			return fmt.Errorf("GAME OVER!")
		case UnitMove:
			if s.areUnitCoordsVisible(message.X0, message.Y0) || s.areUnitCoordsVisible(message.X1, message.Y1) {
				s.animation = NewUnitAnimation(s.mapView, message.Unit,
					message.X0, message.Y0, message.X1, message.Y1, 30)
				break
			}
		case SupplyTruckMove:
			if s.areUnitCoordsVisible(message.X0, message.Y0) || s.areUnitCoordsVisible(message.X1, message.Y1) {
				s.animation = NewIconAnimation(s.mapView, data.SupplyTruck,
					message.X0, message.Y0, message.X1, message.Y1, 4)
				break
			}
		default:
			return fmt.Errorf("Unknown message: %v", message)
		}
	}
	return nil
}

func (s *ShowMap) applyCursorChange() {
	scenario := s.mainGame.scenarios[s.mainGame.selectedScenario]
	s.mapView.cursorX = Clamp(s.mapView.cursorX, scenario.MinX, scenario.MaxX)
	s.mapView.cursorY = Clamp(s.mapView.cursorY, scenario.MinY, scenario.MaxY)
	if s.mapView.cursorX-scenario.MinX < s.dx {
		s.dx = s.mapView.cursorX - scenario.MinX
	}
	if s.mapView.cursorY-scenario.MinY < s.dy {
		s.dy = s.mapView.cursorY - scenario.MinY
	}
	if s.mapView.cursorX-scenario.MinX-s.dx >= int(320/s.mapView.tileWidth) {
		s.dx = s.mapView.cursorX - scenario.MinX - int(320/s.mapView.tileWidth) + 1
	}
	if s.mapView.cursorY-scenario.MinY-s.dy >= int(192/s.mapView.tileHeight) {
		s.dy = s.mapView.cursorY - scenario.MinY - int(192/s.mapView.tileHeight) + 1
	}
}
func (s *ShowMap) areUnitCoordsVisible(x, y int) bool {
	screenX, screenY := s.mapView.MapCoordsToScreenCoords(x/2, y)
	dx, dy := s.dx*int(s.mapView.tileWidth), s.dy*int(s.mapView.tileHeight)
	return image.Pt(int(screenX), int(screenY)).In(image.Rect(dx, dy, dx+320, dy+192))
}
func (s *ShowMap) tryGiveOrderAtMapCoords(x, y int, order data.OrderType) {
	if unit, ok := s.gameState.FindUnitAtMapCoords(x, y); ok {
		s.giveOrder(unit, order)
		s.orderedUnit = &unit
	} else {
		fmt.Println("NO FRIENDLY UNIT.")
	}
}
func (s *ShowMap) giveOrder(unit data.Unit, order data.OrderType) {
	unit.Order = order
	unit.State &= 223
	switch order {
	case data.Reserve:
		unit.ObjectiveX = 0
		fmt.Println("RESERVE")
	case data.Attack:
		unit.ObjectiveX = 0
		fmt.Println("ATTACKING")
	case data.Defend:
		fmt.Println("DEFENDING")
		unit.ObjectiveX, unit.ObjectiveY = unit.X, unit.Y
	case data.Move:
		fmt.Println("MOVE WHERE ?")
	}
	s.mainGame.units[unit.Side][unit.Index] = unit
}
func (s *ShowMap) trySetObjective(x, y int) {
	if s.orderedUnit == nil {
		fmt.Println("GIVE ORDERS FIRST!")
		return
	}
	unitX := 2*x + y%2
	s.setObjective(s.mainGame.units[s.orderedUnit.Side][s.orderedUnit.Index], unitX, y)

}
func (s *ShowMap) setObjective(unit data.Unit, x, y int) {
	unit.ObjectiveX, unit.ObjectiveY = x, y
	unit.State &= 223 // clean bit 5 (32)
	fmt.Println(s.gameState.unitInfo(unit))
	fmt.Println("OBJECTIVE HERE.")
	distance := Function15_distanceToObjective(unit)
	if distance > 0 {
		fmt.Println("DISTANCE:", distance*s.mainGame.scenarioData.HexSizeInMiles, "MILES.")
	}
	s.mainGame.units[unit.Side][unit.Index] = unit
	s.orderedUnit = nil
}
func (s *ShowMap) increaseGameSpeed() {
	s.changeGameSpeed(-1)
}
func (s *ShowMap) decreaseGameSpeed() {
	s.changeGameSpeed(1)
}
func (s *ShowMap) changeGameSpeed(delta int) {
	s.currentSpeed = Clamp(s.currentSpeed+delta, 1, 3)
	speedNames := []string{"FAST", "MEDIUM", "SLOW"}
	fmt.Printf("SPEED: %s\n", speedNames[s.currentSpeed-1])
}

func (s *ShowMap) dateTimeString() string {
	meridianString := "AM"
	if s.gameState.hour >= 12 {
		meridianString = "PM"
	}
	hour := Abs(s.gameState.hour - 12*((s.gameState.hour+11)/12-1))
	return fmt.Sprintf("  %02d:%02d %s %s, %d %d  %s", hour, s.gameState.minute, meridianString, s.mainGame.scenarioData.Months[s.gameState.month], s.gameState.day+1, s.gameState.year, s.mainGame.scenarioData.Weather[s.gameState.weather])
}

func (s *ShowMap) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(s.dx)*(-s.mapView.tileWidth), float64(s.dy)*(-s.mapView.tileHeight))

	s.mapView.SetIsNight(s.gameState.isNight)
	s.mapView.SetUseIcons(s.unitIconView)

	s.mapView.Draw(screen, opts)
	if s.animation != nil {
		s.animation.Draw(screen, opts)
	}

	ebitenutil.DebugPrint(screen, s.dateTimeString())
}
func (s *ShowMap) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 320, 192
}
