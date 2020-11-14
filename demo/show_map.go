package main

import "fmt"

import "image/color"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/ebitenutil"
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
	mainGame                  *Game
	keyboardHandler           *KeyboardHandler
	mouseHandler              *MouseHandler
	mapView                   *MapView
	mapImage                  *ebiten.Image
	options                   Options
	dx, dy                    int

	currentSpeed              int
	idleTicksLeft             int

	isFrozen                  bool

	unitIconView              bool

	playerSide int

	gameState *GameState
}

func NewShowMap(g *Game) *ShowMap {
	scenario := g.scenarios[g.selectedScenario]
	variant := g.variants[g.selectedVariant]
	s := &ShowMap{
		mainGame:        g,
		keyboardHandler: NewKeyboardHandler(),
		mouseHandler:    NewMouseHandler(),
		dx:              0,
		dy:              0,
		currentSpeed:    1,
		idleTicksLeft:   60,
	}
	s.options.AlliedCommander = 0
	s.options.GermanCommander = 0
	s.options.GameBalance = 2
	s.gameState = NewGameState(&scenario, &s.mainGame.scenarioData, &variant, g.selectedVariant, s.mainGame.units, &s.mainGame.terrain, &s.mainGame.terrainMap, &s.mainGame.generic, &s.mainGame.hexes, s.mainGame.generals, s.options)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyF)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyQ)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyU)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeySlash)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyComma)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyPeriod)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyShift)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyUp)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyDown)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyLeft)
	s.keyboardHandler.AddKeyToHandle(ebiten.KeyRight)
	s.mouseHandler.AddButtonToHandle(ebiten.MouseButtonLeft)
	s.mapView = NewMapView(
		&g.terrainMap, scenario.MinX, scenario.MinY, scenario.MaxX, scenario.MaxY,
		&g.sprites.TerrainTiles, &g.sprites.UnitSymbolSprites, &g.sprites.UnitIconSprites,
		&g.scenarioData.DaytimePalette, &g.scenarioData.NightPalette)
	s.unitIconView = true
	s.mapView.dx = s.dx
	s.mapView.dy = s.dy

	s.gameState.everyHour()
	return s
}

func (s *ShowMap) screenCoordsToMapCoords(screenX, screenY int) (x, y int) {
	return s.mapView.ToMapCoords(screenX+s.dx*8, screenY+s.dy*8)
}

func (s *ShowMap) Update() error {
	s.keyboardHandler.Update()
	s.mouseHandler.Update()
	if s.keyboardHandler.IsKeyJustPressed(ebiten.KeySlash) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		fmt.Println(s.gameState.statusReport())
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyF) {
		s.isFrozen = !s.isFrozen
		s.idleTicksLeft = 0
		if s.isFrozen {
			fmt.Println("FROZEN")
		} else {
			fmt.Println("UNFROZEN")
		}
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyComma) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		s.decreaseGameSpeed()
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyPeriod) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		s.increaseGameSpeed()
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyU) {
		s.unitIconView = !s.unitIconView
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyQ) {
		return fmt.Errorf("QUIT")
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyDown) {
		s.dy++
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyUp) {
		s.dy--
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyRight) {
		s.dx++
	} else if s.keyboardHandler.IsKeyJustPressed(ebiten.KeyLeft) {
		s.dx--
	} else if s.mouseHandler.IsButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		x, y := s.screenCoordsToMapCoords(mouseX, mouseY)
		if unit, ok := s.gameState.FindUnit(x, y); ok {
			fmt.Println()
			fmt.Println(s.gameState.unitInfo(unit))
		} else {
			fmt.Println("NO UNIT")
		}
	}

	if s.isFrozen {
		return nil
	}
	if s.idleTicksLeft > 0 {
		s.idleTicksLeft--
		return nil
	}
	message, goOn := s.gameState.Update()
	if message != nil {
		unit := message.Unit()
		if unit.Side == s.playerSide {
			fmt.Printf("\nMESSAGE FROM ...\n%s %s:\n", unit.Name, s.mainGame.scenarioData.UnitTypes[unit.Type])
			fmt.Printf("'%s'\n", message.String())
			s.idleTicksLeft = 60 * s.currentSpeed
		}
	}
	if !goOn {
		return fmt.Errorf("GAME OVER!")
	}
	return nil
}

func (s *ShowMap) giveOrder(unit data.Unit, order data.OrderType) {
	unit.Order = order
	unit.State &= 223
	switch order {
	case data.Reserve:
		unit.ObjectiveX = 0
	case data.Attack:
		unit.ObjectiveX = 0
	case data.Defend:
		unit.ObjectiveX, unit.ObjectiveY = unit.X, unit.Y
	}
	s.mainGame.units[unit.Side][unit.Index] = unit
}
func (s *ShowMap) setObjective(unit data.Unit, x, y int) {
	unit.ObjectiveX, unit.ObjectiveY = x, y
	s.mainGame.units[unit.Side][unit.Index] = unit
}
func (s *ShowMap) increaseGameSpeed() {
	s.changeGameSpeed(-1)
}
func (s *ShowMap) decreaseGameSpeed() {
	s.changeGameSpeed(1)
}
func (s *ShowMap) changeGameSpeed(delta int) {
	s.currentSpeed = Clamp(s.currentSpeed+delta, 0, 2)
	speedNames := []string{"FAST", "MEDIUM", "SLOW"}
	fmt.Printf("SPEED: %s\n", speedNames[s.currentSpeed])
}

func (s *ShowMap) dateTimeString() string {
	meridianString := "AM"
	if s.gameState.hour >= 12 {
		meridianString = "PM"
	}
	hour := Abs(s.gameState.hour - 12*((s.gameState.hour+11)/12-1))
	return fmt.Sprintf("%02d:%02d %s %s, %d %d  %s", hour, s.gameState.minute, meridianString, s.mainGame.scenarioData.Months[s.gameState.month], s.gameState.day+1, s.gameState.year, s.mainGame.scenarioData.Weather[s.gameState.weather])
}

func (s *ShowMap) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(s.dx)*(-8), float64(s.dy)*(-8))

	s.mapView.SetIsNight(s.gameState.isNight)
	s.mapView.SetUseIcons(s.unitIconView)
	s.mapView.Draw(screen, opts)

	ebitenutil.DebugPrint(screen, s.dateTimeString())
}
func (s *ShowMap) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 320, 192
}
