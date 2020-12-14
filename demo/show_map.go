package main

import "fmt"
import "image"
import "strings"
import "math/rand"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/inpututil"

import "github.com/pwiecz/command_series/data"

type ShowMap struct {
	scenarioData *data.ScenarioData
	gameData     *data.GameData
	options      data.Options
	audioPlayer  *AudioPlayer
	onGameOver   func(int, int, int)

	mapView       *MapView
	topRect       *Rectangle
	messageBox    *MessageBox
	statusBar     *MessageBox
	separatorRect *Rectangle

	flashback *Flashback
	animation *Animation

	idleTicksLeft  int
	isFrozen       bool
	areUnitsHidden bool
	playerSide     int

	orderedUnit *data.Unit

	gameState     *data.GameState
	commandBuffer *CommandBuffer

	sync    *data.MessageSync
	started bool

	overviewMap *OverviewMap

	lastMessageFromUnit data.MessageFromUnit

	gameOver bool
}

func NewShowMap(g *Game, options data.Options, audioPlayer *AudioPlayer, onGameOver func(int, int, int)) *ShowMap {
	scenario := &g.gameData.Scenarios[g.selectedScenario]
	for x := scenario.MinX - 1; x <= scenario.MaxX+1; x++ {
		g.gameData.Map.SetTile(x, scenario.MinY-1, 12)
		g.gameData.Map.SetTile(x, scenario.MaxY+1, 12)
	}
	for y := scenario.MinY; y <= scenario.MaxY; y++ {
		g.gameData.Map.SetTile(scenario.MinX-1, y, 10)
		g.gameData.Map.SetTile(scenario.MaxX+1, y, 12)
	}
	s := &ShowMap{
		gameData:      g.gameData,
		scenarioData:  g.scenarioData,
		options:       options,
		audioPlayer:   audioPlayer,
		playerSide:    options.AlliedCommander,
		commandBuffer: NewCommandBuffer(20),
		sync:          data.NewMessageSync(),
		onGameOver:    onGameOver}
	rnd := rand.New(rand.NewSource(1))
	s.gameState = data.NewGameState(rnd, g.gameData, g.scenarioData, g.selectedScenario, g.selectedVariant, s.options, s.sync)
	s.mapView = NewMapView(
		&g.gameData.Map, scenario.MinX, scenario.MinY, scenario.MaxX, scenario.MaxY,
		&g.gameData.Sprites.TerrainTiles,
		&g.gameData.Sprites.UnitSymbolSprites, &g.gameData.Sprites.UnitIconSprites,
		&g.gameData.Icons.Sprites, &g.scenarioData.Data.DaytimePalette, &g.scenarioData.Data.NightPalette,
		image.Pt(160, 19*8))
	s.messageBox = NewMessageBox(image.Pt(336, 40), g.gameData.Sprites.GameFont)
	s.messageBox.Print("PREPARE FOR BATTLE!", 12, 1, false)
	s.statusBar = NewMessageBox(image.Pt(376, 8), g.gameData.Sprites.GameFont)
	s.statusBar.SetTextColor(16)
	s.statusBar.SetRowBackground(0, 30)

	s.topRect = NewRectangle(image.Pt(336, 22))
	s.separatorRect = NewRectangle(image.Pt(336, 2))
	return s
}

func (s *ShowMap) Update() error {
	if s.overviewMap != nil {
		for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
			if k == ebiten.KeyAlt || k == ebiten.KeyControl || k == ebiten.KeyShift || k == ebiten.KeySuper {
				continue
			}
			if inpututil.IsKeyJustPressed(k) {
				s.overviewMap = nil
				if s.areUnitsHidden {
					s.hideUnits()
				}
				break
			}
		}
		if s.overviewMap != nil {
			s.overviewMap.Update()
		}
		return nil
	}
	if s.flashback != nil {
		if s.flashback.Update() != nil {
			s.flashback = nil
			s.messageBox.Clear()
			if s.areUnitsHidden {
				s.hideUnits()
			}
		}
		return nil
	}
	if !s.started && !s.areUnitsHidden {
		s.idleTicksLeft = 100
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
				if s.gameOver {
					break
				}
				s.isFrozen = !s.isFrozen
				s.idleTicksLeft = 0
				s.statusBar.Clear()
				if s.isFrozen {
					s.statusBar.Print("FROZEN", 2, 0, false)
				} else {
					s.statusBar.Print("UNFROZEN", 2, 0, false)
				}
			case StatusReport:
				if !s.gameOver {
					s.showStatusReport()
					s.idleTicksLeft = 60 * s.options.Speed
				} else {
					result, balance, rank := s.gameState.FinalResults()
					s.onGameOver(result, balance, rank)
				}
			case UnitInfo:
				s.showUnitInfo()
				s.idleTicksLeft = 60 * s.options.Speed
			case GeneralInfo:
				s.showGeneralInfo()
				s.idleTicksLeft = 60 * s.options.Speed
			case CityInfo:
				s.showCityInfo()
				s.idleTicksLeft = 60 * s.options.Speed
			case HideUnits:
				s.hideUnits()
				s.idleTicksLeft = 60 * s.options.Speed
			case ShowOverviewMap:
				s.showOverviewMap()
				s.idleTicksLeft = 60 * s.options.Speed
			case ShowFlashback:
				s.showFlashback()
				s.idleTicksLeft = 60 * s.options.Speed
			case Who:
				s.showLastMessageUnit()
				s.idleTicksLeft = 60 * s.options.Speed
			case DecreaseSpeed:
				s.idleTicksLeft = 60 * s.options.Speed
				s.decreaseGameSpeed()
			case IncreaseSpeed:
				s.idleTicksLeft = 60 * s.options.Speed
				s.increaseGameSpeed()
			case SwitchUnitDisplay:
				s.idleTicksLeft = 60 * s.options.Speed
				s.options.UnitDisplay = 1 - s.options.UnitDisplay
			case SwitchSides:
				s.playerSide = 1 - s.playerSide
				s.mapView.HideIcon()
				s.messageBox.Clear()
				s.messageBox.Print(s.scenarioData.Data.Sides[s.playerSide]+" PLAYER:", 2, 0, false)
				s.messageBox.Print("PRESS \"T\" TO CONTINUE", 2, 1, false)
				if !s.areUnitsHidden {
					s.hideUnits()
				}
				s.idleTicksLeft = 60 * s.options.Speed
			case Quit:
				s.sync.Stop()
				return fmt.Errorf("QUIT")
			case Reserve:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Reserve)
				s.idleTicksLeft = 60 * s.options.Speed
			case Defend:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Defend)
				s.idleTicksLeft = 60 * s.options.Speed
			case Attack:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Attack)
				s.idleTicksLeft = 60 * s.options.Speed
			case Move:
				s.tryGiveOrderAtMapCoords(s.mapView.cursorX, s.mapView.cursorY, data.Move)
				s.idleTicksLeft = 60 * s.options.Speed
			case SetObjective:
				s.trySetObjective(s.mapView.cursorX, s.mapView.cursorY)
				s.idleTicksLeft = 60 * s.options.Speed
			case ScrollDown:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX, curY+1)
			case ScrollDownFast:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX, curY+2)
			case ScrollUp:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX, curY-1)
			case ScrollUpFast:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX, curY-2)
			case ScrollRight:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX+1, curY)
			case ScrollRightFast:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX+2, curY)
			case ScrollLeft:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX-1, curY)
			case ScrollLeftFast:
				s.idleTicksLeft = 60 * s.options.Speed
				curX, curY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(curX-2, curY)
			}
		default:
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mouseX, mouseY := ebiten.CursorPosition()
			x, y := s.screenCoordsToUnitCoords(mouseX, mouseY)
			s.mapView.SetCursorPosition(x/2, y)
		}
	}
	if s.isFrozen || s.areUnitsHidden || s.gameOver {
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
		case data.Initialized:
			s.idleTicksLeft = 60
			s.statusBar.Clear()
			s.statusBar.Print(s.dateTimeString(), 2, 0, false)
			break loop
		case data.MessageFromUnit:
			unit := message.Unit()
			if unit.Side == s.playerSide {
				s.showMessageFromUnit(message)
				break loop
			} else if s.gameData.Game == data.Conflict {
				if msg, ok := message.(data.WeAreAttacking); ok {
					s.showMessageFromUnit(data.NewWeAreUnderFire(msg.Enemy()))
					break loop
				}
			}
		case data.Reinforcements:
			if message.Sides[s.playerSide] {
				s.messageBox.Clear()
				s.messageBox.Print("REINFORCEMENTS!", 2, 1, false)
				s.idleTicksLeft = 100
			}
			break loop
		case data.GameOver:
			s.gameOver = true
			s.showStatusReport()
			s.statusBar.Print("GAME OVER, PRESS '?' FOR RESULTS.", 2, 0, false)
			s.sync.Stop()
			break loop
		case data.UnitMove:
			if s.mapView.AreMapCoordsVisible(message.X0, message.Y0) || s.mapView.AreMapCoordsVisible(message.X1, message.Y1) {
				s.animation = NewUnitAnimation(s.mapView /*s.audioPlayer*/, nil,
					message.Unit, message.X0, message.Y0, message.X1, message.Y1, 30)
				break loop
			}
		case data.SupplyTruckMove:
			if s.mapView.AreMapCoordsVisible(message.X0, message.Y0) || s.mapView.AreMapCoordsVisible(message.X1, message.Y1) {
				s.animation = NewIconAnimation(s.mapView, data.SupplyTruck,
					message.X0, message.Y0, message.X1, message.Y1, 3)
				break loop
			}
		case data.WeatherForecast:
			s.messageBox.Clear()
			s.messageBox.Print(fmt.Sprintf("WEATHER FORECAST: %s", s.scenarioData.Data.Weather[message.Weather]), 2, 0, false)
		case data.SupplyDistributionStart:
			s.mapView.HideIcon()
			s.messageBox.Print(" SUPPLY DISTRIBUTION ", 2, 1, true)
		case data.SupplyDistributionEnd:
		case data.DailyUpdate:
			s.messageBox.Print(fmt.Sprintf("%d DAYS REMAINING.", message.DaysRemaining), 2, 2, false)
			s.messageBox.Print("SUPPLY LEVEL:", 2, 3, true)
			supplyLevels := []string{"CRITICAL", "SUFFICIENT", "AMPLE"}
			s.messageBox.Print(supplyLevels[message.SupplyLevel], 16, 3, false)
			s.idleTicksLeft = 60 * s.options.Speed
			break loop
		case data.TimeChanged:
			s.statusBar.Clear()
			s.statusBar.Print(s.dateTimeString(), 2, 0, false)
			if s.gameState.Hour() == 18 && s.gameState.Minute() == 0 {
				s.showStatusReport()
				s.idleTicksLeft = 60 * s.options.Speed
			}
		default:
			return fmt.Errorf("Unknown message: %v", message)
		}
	}
	return nil
}

func (s *ShowMap) showMessageFromUnit(message data.MessageFromUnit) {
	for y := 0; y < 5; y++ {
		s.messageBox.ClearRow(y)
	}
	s.messageBox.Print("MESSAGE FROM ...", 2, 0, true)
	unit := message.Unit()
	unitName := fmt.Sprintf("%s:", unit.String())
	s.messageBox.Print(unitName, 2, 1, false)
	lines := strings.Split("\""+message.String()+"\"", "\n")
	for i, line := range lines {
		s.messageBox.Print(line, 2, 2+i, false)
	}
	s.mapView.ShowIcon(message.Icon(), unit.X/2, unit.Y)
	s.idleTicksLeft = 60 * s.options.Speed
	s.lastMessageFromUnit = message
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
	unit.HasLocalCommand = false
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
	s.scenarioData.Units[unit.Side][unit.Index] = unit
}
func (s *ShowMap) trySetObjective(x, y int) {
	if s.orderedUnit == nil {
		s.messageBox.Clear()
		s.messageBox.Print("GIVE ORDERS FIRST!", 2, 0, false)
		return
	}
	unitX := 2*x + y%2
	s.setObjective(s.scenarioData.Units[s.orderedUnit.Side][s.orderedUnit.Index], unitX, y)

}
func (s *ShowMap) setObjective(unit data.Unit, x, y int) {
	unit.ObjectiveX, unit.ObjectiveY = x, y
	unit.HasLocalCommand = false
	s.messageBox.Clear()
	s.messageBox.Print("WHO ", 2, 0, true)
	s.messageBox.Print(unit.String(), 7, 0, false)
	s.messageBox.Print("OBJECTIVE HERE.", 2, 1, false)
	distance := data.Function15_distanceToObjective(unit)
	if distance > 0 {
		s.messageBox.Print(fmt.Sprintf("DISTANCE: %d MILES.", distance*s.scenarioData.Data.HexSizeInMiles), 2, 2, false)
	}
	s.scenarioData.Units[unit.Side][unit.Index] = unit
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
	if unit.Side != s.playerSide && !unit.InContactWithEnemy {
		s.messageBox.Print(" NO INFORMATION ", 2, 0, true)
		return
	}
	var nextRow int
	if unit.Side != s.playerSide {
		s.messageBox.Print(" ENEMY UNIT ", 2, 0, true)
		nextRow++
	}
	s.messageBox.Print("WHO ", 2, nextRow, true)
	s.messageBox.Print(unit.String(), 7, nextRow, false)
	nextRow++

	s.messageBox.Print("    ", 2, nextRow, true)
	var menStr string
	men := unit.MenCount
	if unit.Side != s.playerSide {
		men -= men % 10
	}
	if men > 0 {
		menStr = fmt.Sprintf("%d MEN, ", men*s.scenarioData.Data.MenMultiplier)
	}
	tanks := unit.EquipCount
	if unit.Side != s.playerSide {
		tanks -= tanks % 10
	}
	if tanks > 0 {
		menStr += fmt.Sprintf("%d %s, ", tanks*s.scenarioData.Data.TanksMultiplier, s.scenarioData.Data.Equipments[unit.Type])
	}
	s.messageBox.Print(menStr, 7, nextRow, false)
	nextRow++

	if unit.Side == s.playerSide {
		s.messageBox.Print("    ", 2, nextRow, true)
		supplyDays := unit.SupplyLevel / (s.scenarioData.Data.AvgDailySupplyUse + s.scenarioData.Data.Data163)
		if s.gameData.Game != data.Crusade {
			supplyDays /= 2
		}
		supplyStr := fmt.Sprintf("%d DAYS SUPPLY.", supplyDays)
		if !unit.HasSupplyLine {
			supplyStr += " (NO SUPPLY LINE!)"
		}
		s.messageBox.Print(supplyStr, 7, nextRow, false)
		nextRow++
	}

	s.messageBox.Print("FORM", 2, nextRow, true)
	formationStr := s.scenarioData.Data.Formations[unit.Formation]
	s.messageBox.Print(formationStr, 7, nextRow, false)
	if unit.Side != s.playerSide {
		return
	}
	s.messageBox.Print("EXP", 7+len(formationStr)+1, nextRow, true)
	expStr := s.scenarioData.Data.Experience[unit.Morale/27]
	s.messageBox.Print(expStr, 7+len(formationStr)+5, nextRow, false)
	s.messageBox.Print("EFF", 7+len(formationStr)+5+len(expStr)+1, nextRow, true)
	s.messageBox.Print(fmt.Sprintf("%d", 10*((256-unit.Fatigue)/25)), 7+len(formationStr)+5+len(expStr)+5, nextRow, false)
	nextRow++

	s.messageBox.Print("ORDR", 2, nextRow, true)
	orderStr := unit.Order.String()
	if unit.HasLocalCommand {
		orderStr += " (LOCAL COMMAND)"
	}
	s.messageBox.Print(orderStr, 7, nextRow, false)
}
func numberToGeneralRating(num int) string {
	if num < 10 {
		return "POOR"
	}
	ratings := []string{"FAIR", "GOOD", "EXCELLNT"}
	return ratings[(num-10)/2]
}
func (s *ShowMap) showGeneralInfo() {
	if s.areUnitsHidden {
		return
	}
	cursorX, cursorY := s.mapView.GetCursorPosition()
	unit, ok := s.gameState.FindUnitAtMapCoords(cursorX, cursorY)
	if !ok {
		return
	}
	s.messageBox.Clear()
	if unit.Side != s.playerSide {
		s.messageBox.Print(" NO INFORMATION ", 2, 0, true)
		return
	}
	general := unit.General
	s.messageBox.Print("GENERAL ", 2, 0, true)
	s.messageBox.Print(general.Name, 11, 0, false)
	s.messageBox.Print("("+s.scenarioData.Data.Sides[unit.Side]+")", 23, 0, false)
	s.messageBox.Print("ATTACK  ", 2, 1, true)
	s.messageBox.Print(numberToGeneralRating(general.Attack), 11, 1, false)
	s.messageBox.Print("DEFEND  ", 2, 2, true)
	s.messageBox.Print(numberToGeneralRating(general.Defence), 11, 2, false)
	s.messageBox.Print("MOVEMENT", 2, 3, true)
	s.messageBox.Print(numberToGeneralRating(general.Movement), 11, 3, false)
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
	s.messageBox.Print(fmt.Sprintf("%d VICTORY POINTS, %s", city.VictoryPoints, s.scenarioData.Data.Sides[city.Owner]), 2, 1, false)
}
func (s *ShowMap) showStatusReport() {
	s.messageBox.Clear()
	if s.gameData.Game != data.Conflict {
		s.messageBox.Print("STATUS REPORT", 2, 0, true)
		s.messageBox.Print(s.scenarioData.Data.Sides[0], 16, 0, false)
		s.messageBox.Print(s.scenarioData.Data.Sides[1], 26, 0, false)
	} else {
		s.messageBox.Print(" STATUS REPORT ", 2, 0, true)
		s.messageBox.Print(s.scenarioData.Data.Sides[0], 19, 0, false)
		s.messageBox.Print(s.scenarioData.Data.Sides[1], 29, 0, false)
	}
	if s.gameData.Game != data.Conflict {
		s.messageBox.Print(" TROOPS LOST ", 2, 1, true)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.MenLost(0)), 16, 1, false)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.MenLost(1)), 26, 1, false)
	} else {
		s.messageBox.Print(" CASUALTIES    ", 2, 1, true)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.MenLost(0)), 19, 1, false)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.MenLost(1)), 29, 1, false)
	}
	if s.gameData.Game != data.Conflict {
		s.messageBox.Print(" TANKS  LOST ", 2, 2, true)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.TanksLost(0)), 16, 2, false)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.TanksLost(1)), 26, 2, false)
	} else {
		s.messageBox.Print(" MATERIEL      ", 2, 2, true)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.TanksLost(0)), 19, 2, false)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.TanksLost(1)), 29, 2, false)
	}
	if s.gameData.Game != data.Conflict {
		s.messageBox.Print(" CITIES HELD ", 2, 3, true)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.CitiesHeld(0)), 16, 3, false)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.CitiesHeld(1)), 26, 3, false)
	} else {
		s.messageBox.Print(" TERRITORY     ", 2, 3, true)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.CitiesHeld(0)), 19, 3, false)
		s.messageBox.Print(fmt.Sprintf("%d", s.gameState.CitiesHeld(1)), 29, 3, false)
	}
	winningSide, advantage := s.gameState.WinningSideAndAdvantage()
	advantageStrs := []string{"SLIGHT", "MARGINAL", "TACTICAL", "DECISIVE", "TOTAL"}
	winningSideStr := s.scenarioData.Data.Sides[winningSide]
	s.messageBox.Print(fmt.Sprintf("%s %s ADVANTAGE.", advantageStrs[advantage], winningSideStr), 2, 4, false)
}
func (s *ShowMap) hideUnits() {
	if s.areUnitsHidden {
		s.gameState.ShowAllVisibleUnits()
	} else {
		s.gameState.HideAllUnits()
	}
	s.areUnitsHidden = !s.areUnitsHidden
}
func (s *ShowMap) showOverviewMap() {
	if !s.areUnitsHidden {
		s.hideUnits()
	}
	s.overviewMap = NewOverviewMap(&s.gameData.Map, &s.scenarioData.Units, &s.gameData.Generic, &s.scenarioData.Data, &s.options)
}
func (s *ShowMap) showFlashback() {
	if !s.areUnitsHidden {
		s.hideUnits()
	}
	s.flashback = NewFlashback(s.mapView, s.messageBox, &s.gameData.Map, s.gameState.FlashbackUnits())
}
func (s *ShowMap) showLastMessageUnit() {
	if s.lastMessageFromUnit == nil {
		return
	}
	messageUnit := s.lastMessageFromUnit.Unit()
	s.mapView.SetCursorPosition(messageUnit.X/2, messageUnit.Y)
	s.mapView.ShowIcon(s.lastMessageFromUnit.Icon(), messageUnit.X/2, messageUnit.Y)
}
func (s *ShowMap) increaseGameSpeed() {
	s.changeGameSpeed(-1)
}
func (s *ShowMap) decreaseGameSpeed() {
	s.changeGameSpeed(1)
}
func (s *ShowMap) changeGameSpeed(delta int) {
	s.options.Speed = data.Clamp(s.options.Speed+delta, 1, 3)
	s.messageBox.Clear()
	speedNames := []string{"FAST", "MEDIUM", "SLOW"}
	s.messageBox.Print(fmt.Sprintf("SPEED: %s", speedNames[s.options.Speed-1]), 2, 0, false)
}

func (s *ShowMap) dateTimeString() string {
	meridianString := "AM"
	if s.gameState.Hour() >= 12 {
		meridianString = "PM"
	}
	hour := s.gameState.Hour() % 12
	if hour == 0 {
		hour = 12
	}
	return fmt.Sprintf("%d:%02d %s %s, %d %d  %s", hour, s.gameState.Minute(), meridianString, s.gameState.Month(), s.gameState.Day()+1, s.gameState.Year(), s.gameState.Weather())
}

func (s *ShowMap) screenCoordsToUnitCoords(screenX, screenY int) (x, y int) {
	return s.mapView.ToUnitCoords((screenX-8)/2, screenY-72)
}

func (s *ShowMap) Draw(screen *ebiten.Image) {
	if s.overviewMap != nil {
		screen.Fill(data.RGBPalette[8])
		opts := ebiten.DrawImageOptions{}
		opts.GeoM.Scale(4, 2)
		opts.GeoM.Translate(float64(336-4*s.gameData.Map.Width)/2, float64(240-2*s.gameData.Map.Height)/2)
		s.overviewMap.Draw(screen, &opts)
		return
	}
	if !s.gameState.IsNight() {
		screen.Fill(data.RGBPalette[s.scenarioData.Data.DaytimePalette[2]])
		s.separatorRect.SetColor(int(s.scenarioData.Data.DaytimePalette[0]))
	} else {
		screen.Fill(data.RGBPalette[s.scenarioData.Data.NightPalette[2]])
		s.separatorRect.SetColor(int(s.scenarioData.Data.NightPalette[0]))
	}
	s.mapView.SetIsNight(s.gameState.IsNight())
	s.mapView.SetUseIcons(s.options.UnitDisplay == 1)

	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Scale(2, 1)
	opts.GeoM.Translate(8, 72)

	if s.flashback != nil {
		s.flashback.Draw(screen, &opts)
	} else {
		s.mapView.Draw(screen, &opts)
		if s.animation != nil {
			s.animation.Draw(screen, &opts)
		}
	}

	playerBaseColor := s.scenarioData.Data.SideColor[s.playerSide] * 16
	opts.GeoM.Reset()
	s.topRect.SetColor(playerBaseColor + 10)
	s.topRect.Draw(screen, &opts)
	opts.GeoM.Translate(0, 22)
	s.messageBox.SetRowBackground(0, playerBaseColor+12)
	s.messageBox.SetRowBackground(1, playerBaseColor+10)
	s.messageBox.SetRowBackground(2, playerBaseColor+12)
	s.messageBox.SetRowBackground(3, playerBaseColor+10)
	s.messageBox.SetRowBackground(4, playerBaseColor+12)
	s.messageBox.SetTextColor(playerBaseColor)
	s.messageBox.Draw(screen, &opts)
	opts.GeoM.Translate(0, 40)
	s.statusBar.Draw(screen, &opts)
	opts.GeoM.Translate(0, 8)
	s.separatorRect.Draw(screen, &opts)
}
