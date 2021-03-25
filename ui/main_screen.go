package ui

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/pwiecz/command_series/lib"
)

type MainScreen struct {
	selectedScenario int
	selectedVariant  int
	scenarioData     *lib.ScenarioData
	gameData         *lib.GameData
	options          *lib.Options
	audioPlayer      *AudioPlayer
	onGameOver       func(int, int, int)

	mapView                                  *MapView
	topRect, leftRect, rightRect, bottomRect *Rectangle
	messageBox                               *MessageBox
	statusBar                                *MessageBox
	separatorRect                            *Rectangle

	flashback *Flashback
	animation Animation

	idleTicksLeft  int
	turboMode      bool
	isFrozen       bool
	areUnitsHidden bool
	playerSide     int

	orderedUnit *lib.Unit

	gameState     *lib.GameState
	commandBuffer *CommandBuffer

	sync    *lib.MessageSync
	started bool

	overviewMap *OverviewMap
	inputBox    *InputBox
	listBox     *ListBox

	lastMessageFromUnit lib.MessageFromUnit

	gameOver bool
}

func NewMainScreen(g *Game, options *lib.Options, audioPlayer *AudioPlayer, rand *rand.Rand, onGameOver func(int, int, int)) *MainScreen {
	scenario := &g.gameData.Scenarios[g.selectedScenario]
	for x := scenario.MinX - 1; x <= scenario.MaxX+1; x++ {
		g.gameData.Map.SetTile(lib.MapCoords{x, scenario.MinY - 1}, 12)
		g.gameData.Map.SetTile(lib.MapCoords{x, scenario.MaxY + 1}, 12)
	}
	for y := scenario.MinY; y <= scenario.MaxY; y++ {
		g.gameData.Map.SetTile(lib.MapCoords{scenario.MinX - 1, y}, 10)
		g.gameData.Map.SetTile(lib.MapCoords{scenario.MaxX + 1, y}, 12)
	}
	s := &MainScreen{
		selectedScenario: g.selectedScenario,
		selectedVariant:  g.selectedVariant,
		gameData:         g.gameData,
		scenarioData:     g.scenarioData,
		options:          options,
		audioPlayer:      audioPlayer,
		commandBuffer:    NewCommandBuffer(20),
		sync:             lib.NewMessageSync(),
		onGameOver:       onGameOver}
	if options.AlliedCommander == lib.Player {
		s.playerSide = 0
	} else {
		s.playerSide = 1
	}
	s.gameState = lib.NewGameState(rand, g.gameData, g.scenarioData, g.selectedScenario, g.selectedVariant, s.playerSide, s.options, s.sync)
	s.mapView = NewMapView(
		8, 72, 320, 19*8,
		g.gameData.Map, s.gameState.TerrainTypeMap(), g.scenarioData.Units,
		scenario.MinX, scenario.MinY, scenario.MaxX, scenario.MaxY,
		&g.gameData.Sprites.TerrainTiles,
		&g.gameData.Sprites.UnitSymbolSprites, &g.gameData.Sprites.UnitIconSprites,
		&g.gameData.Icons.Sprites, &g.scenarioData.Data.DaytimePalette, &g.scenarioData.Data.NightPalette)
	s.mapView.SetCursorPosition(lib.MapCoords{scenario.MinX + 10, scenario.MinY + 9})
	s.messageBox = NewMessageBox(0, 22, 336, 40, g.gameData.Sprites.GameFont)
	s.messageBox.Print("PREPARE FOR BATTLE!", 12, 1)
	s.statusBar = NewMessageBox(0, 62, 376, 8, g.gameData.Sprites.GameFont)
	s.statusBar.SetTextColor(16)
	s.statusBar.SetRowBackground(0, 30)

	s.topRect = NewRectangle(0, 0, 336, 22)
	s.leftRect = NewRectangle(0, 72, 8, 240-72)
	s.rightRect = NewRectangle(336-8, 72, 8, 240-72)
	s.bottomRect = NewRectangle(8, 240-16, 320, 16)
	s.separatorRect = NewRectangle(0, 70, 336, 2)
	return s
}

func (s *MainScreen) Update() error {
	if s.overviewMap != nil {
		for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
			if k == ebiten.KeyAlt || k == ebiten.KeyControl || k == ebiten.KeyShift /*|| k == ebiten.KeySuper*/ {
				continue
			}
			if inpututil.IsKeyJustPressed(k) {
				s.overviewMap = nil
				if s.areUnitsHidden {
					s.toggleHideUnits()
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
				s.toggleHideUnits()
			}
		}
		return nil
	}
	if s.inputBox != nil {
		s.inputBox.Update()
		return nil
	}
	if s.listBox != nil {
		s.listBox.Update()
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
					s.statusBar.Print("FROZEN", 2, 0)
				} else {
					s.statusBar.Print("UNFROZEN", 2, 0)
				}
			case StatusReport:
				if !s.gameOver {
					s.showStatusReport()
					s.idleTicksLeft = s.options.Speed.DelayTicks()
				} else {
					result, balance, rank := s.gameState.FinalResults(s.playerSide)
					s.onGameOver(result, balance, rank)
				}
			case UnitInfo:
				s.showUnitInfo()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case GeneralInfo:
				s.showGeneralInfo()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case CityInfo:
				s.showCityInfo()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case HideUnits:
				s.toggleHideUnits()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case ShowOverviewMap:
				s.showOverviewMap()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case ShowFlashback:
				s.showFlashback()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case Who:
				s.showLastMessageUnit()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case DecreaseSpeed:
				if s.gameOver {
					break
				}
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				s.decreaseGameSpeed()
			case IncreaseSpeed:
				if s.gameOver {
					break
				}
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				s.increaseGameSpeed()
			case SwitchUnitDisplay:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				s.options.UnitDisplay = 1 - s.options.UnitDisplay
			case SwitchSides:
				s.playerSide = 1 - s.playerSide
				s.orderedUnit = nil
				s.gameState.SwitchSides()
				s.mapView.HideIcon()
				s.messageBox.Clear()
				s.messageBox.Print(s.scenarioData.Data.Sides[s.playerSide]+" PLAYER:", 2, 0)
				s.messageBox.Print("PRESS \"T\" TO CONTINUE", 2, 1)
				if !s.areUnitsHidden {
					s.toggleHideUnits()
				}
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case Quit:
				if !s.gameOver {
					s.sync.Stop()
				}
				return fmt.Errorf("QUIT")
			case Reserve:
				if s.gameOver {
					break
				}
				s.tryGiveOrderAtMapCoords(s.mapView.cursorXY, lib.Reserve)
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case Defend:
				if s.gameOver {
					break
				}
				s.tryGiveOrderAtMapCoords(s.mapView.cursorXY, lib.Defend)
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case Attack:
				if s.gameOver {
					break
				}
				s.tryGiveOrderAtMapCoords(s.mapView.cursorXY, lib.Attack)
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case Move:
				if s.gameOver {
					break
				}
				s.tryGiveOrderAtMapCoords(s.mapView.cursorXY, lib.Move)
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case SetObjective:
				if s.gameOver {
					break
				}
				s.trySetObjective(s.mapView.cursorXY)
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			case ScrollDown:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X, curXY.Y + 1})
			case ScrollDownFast:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X, curXY.Y + 2})
			case ScrollUp:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X, curXY.Y - 1})
			case ScrollUpFast:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X, curXY.Y - 2})
			case ScrollRight:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X + 1, curXY.Y})
			case ScrollRightFast:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X + 2, curXY.Y})
			case ScrollLeft:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X - 1, curXY.Y})
			case ScrollLeftFast:
				s.idleTicksLeft = s.options.Speed.DelayTicks()
				curXY := s.mapView.GetCursorPosition()
				s.mapView.SetCursorPosition(lib.MapCoords{curXY.X - 2, curXY.Y})
			case Save:
				if s.gameOver {
					break
				}
				s.saveGame()
			case Load:
				if s.gameOver {
					break
				}
				s.loadGame()
			case TurboMode:
				s.turboMode = !s.turboMode
			}
		default:
		}
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
			mouseX, mouseY := ebiten.CursorPosition()
			xy := s.screenCoordsToUnitCoords(mouseX, mouseY)
			if s.mapView.AreMapCoordsVisible(xy.ToMapCoords()) {
				s.mapView.SetCursorPosition(xy.ToMapCoords())
			}
		}
		if inpututil.MouseButtonPressDuration(ebiten.MouseButtonLeft) > 30 {
			mouseX, mouseY := ebiten.CursorPosition()
			xy := s.screenCoordsToUnitCoords(mouseX, mouseY)
			if s.mapView.AreMapCoordsVisible(xy.ToMapCoords()) {
				s.mapView.SetCursorPosition(xy.ToMapCoords())
				s.pickOrder(xy)
			}
		}
		for _, touchID := range inpututil.JustPressedTouchIDs() {
			touchX, touchY := ebiten.TouchPosition(touchID)
			xy := s.screenCoordsToUnitCoords(touchX, touchY)
			if s.mapView.AreMapCoordsVisible(xy.ToMapCoords()) {
				s.mapView.SetCursorPosition(xy.ToMapCoords())
				break
			}
		}
		for _, touchID := range ebiten.TouchIDs() {
			if inpututil.TouchPressDuration(touchID) > 30 {
				touchX, touchY := ebiten.TouchPosition(touchID)
				xy := s.screenCoordsToUnitCoords(touchX, touchY)
				if s.mapView.AreMapCoordsVisible(xy.ToMapCoords()) {
					s.mapView.SetCursorPosition(xy.ToMapCoords())
					s.pickOrder(xy)
					break
				}
			}
		}
	}
	if s.isFrozen || s.areUnitsHidden || s.gameOver {
		return nil
	}
	if s.turboMode {
		s.idleTicksLeft = 0
	}
	if s.idleTicksLeft > 0 {
		s.idleTicksLeft--
		return nil
	}
loop:
	for {
		update := s.sync.GetUpdate()
		if update == nil {
			// some delay to "simulate" computation time
			s.idleTicksLeft = 15
			break loop
		}
		switch message := update.(type) {
		case lib.Initialized:
			s.idleTicksLeft = 60
			s.statusBar.Clear()
			s.statusBar.Print(s.dateTimeString(), 2, 0)
			break loop
		case lib.MessageFromUnit:
			unit := message.Unit()
			if unit.Side == s.playerSide {
				s.showMessageFromUnit(message)
				break loop
			} else if s.gameData.Game == lib.Conflict {
				if msg, ok := message.(lib.WeAreAttacking); ok {
					s.showMessageFromUnit(msg.EnemyMessage())
					break loop
				}
			}
		case lib.UnitAttack:
			if !s.turboMode {
				s.animation = NewIconsAnimation(s.mapView, lib.CircleIcons, message.XY.ToMapCoords())
				break loop
			}
		case lib.Reinforcements:
			if message.Sides[s.playerSide] {
				s.messageBox.Clear()
				s.messageBox.Print("REINFORCEMENTS!", 2, 1)
				s.idleTicksLeft = 100
			}
			break loop
		case lib.GameOver:
			s.gameOver = true
			s.showStatusReport()
			s.statusBar.Print("GAME OVER, PRESS '?' FOR RESULTS.", 2, 0)
			s.sync.Stop()
			break loop
		case lib.UnitMove:
			if !s.turboMode && (s.mapView.AreMapCoordsVisible(message.XY0) || s.mapView.AreMapCoordsVisible(message.XY1)) {
				s.animation = NewUnitAnimation(s.mapView /*s.audioPlayer*/, nil,
					message.Unit, message.XY0, message.XY1, 30)
				break loop
			}
		case lib.SupplyTruckMove:
			if !s.turboMode && (s.mapView.AreMapCoordsVisible(message.XY0) || s.mapView.AreMapCoordsVisible(message.XY1)) {
				s.animation = NewIconAnimation(s.mapView, lib.SupplyTruck,
					message.XY0, message.XY1, 1)
				break loop
			}
		case lib.WeatherForecast:
			s.messageBox.Clear()
			s.messageBox.Print(fmt.Sprintf("WEATHER FORECAST: %s", s.scenarioData.Data.Weather[message.Weather]), 2, 0)
		case lib.SupplyDistributionStart:
			s.mapView.HideIcon()
			s.messageBox.Print("* SUPPLY DISTRIBUTION *", 2, 1)
		case lib.SupplyDistributionEnd:
		case lib.DailyUpdate:
			s.messageBox.Print(fmt.Sprintf("%d DAYS REMAINING.", message.DaysRemaining), 2, 2)
			supplyLevels := []string{"CRITICAL", "SUFFICIENT", "AMPLE"}
			s.messageBox.Print(fmt.Sprintf("*SUPPLY LEVEL:* %s", supplyLevels[message.SupplyLevel]), 2, 3)
			s.idleTicksLeft = s.options.Speed.DelayTicks()
			break loop
		case lib.TimeChanged:
			s.statusBar.Clear()
			s.statusBar.Print(s.dateTimeString(), 2, 0)
			if s.gameState.Hour() == 18 && s.gameState.Minute() == 0 {
				s.showStatusReport()
				s.idleTicksLeft = s.options.Speed.DelayTicks()
			}
		default:
			return fmt.Errorf("Unknown message: %v", message)
		}
	}
	return nil
}

func (s *MainScreen) showMessageFromUnit(message lib.MessageFromUnit) {
	s.messageBox.Clear()
	s.messageBox.Print("*MESSAGE FROM ...*", 2, 0)
	messageUnit := message.Unit()
	// The unit might have moved after sending the message.
	unit := s.scenarioData.Units[messageUnit.Side][messageUnit.Index]
	s.messageBox.Print(fmt.Sprintf("%s:", unit.FullName()), 2, 1)
	lines := strings.Split("\""+message.String()+"\"", "\n")
	for i, line := range lines {
		s.messageBox.Print(line, 2, 2+i)
	}
	if s.areUnitCoordsVisible(unit.XY) {
		s.mapView.ShowIcon(message.Icon(), unit.XY.ToMapCoords(), 0, -5)
	} else {
		s.mapView.HideIcon()
	}
	// 15 added to "simulate" the computation time
	s.idleTicksLeft = s.options.Speed.DelayTicks() + 15
	s.lastMessageFromUnit = message
}
func (s *MainScreen) areUnitCoordsVisible(xy lib.UnitCoords) bool {
	return s.mapView.AreMapCoordsVisible(xy.ToMapCoords())
}
func (s *MainScreen) tryGiveOrderAtMapCoords(xy lib.MapCoords, order lib.OrderType) {
	s.messageBox.Clear()
	if unit, ok := s.scenarioData.Units.FindUnitOfSideAt(xy.ToUnitCoords(), s.playerSide); ok {
		s.giveOrder(unit, order)
		s.orderedUnit = &unit
	} else {
		s.messageBox.Print("NO FRIENDLY UNIT.", 2, 0)
	}
}
func (s *MainScreen) giveOrder(unit lib.Unit, order lib.OrderType) {
	unit.Order = order
	unit.HasLocalCommand = false
	switch order {
	case lib.Reserve:
		unit.Objective.X = 0
		s.messageBox.Print("RESERVE", 2, 0)
	case lib.Attack:
		unit.Objective.X = 0
		s.messageBox.Print("ATTACKING", 2, 0)
	case lib.Defend:
		unit.Objective = unit.XY
		s.messageBox.Print("DEFENDING", 2, 0)
	case lib.Move:
		s.messageBox.Print("MOVE WHERE ?", 2, 0)
	}
	s.scenarioData.Units[unit.Side][unit.Index] = unit
}
func (s *MainScreen) pickOrder(xy lib.UnitCoords) {
	s.messageBox.Clear()
	if unit, ok := s.scenarioData.Units.FindUnitOfSideAt(xy, s.playerSide); !ok {
		s.messageBox.Print("NO FRIENDLY UNIT.", 2, 0)
	} else {
		commands := []string{"MOVE", "ATTACK", "DEFEND", "RESERVE", "CANCEL"}
		s.listBox = NewListBox(4*8, 22, 7, 5, commands, s.gameData.Sprites.GameFont, func(command string) { s.orderPicked(command, unit) })
		playerBaseColor := s.scenarioData.Data.SideColor[s.playerSide] * 16
		s.listBox.SetTextColor(playerBaseColor)
		s.listBox.SetBackgroundColor(256)
	}
}
func (s *MainScreen) orderPicked(command string, unit lib.Unit) {
	s.listBox = nil
	if command == "MOVE" {
		s.giveOrder(unit, lib.Move)
	} else if command == "ATTACK" {
		s.giveOrder(unit, lib.Attack)
	} else if command == "DEFEND" {
		s.giveOrder(unit, lib.Defend)
	} else if command == "RESERVE" {
		s.giveOrder(unit, lib.Reserve)
	} else {
		return
	}
	s.orderedUnit = &unit
}

func (s *MainScreen) trySetObjective(xy lib.MapCoords) {
	if s.orderedUnit == nil {
		s.messageBox.Clear()
		s.messageBox.Print("GIVE ORDERS FIRST!", 2, 0)
		return
	}
	s.setObjective(s.scenarioData.Units[s.orderedUnit.Side][s.orderedUnit.Index], xy.ToUnitCoords())

}
func (s *MainScreen) setObjective(unit lib.Unit, xy lib.UnitCoords) {
	unit.Objective = xy
	unit.HasLocalCommand = false
	s.messageBox.Clear()
	s.messageBox.Print(fmt.Sprintf("*WHO * %s", unit.FullName()), 2, 0)
	s.messageBox.Print("OBJECTIVE HERE.", 2, 1)
	distance := unit.Function15_distanceToObjective()
	if distance > 0 {
		s.messageBox.Print(fmt.Sprintf("DISTANCE: %d MILES.", distance*s.scenarioData.Data.HexSizeInMiles), 2, 2)
	}
	s.scenarioData.Units[unit.Side][unit.Index] = unit
	s.orderedUnit = nil
}
func (s *MainScreen) showUnitInfo() {
	if s.areUnitsHidden {
		return
	}
	cursorXY := s.mapView.GetCursorPosition()
	unit, ok := s.scenarioData.Units.FindUnitAt(cursorXY.ToUnitCoords())
	if !ok {
		return
	}
	s.messageBox.Clear()
	if unit.Side == s.playerSide && unit.Objective.X > 0 && s.areUnitCoordsVisible(unit.Objective) {
		s.mapView.ShowAnimatedIcon(lib.ArrowIcons, unit.Objective.ToMapCoords(), 0, -5)
	} else {
		s.mapView.HideIcon()
	}

	if unit.Side != s.playerSide && !unit.InContactWithEnemy {
		s.messageBox.Print("* NO INFORMATION *", 2, 0)
		return
	}
	text := ""
	if unit.Side != s.playerSide {
		text += "* ENEMY UNIT *\n"
	}
	text += "*WHO * " + unit.FullName() + "\n"
	text += "*    * "
	men := unit.MenCount
	if unit.Side != s.playerSide {
		men -= men % 10
	}
	if men > 0 {
		text += fmt.Sprintf("%d MEN, ", men*s.scenarioData.Data.MenMultiplier)
	}
	tanks := unit.TankCount
	if unit.Side != s.playerSide {
		tanks -= tanks % 10
	}
	if tanks > 0 {
		text += fmt.Sprintf("%d %s, ", tanks*s.scenarioData.Data.TanksMultiplier, s.scenarioData.Data.Equipments[unit.Type])
	}
	text += "\n"

	if unit.Side == s.playerSide {
		text += "*    * "
		supplyDays := unit.SupplyLevel / (s.scenarioData.Data.AvgDailySupplyUse + s.scenarioData.Data.Data163)
		if s.gameData.Game != lib.Crusade {
			supplyDays /= 2
		}
		text += fmt.Sprintf("%d DAYS SUPPLY.", supplyDays)
		if !unit.HasSupplyLine {
			text += " (NO SUPPLY LINE!)"
		}
		text += "\n"
	}

	text += "*FORM* " + s.scenarioData.Data.Formations[unit.Formation]
	if unit.Side != s.playerSide {
		s.messageBox.Print(text, 2, 0)
		return
	}
	text += " *EXP* " + s.scenarioData.Data.Experience[unit.Morale/27] + " *EFF* " +
		fmt.Sprintf("%d", 10*((256-unit.Fatigue)/25)) + "\n*ORDR* " + unit.Order.String()
	if unit.HasLocalCommand {
		text += " (LOCAL COMMAND)"
	}
	s.messageBox.Print(text, 2, 0)
}
func numberToGeneralRating(num int) string {
	if num < 10 {
		return "POOR"
	}
	ratings := []string{"FAIR", "GOOD", "EXCELLNT"}
	return ratings[(num-10)/2]
}
func (s *MainScreen) showGeneralInfo() {
	if s.areUnitsHidden {
		return
	}
	cursorXY := s.mapView.GetCursorPosition()
	unit, ok := s.scenarioData.Units.FindUnitAt(cursorXY.ToUnitCoords())
	if !ok {
		return
	}
	s.messageBox.Clear()
	if unit.Side != s.playerSide {
		s.messageBox.Print("* NO INFORMATION *", 2, 0)
		return
	}
	general := unit.General
	s.messageBox.Print(fmt.Sprintf("*GENERAL * %-12s(%s)", general.Name, s.scenarioData.Data.Sides[unit.Side]), 2, 0)
	s.messageBox.Print("*ATTACK  * "+numberToGeneralRating(general.Attack), 2, 1)
	s.messageBox.Print("*DEFEND  * "+numberToGeneralRating(general.Defence), 2, 2)
	s.messageBox.Print("*MOVEMENT* "+numberToGeneralRating(general.Movement), 2, 3)
}
func (s *MainScreen) showCityInfo() {
	s.messageBox.Clear()
	cursorXY := s.mapView.GetCursorPosition()
	city, ok := s.scenarioData.Terrain.FindCityAt(cursorXY.ToUnitCoords())
	if !ok {
		s.messageBox.Print("NONE", 2, 0)
		return
	}
	s.messageBox.Print(city.Name, 2, 0)
	msg := fmt.Sprintf("%d VICTORY POINTS, %s", city.VictoryPoints, s.scenarioData.Data.Sides[city.Owner])
	s.messageBox.Print(msg, 2, 1)
}
func (s *MainScreen) showStatusReport() {
	s.messageBox.Clear()
	if s.gameData.Game != lib.Conflict {
		s.messageBox.Print(fmt.Sprintf("*STATUS REPORT* %-10s%-10s", s.scenarioData.Data.Sides[0], s.scenarioData.Data.Sides[1]), 2, 0)
		s.messageBox.Print(fmt.Sprintf("* TROOPS LOST * %-10d%-10d", s.gameState.MenLost(0), s.gameState.MenLost(1)), 2, 1)
		s.messageBox.Print(fmt.Sprintf("* TANKS  LOST * %-10d%-10d", s.gameState.TanksLost(0), s.gameState.TanksLost(1)), 2, 2)
		s.messageBox.Print(fmt.Sprintf("* CITIES HELD * %-10d%-10d", s.gameState.CitiesHeld(0), s.gameState.CitiesHeld(1)), 2, 3)
	} else {
		s.messageBox.Print(fmt.Sprintf("* STATUS REPORT *  %-10s%-10s", s.scenarioData.Data.Sides[0], s.scenarioData.Data.Sides[1]), 2, 0)
		s.messageBox.Print(fmt.Sprintf("* CASUALTIES    *  %-10d%-10d", s.gameState.MenLost(0), s.gameState.MenLost(1)), 2, 1)
		s.messageBox.Print(fmt.Sprintf("* MATERIEL      *  %-10d%-10d", s.gameState.TanksLost(0), s.gameState.MenLost(1)), 2, 2)
		s.messageBox.Print(fmt.Sprintf("* TERRITORY     *  %-10d%-10d", s.gameState.CitiesHeld(0), s.gameState.CitiesHeld(1)), 2, 3)
	}
	winningSide, advantage := s.gameState.WinningSideAndAdvantage()
	advantageStrs := []string{"SLIGHT", "MARGINAL", "TACTICAL", "DECISIVE", "TOTAL"}
	winningSideStr := s.scenarioData.Data.Sides[winningSide]
	s.messageBox.Print(fmt.Sprintf("%s %s ADVANTAGE.", advantageStrs[advantage], winningSideStr), 2, 4)
}
func (s *MainScreen) toggleHideUnits() {
	if s.areUnitsHidden {
		s.gameState.ShowAllVisibleUnits()
	} else {
		s.gameState.HideAllUnits()
	}
	s.areUnitsHidden = !s.areUnitsHidden
}
func (s *MainScreen) showOverviewMap() {
	if !s.areUnitsHidden {
		s.toggleHideUnits()
	}
	s.overviewMap = NewOverviewMap(s.gameData.Map, s.scenarioData.Units, s.gameData.Generic, s.scenarioData.Data, s.gameState.IsUnitVisible)
}
func (s *MainScreen) showFlashback() {
	if !s.areUnitsHidden {
		s.toggleHideUnits()
	}
	s.flashback = NewFlashback(s.mapView, s.messageBox, s.gameState.Flashback(), s.gameState.TerrainTypeMap())
}
func (s *MainScreen) showLastMessageUnit() {
	if s.lastMessageFromUnit == nil {
		return
	}
	messageUnit := s.lastMessageFromUnit.Unit()
	s.mapView.SetCursorPosition(messageUnit.XY.ToMapCoords())
	s.mapView.ShowIcon(s.lastMessageFromUnit.Icon(), messageUnit.XY.ToMapCoords(), 0, -5)
}
func (s *MainScreen) increaseGameSpeed() {
	s.changeGameSpeed(true)
}
func (s *MainScreen) decreaseGameSpeed() {
	s.changeGameSpeed(false)
}
func (s *MainScreen) changeGameSpeed(faster bool) {
	if faster {
		s.options.Speed = s.options.Speed.Faster()
	} else {
		s.options.Speed = s.options.Speed.Slower()
	}
	s.messageBox.Clear()
	s.messageBox.Print("SPEED: "+s.options.Speed.String(), 2, 0)
}

func (s *MainScreen) dateTimeString() string {
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

func (s *MainScreen) screenCoordsToUnitCoords(screenX, screenY int) lib.UnitCoords {
	//return s.mapView.ToUnitCoords((screenX-8)/2, screenY-72)
	return s.mapView.ScreenCoordsToUnitCoords(screenX, screenY)
}

func (s *MainScreen) saveGame() {
	s.messageBox.Clear()
	s.messageBox.Print("(PRESS ESCAPE TO CANCEL)", 2, 1)
	s.messageBox.Print("SAVE SCENARIO NAME: ?", 2, 2)
	s.inputBox = NewInputBox(23*8., 22+2*8., 8, s.gameData.Sprites.GameFont, func(filename string) { s.saveGameToFile(filename) })
}
func (s *MainScreen) saveGameToFile(filename string) {
	s.inputBox = nil
	if len(filename) == 0 {
		s.messageBox.Clear()
		return
	}
	s.messageBox.Print(filename, 23, 2)
	dir, err := saveDir(s.gameData.Scenarios[s.selectedScenario].FilePrefix)
	if err != nil {
		s.messageBox.Print("DISK ERROR: 1", 2, 4)
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		s.messageBox.Print("DISK ERROR: 2", 2, 4)
		return
	}
	file, err := os.Create(filepath.Join(dir, filename+".sav"))
	if err != nil {
		s.messageBox.Print("DISK ERROR: 3", 2, 4)
		return
	}
	defer file.Close()
	scenarioFilePrefix := s.gameData.Scenarios[s.selectedScenario].FilePrefix
	if _, err := file.Write([]byte(scenarioFilePrefix)); err != nil {
		s.messageBox.Print("DISK ERROR: 4", 2, 4)
		return
	}
	if _, err := file.Write([]byte{0}); err != nil {
		s.messageBox.Print("DISK ERROR: 5", 2, 4)
		return
	}
	if err := binary.Write(file, binary.LittleEndian, uint8(s.selectedScenario)); err != nil {
		s.messageBox.Print("DISK ERROR: 6", 2, 4)
		return
	}
	if err := binary.Write(file, binary.LittleEndian, uint8(s.selectedVariant)); err != nil {
		s.messageBox.Print("DISK ERROR: 7", 2, 4)
		return
	}
	if err := s.options.Write(file); err != nil {
		s.messageBox.Print("DISK ERROR: 8", 2, 4)
		return
	}
	if err := s.gameState.Save(file); err != nil {
		s.messageBox.Print("DISK ERROR: 10", 2, 4)
		return
	}
	s.messageBox.Print("COMPLETED", 2, 4)
}
func (s *MainScreen) loadGame() {
	s.messageBox.Clear()
	saveFiles := listSaveFiles(s.gameData.Scenarios[s.selectedScenario].FilePrefix)
	if len(saveFiles) == 0 {
		s.messageBox.Print("NO SAVEFILES FOUND", 2, 1)
		return
	}
	saveNames := make([]string, 0, len(saveFiles))
	for _, filename := range saveFiles {
		saveNames = append(saveNames, strings.TrimSuffix(filename, ".sav"))
	}
	s.messageBox.Print("(PRESS ESCAPE TO CANCEL)", 2, 1)
	s.messageBox.Print("LOAD SCENARIO NAME: ?", 2, 2)
	listLen := len(saveNames)
	if listLen > 8 {
		listLen = 8
	}
	s.listBox = NewListBox(23*8., 22+2*8, 8, listLen, saveNames, s.gameData.Sprites.GameFont, func(filename string) { s.loadGameFromFile(filename) })
	playerBaseColor := s.scenarioData.Data.SideColor[s.playerSide] * 16
	s.listBox.SetTextColor(playerBaseColor)
	s.listBox.SetBackgroundColor(int(s.scenarioData.Data.DaytimePalette[2]))
}

func (s *MainScreen) loadGameFromFile(filename string) {
	s.listBox = nil
	if len(filename) == 0 {
		s.messageBox.Clear()
		return
	}
	s.messageBox.Print(filename, 23, 2)
	dir, err := saveDir(s.gameData.Scenarios[s.selectedScenario].FilePrefix)
	if err != nil {
		s.messageBox.Print("DISK ERROR: 1", 2, 4)
		return
	}
	file, err := os.Open(filepath.Join(dir, filename+".sav"))
	if err != nil {
		s.messageBox.Print("CANNOT OPEN SAVEFILE", 2, 4)
		return
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	prefix, err := reader.ReadString(0)
	if err != nil {
		s.messageBox.Print("DISK ERROR: 3", 2, 4)
		return
	}
	var selectedScenario, selectedVariant uint8
	if err := binary.Read(reader, binary.LittleEndian, &selectedScenario); err != nil {
		s.messageBox.Print("DISK ERROR: 4", 2, 4)
		return
	}
	if err := binary.Read(reader, binary.LittleEndian, &selectedVariant); err != nil {
		s.messageBox.Print("DISK ERROR: 5", 2, 5)
		return
	}
	// strip the 0 delimiter from the end of the prefix
	prefix = prefix[:len(prefix)-1]
	scenarioFound := false
	for i, scenario := range s.gameData.Scenarios {
		if scenario.FilePrefix == prefix {
			if i == int(selectedScenario) {
				scenarioFound = true
			}
			break
		}
	}
	if !scenarioFound || int(selectedScenario) != s.selectedScenario {
		s.messageBox.Print("*WARNING:*   SCENARIO MISMATCH", 2, 4)
		return
	}
	if err := s.options.Read(reader); err != nil {
		s.messageBox.Print("DISK ERROR: 6", 2, 4)
		return
	}
	if !s.areUnitsHidden {
		s.toggleHideUnits()
	}
	if err := s.gameState.Load(reader); err != nil {
		s.messageBox.Print("DISK ERROR: 7", 2, 4)
		return
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		s.messageBox.Print("DISK ERROR: 8", 2, 4)
	}
	s.messageBox.Print("COMPLETED", 2, 3)
	s.messageBox.Print("PRESS \"T\" TO CONTINUE", 2, 4)

}

func (s *MainScreen) Draw(screen *ebiten.Image) {
	if s.overviewMap != nil {
		screen.Fill(lib.RGBPalette[8])
		opts := ebiten.DrawImageOptions{}
		opts.GeoM.Scale(4, 2)
		opts.GeoM.Translate(float64(336-4*s.gameData.Map.Width)/2, float64(240-2*s.gameData.Map.Height)/2)
		s.overviewMap.Draw(screen, &opts)
		return
	}
	if !s.gameState.IsNight() {
		screen.Fill(lib.RGBPalette[s.scenarioData.Data.DaytimePalette[2]])
		s.leftRect.SetColor(int(s.scenarioData.Data.DaytimePalette[2]))
		s.rightRect.SetColor(int(s.scenarioData.Data.DaytimePalette[2]))
		s.bottomRect.SetColor(int(s.scenarioData.Data.DaytimePalette[2]))
		s.separatorRect.SetColor(int(s.scenarioData.Data.DaytimePalette[0]))
	} else {
		screen.Fill(lib.RGBPalette[s.scenarioData.Data.NightPalette[2]])
		s.leftRect.SetColor(int(s.scenarioData.Data.NightPalette[2]))
		s.rightRect.SetColor(int(s.scenarioData.Data.NightPalette[2]))
		s.bottomRect.SetColor(int(s.scenarioData.Data.NightPalette[2]))
		s.separatorRect.SetColor(int(s.scenarioData.Data.NightPalette[0]))
	}
	s.mapView.SetIsNight(s.gameState.IsNight())
	s.mapView.SetUnitDisplay(s.options.UnitDisplay)

	if s.flashback != nil {
		s.flashback.Draw(screen)
	} else {
		s.mapView.Draw(screen)
		if s.animation != nil {
			s.animation.Draw(screen)
		}
	}

	playerBaseColor := s.scenarioData.Data.SideColor[s.playerSide] * 16
	s.topRect.SetColor(playerBaseColor + 10)
	s.topRect.Draw(screen)
	s.messageBox.SetRowBackground(0, playerBaseColor+12)
	s.messageBox.SetRowBackground(1, playerBaseColor+10)
	s.messageBox.SetRowBackground(2, playerBaseColor+12)
	s.messageBox.SetRowBackground(3, playerBaseColor+10)
	s.messageBox.SetRowBackground(4, playerBaseColor+12)
	s.messageBox.SetTextColor(playerBaseColor)
	s.messageBox.Draw(screen)
	s.statusBar.Draw(screen)
	s.leftRect.Draw(screen)
	s.rightRect.Draw(screen)
	s.bottomRect.Draw(screen)
	s.separatorRect.Draw(screen)

	if s.inputBox != nil {
		s.inputBox.SetTextColor(playerBaseColor)
		s.inputBox.SetBackgroundColor(playerBaseColor + 12)
		s.inputBox.Draw(screen)
	}
	if s.listBox != nil {
		s.listBox.Draw(screen)
	}

}
