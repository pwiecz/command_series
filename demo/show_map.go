package main

import "fmt"
import "image"
import "strings"

import "github.com/hajimehoshi/ebiten"
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
	mainGame      *Game
	mapView       *MapView
	messageBox    *MessageBox
	topRect       *Rectangle
	separatorRect *Rectangle

	animation *Animation
	mapImage  *ebiten.Image
	options   Options

	currentSpeed  int
	idleTicksLeft int
	isFrozen      bool
	areUnitsHidden bool
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
		currentSpeed:  2,
		commandBuffer: NewCommandBuffer(20),
		sync:          NewMessageSync()}
	s.options.AlliedCommander = 1
	s.options.GermanCommander = 0
	s.options.GameBalance = 2
	s.gameState = NewGameState(&scenario, &g.scenarioData, &variant, g.selectedVariant, g.units, &g.terrain, &g.terrainMap, &g.generic, &g.hexes, g.generals, s.options, s.sync)
	s.mapView = NewMapView(
		&g.terrainMap, scenario.MinX, scenario.MinY, scenario.MaxX, scenario.MaxY,
		&g.sprites.TerrainTiles, &g.sprites.UnitSymbolSprites, &g.sprites.UnitIconSprites,
		&g.icons.Sprites, &g.scenarioData.DaytimePalette, &g.scenarioData.NightPalette,
		image.Pt(160, 19*8))
	s.messageBox = NewMessageBox(image.Pt(336, 48), g.sprites.GameFont)

	s.topRect = NewRectangle(image.Pt(336, 22))
	s.separatorRect = NewRectangle(image.Pt(336, 2))
	s.unitIconView = true
	return s
}

func (s *ShowMap) Update() error {
	if !s.started && !s.areUnitsHidden {
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
				s.showStatusReport()
				s.idleTicksLeft = 60 * s.currentSpeed
			case UnitInfo:
				s.showUnitInfo()
				s.idleTicksLeft = 60 * s.currentSpeed
			case CityInfo:
				s.showCityInfo()
				s.idleTicksLeft = 60 * s.currentSpeed
				case HideUnits:
				s.hideUnits()
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
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX, curY+1)
			case ScrollUp:
				s.idleTicksLeft = 60 * s.currentSpeed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX, curY-1)
			case ScrollRight:
				s.idleTicksLeft = 60 * s.currentSpeed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX+1, curY)
			case ScrollLeft:
				s.idleTicksLeft = 60 * s.currentSpeed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX-1, curY)
			}
		default:
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mouseX, mouseY := ebiten.CursorPosition()
			x, y := s.screenCoordsToUnitCoords(mouseX, mouseY)
			s.mapView.SetCursorPosition(x/2, y)
		}
	}
	if s.isFrozen || s.areUnitsHidden {
		return nil
	}
	if s.idleTicksLeft > 0 {
		s.idleTicksLeft--
		return nil
	}
loop:
	for {
		update := s.sync.GetUpdate()
		if update == nil {
			break loop
		}
		switch message := update.(type) {
		case Initialized:
			s.idleTicksLeft = 60
			break loop
		case MessageFromUnit:
			unit := message.Unit()
			if unit.Side == s.playerSide {
				for y := 0; y < 5; y++ {
					s.messageBox.ClearRow(y)
				}
				s.messageBox.Print("MESSAGE FROM ...", 2, 0, true)
				unitName := fmt.Sprintf("%s %s:", unit.Name, s.mainGame.scenarioData.UnitTypes[unit.Type])
				s.messageBox.Print(unitName, 2, 1, false)
				lines := strings.Split("\""+message.String()+"\"", "\n")
				for i, line := range lines {
					s.messageBox.Print(line, 2, 2+i, false)
				}
				s.idleTicksLeft = 60 * s.currentSpeed
				break loop
			}
		case Reinforcements:
			if message.Sides[s.playerSide] {
				fmt.Println("\nREINFORCEMENTS!")
				s.idleTicksLeft = 100
			}
			break loop
		case GameOver:
			fmt.Printf("\n%s\n", message.Results)
			return fmt.Errorf("GAME OVER!")
		case UnitMove:
			if s.mapView.AreMapCoordsVisible(message.X0, message.Y0) || s.mapView.AreMapCoordsVisible(message.X1, message.Y1) {
				s.animation = NewUnitAnimation(s.mapView, message.Unit,
					message.X0, message.Y0, message.X1, message.Y1, 30)
				break loop
			}
		case SupplyTruckMove:
			if s.mapView.AreMapCoordsVisible(message.X0, message.Y0) || s.mapView.AreMapCoordsVisible(message.X1, message.Y1) {
				s.animation = NewIconAnimation(s.mapView, data.SupplyTruck,
					message.X0, message.Y0, message.X1, message.Y1, 4)
				break loop
			}
		case WeatherForecast:
			s.messageBox.Clear()
			s.messageBox.Print(fmt.Sprintf("WEATHER FORECAST: %s", s.mainGame.scenarioData.Weather[message.Weather]), 2, 0, false)
		case SupplyDistributionStart:
			s.messageBox.Print(" SUPPLY DISTRIBUTION ", 2, 1, true)
		case SupplyDistributionEnd:
		case DailyUpdate:
			s.messageBox.Print(fmt.Sprintf("%d DAYS REMAINING.", message.DaysRemaining), 2, 2, false)
			s.messageBox.Print("SUPPLY LEVEL:", 2, 3, true)
			supplyLevels := []string{"CRITICAL", "SUFFICIENT", "AMPLE"}
			s.messageBox.Print(supplyLevels[message.SupplyLevel], 16, 3, false)
			s.idleTicksLeft = 60 * s.currentSpeed
			break loop
		default:
			return fmt.Errorf("Unknown message: %v", message)
		}
	}
	return nil
}

func (s *ShowMap) areUnitCoordsVisible(x, y int) bool {
	return s.mapView.AreMapCoordsVisible(x/2, y)
}
func (s *ShowMap) tryGiveOrderAtMapCoords(x, y int, order data.OrderType) {
	s.messageBox.Clear()
	if unit, ok := s.gameState.FindUnitAtMapCoords(x, y); ok && unit.Side == s.playerSide {
		s.giveOrder(unit, order)
		s.orderedUnit = &unit
	} else {
		s.messageBox.Print("NO FRIENDLY UNIT.", 2, 0, false)
	}
}
func (s *ShowMap) giveOrder(unit data.Unit, order data.OrderType) {
	unit.Order = order
	unit.State &= 223
	switch order {
	case data.Reserve:
		unit.ObjectiveX = 0
		s.messageBox.Print("RESERVE", 2, 0, false)
	case data.Attack:
		unit.ObjectiveX = 0
		s.messageBox.Print("ATTACKING", 2, 0, false)
	case data.Defend:
		unit.ObjectiveX, unit.ObjectiveY = unit.X, unit.Y
		s.messageBox.Print("DEFENDING", 2, 0, false)
	case data.Move:
		s.messageBox.Print("MOVE WHERE ?", 2, 0, false)
	}
	s.mainGame.units[unit.Side][unit.Index] = unit
}
func (s *ShowMap) trySetObjective(x, y int) {
	if s.orderedUnit == nil {
		s.messageBox.Clear()
		s.messageBox.Print("GIVE ORDERS FIRST!", 2, 0, false)
		return
	}
	unitX := 2*x + y%2
	s.setObjective(s.mainGame.units[s.orderedUnit.Side][s.orderedUnit.Index], unitX, y)

}
func (s *ShowMap) setObjective(unit data.Unit, x, y int) {
	unit.ObjectiveX, unit.ObjectiveY = x, y
	unit.State &= 223 // clean bit 5 (32)
	s.messageBox.Clear()
	s.messageBox.Print("WHO ", 2, 0, true)
	s.messageBox.Print(fmt.Sprintf("%s %s", unit.Name, s.mainGame.scenarioData.UnitTypes[unit.Type]), 7, 0, false)
	s.messageBox.Print("OBJECTIVE HERE.", 2, 1, false)
	distance := Function15_distanceToObjective(unit)
	if distance > 0 {
		s.messageBox.Print(fmt.Sprintf("DISTANCE: %d MILES.", distance*s.mainGame.scenarioData.HexSizeInMiles), 2, 2, false)
	}
	s.mainGame.units[unit.Side][unit.Index] = unit
	s.orderedUnit = nil
}
func (s *ShowMap) showUnitInfo() {
	if s.areUnitsHidden {
		return
	}
	cursorX, cursorY := s.mapView.GetCursorPosition()
	unit, ok := s.gameState.FindUnitAtMapCoords(cursorX, cursorY)
	if !ok {
		return
	}
	s.messageBox.Clear()
	if unit.Side != s.playerSide && unit.State&1 == 0 {
		s.messageBox.Print(" NO INFORMATION ", 2, 0, true)
		return
	}
	var nextRow int
	if unit.Side != s.playerSide {
		s.messageBox.Print(" ENEMY UNIT ", 2, 0, true)
		nextRow++
	}
	s.messageBox.Print("WHO ", 2, nextRow, true)
	s.messageBox.Print(fmt.Sprintf("%s %s", unit.Name, s.mainGame.scenarioData.UnitTypes[unit.Type]), 7, nextRow, false)
	nextRow++

	s.messageBox.Print("    ", 2, nextRow, true)
	var menStr string
	men := unit.MenCount
	if unit.Side != s.playerSide {
		men -= men % 10
	}
	if men > 0 {
		menStr = fmt.Sprintf("%d MEN, ", men*s.mainGame.scenarioData.MenMultiplier)
	}
	tanks := unit.EquipCount
	if unit.Side != s.playerSide {
		tanks -= tanks % 10
	}
	if tanks > 0 {
		menStr += fmt.Sprintf("%d %s, ", tanks*s.mainGame.scenarioData.TanksMultiplier, s.mainGame.scenarioData.Equipments[unit.Type])
	}
	s.messageBox.Print(menStr, 7, nextRow, false)
	nextRow++

	if unit.Side == s.playerSide {
		s.messageBox.Print("    ", 2, nextRow, true)
		supplyDays := unit.SupplyLevel / (s.mainGame.scenarioData.AvgDailySupplyUse + s.mainGame.scenarioData.Data163)
		supplyStr := fmt.Sprintf("%d DAYS SUPPLY.", supplyDays)
		if unit.State&8 != 0 {
			supplyStr += " (NO SUPPLY LINE!)"
		}
		s.messageBox.Print(supplyStr, 7, nextRow, false)
		nextRow++
	}

	s.messageBox.Print("FORM", 2, nextRow, true)
	formationStr := s.mainGame.scenarioData.Formations[unit.Formation]
	s.messageBox.Print(formationStr, 7, nextRow, false)
	if unit.Side != s.playerSide {
		return
	}
	s.messageBox.Print("EXP", 7+len(formationStr)+1, nextRow, true)
	expStr := s.mainGame.scenarioData.Experience[unit.Morale/27]
	s.messageBox.Print(expStr, 7+len(formationStr)+5, nextRow, false)
	s.messageBox.Print("EFF", 7+len(formationStr)+5+len(expStr)+1, nextRow, true)
	s.messageBox.Print(fmt.Sprintf("%d", 10*((256-unit.Fatigue)/25)), 7+len(formationStr)+5+len(expStr)+5, nextRow, false)
	nextRow++

	s.messageBox.Print("ORDR", 2, nextRow, true)
	orderStr := unit.Order.String()
	if unit.State&32 != 0 {
		orderStr += " (LOCAL COMMAND)"
	}
	s.messageBox.Print(orderStr, 7, nextRow, false)
}
func (s *ShowMap) showCityInfo() {
	s.messageBox.Clear()
	cursorX, cursorY := s.mapView.GetCursorPosition()
	city, ok := s.gameState.FindCityAtMapCoords(cursorX, cursorY)
	if !ok {
		s.messageBox.Print("NONE", 2, 0, false)
		return
	}
	s.messageBox.Print(city.Name, 2, 0, false)
	s.messageBox.Print(fmt.Sprintf("%d VICTORY POINTS, %s", city.VictoryPoints, s.mainGame.scenarioData.Sides[city.Owner]), 2, 1, false)
}
func (s *ShowMap) showStatusReport() {
	s.messageBox.Clear()
	s.messageBox.Print("STATUS REPORT", 2, 0, true)
	s.messageBox.Print(s.mainGame.scenarioData.Sides[0], 16, 0, false)
	s.messageBox.Print(s.mainGame.scenarioData.Sides[1], 26, 0, false)
	s.messageBox.Print(" TROOPS LOST ", 2, 1, true)
	menMultiplier, tanksMultiplier := s.mainGame.scenarioData.MenMultiplier, s.mainGame.scenarioData.TanksMultiplier
	s.messageBox.Print(fmt.Sprintf("%d", s.gameState.menLost[0]*menMultiplier), 16, 1, false)
	s.messageBox.Print(fmt.Sprintf("%d", s.gameState.menLost[1]*menMultiplier), 26, 1, false)
	s.messageBox.Print(" TANKS  LOST ", 2, 2, true)
	s.messageBox.Print(fmt.Sprintf("%d", s.gameState.tanksLost[0]*tanksMultiplier), 16, 2, false)
	s.messageBox.Print(fmt.Sprintf("%d", s.gameState.tanksLost[1]*tanksMultiplier), 26, 2, false)
	s.messageBox.Print(" CITIES HELD ", 2, 3, true)
	s.messageBox.Print(fmt.Sprintf("%d", s.gameState.citiesHeld[0]), 16, 3, false)
	s.messageBox.Print(fmt.Sprintf("%d", s.gameState.citiesHeld[1]), 26, 3, false)
	winningSide, advantage := s.gameState.winningSideAndAdvantage()
	advantageStrs := []string{"SLIGHT", "MARGINAL", "TACTICAL", "DECISIVE", "TOTAL"}
	winningSideStr := s.mainGame.scenarioData.Sides[winningSide]
	s.messageBox.Print(fmt.Sprintf("%s %s ADVANTAGE.", advantageStrs[advantage], winningSideStr), 2, 4, false)
}
func (s *ShowMap) hideUnits() {
	if s.areUnitsHidden {
		s.gameState.showAllVisibleUnits()
	} else {
		s.gameState.hideAllUnits()
	}
	s.areUnitsHidden = !s.areUnitsHidden
}
func (s *ShowMap) increaseGameSpeed() {
	s.changeGameSpeed(-1)
}
func (s *ShowMap) decreaseGameSpeed() {
	s.changeGameSpeed(1)
}
func (s *ShowMap) changeGameSpeed(delta int) {
	s.currentSpeed = Clamp(s.currentSpeed+delta, 1, 3)
	s.messageBox.Clear()
	speedNames := []string{"FAST", "MEDIUM", "SLOW"}
	s.messageBox.Print(fmt.Sprintf("SPEED: %s", speedNames[s.currentSpeed-1]), 2, 0, false)
}

func (s *ShowMap) dateTimeString() string {
	meridianString := "AM"
	if s.gameState.hour >= 12 {
		meridianString = "PM"
	}
	hour := Abs(s.gameState.hour - 12*((s.gameState.hour+11)/12-1))
	return fmt.Sprintf("%d:%02d %s %s, %d %d  %s", hour, s.gameState.minute, meridianString, s.mainGame.scenarioData.Months[s.gameState.month], s.gameState.day+1, s.gameState.year, s.mainGame.scenarioData.Weather[s.gameState.weather])
}

func (s *ShowMap) screenCoordsToUnitCoords(screenX, screenY int) (x, y int) {
	return s.mapView.ToUnitCoords((screenX-8)/2, screenY-72)
}

func (s *ShowMap) Draw(screen *ebiten.Image) {
	if !s.gameState.isNight {
		screen.Fill(data.RGBPalette[s.mainGame.scenarioData.DaytimePalette[2]])
		s.separatorRect.SetColor(int(s.mainGame.scenarioData.DaytimePalette[0]))
	} else {
		screen.Fill(data.RGBPalette[s.mainGame.scenarioData.NightPalette[2]])
		s.separatorRect.SetColor(int(s.mainGame.scenarioData.NightPalette[0]))
	}
	s.mapView.SetIsNight(s.gameState.isNight)
	s.mapView.SetUseIcons(s.unitIconView)

	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Scale(2, 1)
	opts.GeoM.Translate(8, 72)

	s.mapView.Draw(screen, &opts)
	if s.animation != nil {
		s.animation.Draw(screen, &opts)
	}

	opts.GeoM.Reset()
	opts.GeoM.Translate(0, 22)
	playerBaseColor := s.mainGame.scenarioData.SideColor[s.playerSide] * 16
	s.messageBox.SetRowBackground(0, playerBaseColor+12)
	s.messageBox.SetRowBackground(1, playerBaseColor+10)
	s.messageBox.SetRowBackground(2, playerBaseColor+12)
	s.messageBox.SetRowBackground(3, playerBaseColor+10)
	s.messageBox.SetRowBackground(4, playerBaseColor+12)
	s.messageBox.SetRowBackground(5, 30)
	s.messageBox.Print(s.dateTimeString(), 2, 5, false)
	s.messageBox.Draw(screen, &opts)

	opts.GeoM.Reset()
	s.topRect.SetColor(playerBaseColor + 10)
	s.topRect.Draw(screen, &opts)

	opts.GeoM.Translate(0, 70)
	s.separatorRect.Draw(screen, &opts)
}
func (s *ShowMap) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 336, 240
}
