package lib

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
)

type GameState struct {
	rand *rand.Rand

	game Game

	minute       int
	hour         int
	day          int /* 0-based */
	month        int /* 0-based */
	year         int
	daysElapsed  int
	weather      int
	isNight      bool
	supplyLevels [2]int

	playerSide                       int // remove it from here
	commanderFlags                   *CommanderFlags
	unitsUpdated                     int
	numUnitsToUpdatePerTimeIncrement int
	lastUpdatedUnit                  int

	menLost                   [2]int // 29927 + side*2
	tanksLost                 [2]int // 29927 + 4 + side*2
	citiesHeld                [2]int // 29927 + 13 + side*2
	criticalLocationsCaptured [2]int // 29927 + 21 + side*2
	flashback                 FlashbackHistory

	ai *AI

	scenarioData    *Data
	terrain         *Terrain
	terrainTypes    *TerrainTypeMap
	generic         *Generic
	hexes           *Hexes
	units           *Units
	generals        *Generals
	variants        []Variant
	selectedVariant int
	options         *Options

	sync *MessageSync

	allUnitsHidden bool
}

func NewGameState(rand *rand.Rand, gameData *GameData, scenarioData *ScenarioData, scenarioNum, variantNum int, playerSide int, options *Options, sync *MessageSync) *GameState {
	scenario := &gameData.Scenarios[scenarioNum]
	variant := &scenarioData.Variants[variantNum]
	sunriseOffset := Abs(6-scenario.StartMonth) / 2
	s := &GameState{}
	s.game = gameData.Game
	s.rand = rand
	s.game = gameData.Game
	s.minute = scenario.StartMinute
	s.hour = scenario.StartHour
	s.day = scenario.StartDay
	s.month = scenario.StartMonth
	s.year = scenario.StartYear
	s.weather = scenario.StartWeather
	s.isNight = s.hour < 5+sunriseOffset || s.hour > 20-sunriseOffset
	s.supplyLevels = scenario.StartSupplyLevels
	s.numUnitsToUpdatePerTimeIncrement = scenarioData.Data.UnitUpdatesPerTimeIncrement / 2
	s.lastUpdatedUnit = 127
	s.citiesHeld = variant.CitiesHeld
	s.scenarioData = scenarioData.Data
	s.units = scenarioData.Units
	s.terrain = scenarioData.Terrain
	s.terrainTypes = gameData.TerrainTypeMap
	s.generic = gameData.Generic
	s.hexes = gameData.Hexes
	s.generals = scenarioData.Generals
	s.variants = scenarioData.Variants
	s.selectedVariant = variantNum
	s.playerSide = playerSide
	s.commanderFlags = newCommanderFlags(options)
	s.ai = newAI(rand, s.commanderFlags, gameData, scenarioData)
	s.options = options
	s.sync = sync

	for side, sideUnits := range s.units {
		for i, unit := range sideUnits {
			if unit.VariantBitmap&(1<<variantNum) != 0 {
				unit.ClearState()
				unit.HalfDaysUntilAppear = 0
			}
			// The same slot is shared between VariantBitmap and Fatigue.
			unit.Fatigue = 0
			if side == 0 && options.GameBalance > 2 {
				unit.Morale = (3 + options.GameBalance) * unit.Morale / 5
			} else if side == 1 && options.GameBalance < 2 {
				unit.Morale = (7 - options.GameBalance) * unit.Morale / 5
			}
			sideUnits[i] = unit
		}
	}
	for i, city := range s.terrain.Cities {
		if city.VariantBitmap&(1<<variantNum) != 0 {
			city.VictoryPoints = 0
			s.terrain.Cities[i] = city
		}
	}
	s.ShowAllVisibleUnits()

	return s
}

func (s *GameState) Init() bool {
	if !s.everyHour() {
		return false
	}
	if !s.sync.SendUpdate(Initialized{}) {
		return false
	}
	return true
}

type saveData struct {
	Minute, Hour, Day, Month uint8
	Year                     uint16
	DaysElapsed              uint8
	Weather                  uint8
	IsNight                  bool

	PlayerSide     uint8
	CommanderFlags uint8

	SupplyLevels, MenLost, TanksLost, CitiesHeld [2]uint16
	CriticalLocationsCaptured                    [2]uint8

	SelectedVariant uint8

	UnitsUpdated                     uint8
	NumUnitsToUpdatePerTimeIncrement uint8
	LastUpdatedUnit                  uint8
	Update                           uint8

	Map0           [2][16][16]int16
	Map1           [2][16][16]int16
	Map3           [2][16][16]int16
	Map2_0, Map2_1 [2][4][4]int16
}

func (s *GameState) Save(writer io.Writer) error {
	if err := s.units.Write(writer); err != nil {
		return err
	}
	if err := s.terrain.Cities.WriteOwnerAndVictoryPoints(writer); err != nil {
		return err
	}
	if err := s.scenarioData.WriteFirst255Bytes(writer); err != nil {
		return err
	}
	var saveData saveData
	saveData.Minute = uint8(s.minute)
	saveData.Hour = uint8(s.hour)
	saveData.Day = uint8(s.day)
	saveData.Month = uint8(s.month)
	saveData.Year = uint16(s.year)
	saveData.DaysElapsed = uint8(s.daysElapsed)
	saveData.Weather = uint8(s.weather)
	saveData.IsNight = s.isNight
	saveData.PlayerSide = uint8(s.playerSide)
	saveData.CommanderFlags = s.commanderFlags.Serialize()
	saveData.SupplyLevels = [2]uint16{uint16(s.supplyLevels[0]), uint16(s.supplyLevels[1])}
	saveData.MenLost = [2]uint16{uint16(s.menLost[0]), uint16(s.menLost[1])}
	saveData.TanksLost = [2]uint16{uint16(s.tanksLost[0]), uint16(s.menLost[1])}
	saveData.CitiesHeld = [2]uint16{uint16(s.citiesHeld[0]), uint16(s.citiesHeld[1])}
	saveData.CriticalLocationsCaptured = [2]uint8{
		uint8(s.criticalLocationsCaptured[0]),
		uint8(s.criticalLocationsCaptured[1])}
	saveData.SelectedVariant = uint8(s.selectedVariant)
	saveData.UnitsUpdated = uint8(s.unitsUpdated)
	saveData.NumUnitsToUpdatePerTimeIncrement = uint8(s.numUnitsToUpdatePerTimeIncrement)
	saveData.LastUpdatedUnit = uint8(s.lastUpdatedUnit)
	saveData.Update = uint8(s.ai.update)

	for i := 0; i < 2; i++ {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				saveData.Map0[i][x][y] = int16(s.ai.map0[i][x][y])
				saveData.Map1[i][x][y] = int16(s.ai.map1[i][x][y])
				saveData.Map3[i][x][y] = int16(s.ai.map3[i][x][y])
			}
		}
		for x := 0; x < 4; x++ {
			for y := 0; y < 4; y++ {
				saveData.Map2_0[i][x][y] = int16(s.ai.map2_0[i][x][y])
				saveData.Map2_1[i][x][y] = int16(s.ai.map2_1[i][x][y])
			}
		}
	}

	if err := binary.Write(writer, binary.LittleEndian, saveData); err != nil {
		return err
	}
	if err := s.flashback.Write(writer); err != nil {
		return err
	}
	// array of saved game state numbers (first 19 single byte values mirrored to v10_ array)
	//   mapped to memory 29927-28 + i:
	// 0, 1, 2, 3, 4: minute, hour, day, month, year
	// 5: weather
	// 6: variant.Data3
	// 7: game speed
	// 8: "commander flags"
	// 9: 2^variant
	// 10: variant length in days
	// 11,12: critical locations to capture per side
	// 13: elapsed days
	// 14: game balance
	// 15: scenario * 16 + variant
	// 16, 18: MinX / MinY
	// 17, 19: MaxX / MaxY
	// --
	// 28,29,30,31: two byte values - men lost per side
	// 32,33,34,35: two byte values - tanks lost per side
	// 41,42,43,44: two byte values - cities held per side
	// 45,46,47,48: two byte values - supply levels per side
	// 49, 51: critical locations captured per side
	return nil
}
func (s *GameState) Load(reader io.Reader) error {
	units, err := ParseUnits(reader, s.scenarioData.UnitTypes, s.scenarioData.UnitNames, s.generals)
	if err != nil {
		return err
	}
	s.units = units
	if err := s.terrain.Cities.ReadOwnerAndVictoryPoints(reader); err != nil {
		return err
	}
	if err := s.scenarioData.ReadFirst255Bytes(reader); err != nil {
		return err
	}
	var saveData saveData
	if err := binary.Read(reader, binary.LittleEndian, &saveData); err != nil {
		return err
	}
	s.minute = int(saveData.Minute)
	s.hour = int(saveData.Hour)
	s.day = int(saveData.Day)
	s.month = int(saveData.Month)
	s.year = int(saveData.Year)
	s.daysElapsed = int(saveData.DaysElapsed)
	s.weather = int(saveData.Weather)
	s.isNight = saveData.IsNight
	s.playerSide = int(saveData.PlayerSide)
	s.commanderFlags.Deserialize(saveData.CommanderFlags)
	s.supplyLevels = [2]int{int(saveData.SupplyLevels[0]), int(saveData.SupplyLevels[1])}
	s.menLost = [2]int{int(saveData.MenLost[0]), int(saveData.MenLost[1])}
	s.tanksLost = [2]int{int(saveData.TanksLost[0]), int(saveData.TanksLost[1])}
	s.citiesHeld = [2]int{int(saveData.CitiesHeld[0]), int(saveData.CitiesHeld[1])}
	s.criticalLocationsCaptured = [2]int{
		int(saveData.CriticalLocationsCaptured[0]),
		int(saveData.CriticalLocationsCaptured[1])}
	s.selectedVariant = int(saveData.SelectedVariant)
	s.unitsUpdated = int(saveData.UnitsUpdated)
	s.numUnitsToUpdatePerTimeIncrement = int(saveData.NumUnitsToUpdatePerTimeIncrement)
	s.lastUpdatedUnit = int(saveData.LastUpdatedUnit)
	s.ai.update = int(saveData.Update)

	for i := 0; i < 2; i++ {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				s.ai.map0[i][x][y] = int(saveData.Map0[i][x][y])
				s.ai.map1[i][x][y] = int(saveData.Map1[i][x][y])
				s.ai.map3[i][x][y] = int(saveData.Map3[i][x][y])
			}
		}
		for x := 0; x < 4; x++ {
			for y := 0; y < 4; y++ {
				s.ai.map2_0[i][x][y] = int(saveData.Map2_0[i][x][y])
				s.ai.map2_1[i][x][y] = int(saveData.Map2_1[i][x][y])
			}
		}
	}
	if err := s.flashback.Read(reader); err != nil {
		return err
	}

	return nil
}
func (s *GameState) SwitchSides() {
	s.playerSide = 1 - s.playerSide
	s.commanderFlags.SwitchSides()
}
func (s *GameState) Update() bool {
	s.unitsUpdated++
	for ; s.unitsUpdated <= s.numUnitsToUpdatePerTimeIncrement; s.unitsUpdated++ {
		message, quit := s.updateUnit()
		if quit {
			return false
		}
		if !s.sync.SendUpdate(message) {
			return false
		}
	}
	s.unitsUpdated = 0

	s.minute += s.scenarioData.MinutesPerTick
	if s.minute >= 60 {
		s.minute = 0
		s.hour++
	}
	if s.hour >= 24 {
		s.hour = 0
		s.day++
	}
	// game treats all months to have 30 days.
	if s.day >= 30 { // monthLength(s.month+1, s.year+1900) {
		s.day = 0
		s.month++
	}
	if s.month >= 12 {
		s.month = 0
		s.year++
	}
	s.sync.SendUpdate(TimeChanged{})
	if s.minute == 0 {
		if !s.everyHour() {
			return false
		}
		if s.hour == 0 {
			if !s.everyDay() {
				return false
			}
		}
		if s.hour == 18 {
			if s.isGameOver() {
				s.sync.SendUpdate(TimeChanged{})
				s.sync.SendUpdate(GameOver{})
				return false
			}
		}
	}
	return true
}

func (s *GameState) updateUnit() (message MessageFromUnit, quit bool) {
	weather := s.weather
	if s.isNight {
		weather += 8
	}
nextUnit:
	s.lastUpdatedUnit = (s.lastUpdatedUnit + 1) % 128
	unit := s.units[s.lastUpdatedUnit/64][s.lastUpdatedUnit%64]
	if !unit.IsInGame {
		goto nextUnit
	}
	if !s.areUnitCoordsValid(unit.X, unit.Y) {
		panic(fmt.Errorf("%s@(%d,%d):%v", unit.FullName(), unit.X, unit.Y, unit))
	}
	var arg1 int
	if unit.MenCount+unit.EquipCount < 7 || unit.Fatigue == 255 {
		s.terrainTypes.hideUnit(unit)
		message = WeMustSurrender{unit}
		unit.ClearState()
		unit.HalfDaysUntilAppear = 0
		s.citiesHeld[1-unit.Side] += s.scenarioData.UnitScores[unit.Type]
		s.menLost[unit.Side] += unit.MenCount
		s.tanksLost[unit.Side] += unit.EquipCount
		goto end
	}
	if !s.scenarioData.UnitCanMove[unit.Type] {
		goto nextUnit
	}
	arg1 = s.ai.UpdateUnitObjective(&unit, weather)
	//l21:
	s.ai.update = unit.Side
	message = nil
	if unit.SupplyLevel == 0 {
		message = WeHaveExhaustedSupplies{unit}
	}
	{
		sx, sy, shouldQuit := s.performUnitMovement(&unit, &message, &arg1, weather)
		if shouldQuit {
			quit = true
			return
		}
		unit.SupplyLevel = Clamp(unit.SupplyLevel-2, 0, 255)
		wasInContactWithEnemy := unit.InContactWithEnemy

		unit.InContactWithEnemy = false
		unit.IsUnderAttack = false
		unit.State2 = false
		unit.State4 = false // &= 232
		if Rand(s.scenarioData.Data252[unit.Side], s.rand) == 0 {
			unit.SeenByEnemy = false // &= ~64
		}
		if s.game == Conflict && Rand(s.scenarioData.Data175, s.rand)/8 > 0 {
			unit.SeenByEnemy = true // |= 64
		}
		for i := 0; i < 6; i++ {
			nx, ny := s.generic.IthNeighbour(unit.X, unit.Y, i)
			if unit2, ok := s.units.FindUnitOfSideAt(nx, ny, 1-unit.Side); ok {
				unit2.InContactWithEnemy = true
				unit2.SeenByEnemy = true // |= 65
				s.terrainTypes.showUnit(unit2)
				s.units[unit2.Side][unit2.Index] = unit2
				if s.scenarioData.UnitScores[unit2.Type] > 8 {
					if !s.commanderFlags.PlayerControlled[unit.Side] {
						sx, sy = unit2.X, unit2.Y
						unit.Order = Attack
						arg1 = 7
						// arg2 = i
					}
				}
				// in CiE one of supply units or an air wing.
				// in DitD also minefield or artillery
				// in CiV supply units or bombers (not fighters nor artillery)
				if s.scenarioData.UnitMask[unit2.Type]&128 == 0 {
					unit.State4 = true // |= 16
				}
				if s.scenarioData.UnitCanMove[unit2.Type] {
					unit.InContactWithEnemy = true
					unit.SeenByEnemy = true // |= 65
					if unit.Side == 0 {
					}
					if !wasInContactWithEnemy {
						message = WeAreInContactWithEnemy{unit}
					}
				}
			}
		}
		s.function29_showUnit(unit)
		//	l11:
		if unit.ObjectiveX == 0 || unit.Order != Attack || arg1 < 7 {
			goto end
		}
		if unit.Function15_distanceToObjective() == 1 && s.units.IsUnitOfSideAt(sx, sy, unit.Side) {
			unit.ObjectiveX = 0
			goto end
		}
		unit.TargetFormation = s.scenarioData.function10(unit.Order, 2)
		if unit.Fatigue > 64 || unit.SupplyLevel == 0 ||
			!s.units.IsUnitOfSideAt(sx, sy, 1-unit.Side) ||
			unit.Formation != s.scenarioData.Data176[0][2] {
			goto end
		}
		if unit.FormationTopBit {
			s.terrainTypes.hideUnit(unit)
			if !s.sync.SendUpdate(UnitMove{unit, unit.X / 2, unit.Y, sx / 2, sy}) {
				quit = true
				return
			}
			s.terrainTypes.showUnit(unit)
			if s.game == Conflict {
				unit.InContactWithEnemy = true
				unit.SeenByEnemy = true // |= 65
			}
			// function14
		} else {
			susceptibleToWeather := s.scenarioData.Data32[unit.Type]&8 != 0
			if susceptibleToWeather && weather > 3 {
				// [53767] = 0
				goto end
			}
			// function27
		}
		// [53767] = 0
		s.performAttack(&unit, sx, sy, weather, &message)
	}
end: // l3
	for unit.Formation != unit.TargetFormation {
		dir := Sign(unit.Formation - unit.TargetFormation)
		speed := s.scenarioData.FormationChangeSpeed[(dir+1)/2][unit.Formation]
		if speed > Rand(15, s.rand) {
			unit.FormationTopBit = false
			unit.Formation -= dir
		}
		if speed&16 == 0 {
			break
		}
	}
	{
		recovery := s.scenarioData.RecoveryRate[unit.Type]
		if !unit.InContactWithEnemy && unit.HasSupplyLine { // &9 == 0
			recovery *= 2
		}
		unit.Fatigue = Clamp(unit.Fatigue-recovery, 0, 255)
	}
	s.units[unit.Side][unit.Index] = unit
	return
}

func (s *GameState) performUnitMovement(unit *Unit, message *MessageFromUnit, arg1 *int, weather int) (sx, sy int, quit bool) {
	// l22:
	for unitMoveBudget := 25; unitMoveBudget > 0; {
		if unit.ObjectiveX == 0 {
			return
		}
		distance := unit.Function15_distanceToObjective()
		d32 := s.scenarioData.Data32[unit.Type]
		attackRange := (d32 & 31) * 2
		if distance > 0 && distance <= attackRange && unit.Order == Attack {
			sx, sy = unit.ObjectiveX, unit.ObjectiveY
			unit.FormationTopBit = true
			*arg1 = 7
			return // goto l2
		}
		var moveSpeed int
		for mvAdd := 0; mvAdd <= 1; mvAdd++ { // l5:
			if unit.ObjectiveX == unit.X && unit.ObjectiveY == unit.Y {
				unit.ObjectiveX = 0
				unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
				return // goto l2
			}
			unit.TargetFormation = s.scenarioData.function10(unit.Order, 0)
			// If unit is player controlled or its command is local
			if !s.commanderFlags.PlayerControlled[unit.Side] || unit.HasLocalCommand {
				// If it's next to its objective to defend and it's in contact with enemy
				if distance == 1 && unit.Order == Defend && unit.InContactWithEnemy {
					unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
				}
			}
			sx, sy, moveSpeed = s.FindBestMoveFromTowards(unit.X, unit.Y, unit.ObjectiveX, unit.ObjectiveY, unit.Type, mvAdd)
			if d32&64 > 0 { // in CiV (some scenarios) artillery or mortars
				if s.game != Conflict || unit.Formation == 0 {
					sx, sy = unit.ObjectiveX, unit.ObjectiveY
					tt := s.terrainTypes.terrainOrUnitTypeAt(sx/2, sy)
					moveSpeed = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit.Type]
					*arg1 = tt // shouldn't have any impact
					mvAdd = 1
				} else if s.scenarioData.UnitMask[unit.Type]&32 != 0 {
					// Conflict && unit.Formation != 0
					return // goto l2
				}
			}
			if s.units.IsUnitOfSideAt(sx, sy, unit.Side) {
				moveSpeed = 0
			}
			if s.units.IsUnitOfSideAt(sx, sy, 1-unit.Side) {
				moveSpeed = -1
			}
			if moveSpeed >= 1 || (unit.Order == Attack && moveSpeed == -1) ||
				Abs(unit.ObjectiveX-unit.X)+Abs(unit.ObjectiveY-unit.Y) <= 2 {
				break
			}
		}

		if moveSpeed < 1 {
			return // goto l2
		}
		moveSpeed = moveSpeed * s.scenarioData.Data192[unit.Formation] / 8
		if unit.State4 {
			moveSpeed = moveSpeed * s.scenarioData.Data200Low[unit.Type] / 8
		}
		moveSpeed *= (512 - unit.Fatigue) / 32
		moveSpeed = moveSpeed * unit.General.Movement / 16
		if unit.SupplyLevel == 0 {
			moveSpeed /= 2
		}
		if s.game != Crusade {
			if moveSpeed == 0 {
				return
			}
		}
		totalMoveCost := 1024
		if s.game == Conflict {
			totalMoveCost = 1023
		}
		if s.scenarioData.UnitMask[unit.Type]&4 != 0 {
			if s.game != Conflict {
				totalMoveCost += weather * 128
			} else {
				totalMoveCost += weather * 256
			}
		}
		totalMoveCost *= 8
		var moveCost int
		if s.game == Crusade {
			moveCost = totalMoveCost / (moveSpeed + 1)
		} else {
			moveCost = totalMoveCost / moveSpeed
		}
		if moveCost > unitMoveBudget && Rand(moveCost, s.rand) > unitMoveBudget {
			return
		}
		unitMoveBudget -= moveCost
		s.terrainTypes.hideUnit(*unit)
		if s.commanderFlags.PlayerCanSeeUnits[unit.Side] ||
			unit.InContactWithEnemy || unit.SeenByEnemy {
			if !s.sync.SendUpdate(UnitMove{*unit, unit.X / 2, unit.Y, sx / 2, sy}) {
				quit = true
				return
			}
		}
		unit.X, unit.Y = sx, sy
		s.function29_showUnit(*unit)
		if unit.Function15_distanceToObjective() == 0 {
			unit.ObjectiveX = 0
			unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
			if (unit.Order == Defend || unit.Order == Move) &&
				!unit.HasLocalCommand {
				*message = WeHaveReachedOurObjective{*unit}
			}
		}
		unit.Fatigue = Clamp(unit.Fatigue+s.scenarioData.Data173, 0, 255)
		if city, captured := s.function16(*unit); captured {
			*message = WeHaveCaptured{*unit, city}
			return
		}
		if unitMoveBudget > 0 {
			if s.units.NeighbourUnitCount(unit.X, unit.Y, 1-unit.Side) > 0 {
				unit.InContactWithEnemy = true
				unit.State4 = true // |= 17
			} else {
				unit.InContactWithEnemy = false // &= 254
			}
			s.function29_showUnit(*unit)
		}
	}
	// l2:
	return
}

func (s *GameState) performAttack(unit *Unit, sx, sy, weather int, message *MessageFromUnit) {
	if s.game != Conflict {
		unit.InContactWithEnemy = true
		unit.SeenByEnemy = true // |= 65
	}

	unit2, ok := s.units.FindUnitOfSideAt(sx, sy, 1-unit.Side)
	if !ok {
		panic("")
	}
	*message = WeAreAttacking{*unit, unit2, 0 /* placeholder value */, s.scenarioData.Formations}
	var attackerScore int
	{
		tt := s.terrainTypes.terrainTypeAt(unit.X/2, unit.Y)
		var menCoeff int
		if !unit.FormationTopBit {
			menCoeff = s.scenarioData.TerrainMenAttack[tt] * s.scenarioData.FormationMenAttack[unit.Formation] * unit.MenCount / 32
		}
		tankCoeff := s.scenarioData.TerrainTankAttack[tt] * s.scenarioData.FormationTankAttack[unit.Formation] * s.scenarioData.Data16High[unit.Type] / 2 * unit.EquipCount / 64
		susceptibleToWeather := (s.scenarioData.Data32[unit.Type] & 8) != 0
		if s.game == Conflict {
			susceptibleToWeather = (s.scenarioData.Data32[unit.Type] & 32) != 0
		}
		if unit.FormationTopBit && susceptibleToWeather {
			// long range unit
			if weather > 3 {
				return //goto end
			}
			tankCoeff = tankCoeff * (4 - weather) / 4
		}
		attackerScore = (menCoeff + tankCoeff) * unit.Morale / 256 * (255 - unit.Fatigue) / 128
		attackerScore = attackerScore * unit.General.Attack / 16
		attackerScore = attackerScore * s.ai.NeighbourScore(&s.hexes.Arr144, unit.X, unit.Y, unit.Side) / 8
		attackerScore++
	}

	var defenderScore int
	{
		if s.scenarioData.UnitScores[unit2.Type]&248 > 0 {
			unit.State2 = true // |= 4
		}
		tt2 := s.terrainTypes.terrainTypeAt(unit2.X/2, unit2.Y)
		menCoeff := s.scenarioData.TerrainMenDefence[tt2] * s.scenarioData.FormationMenDefence[unit2.Formation] * unit2.MenCount / 32
		tankCoeff := s.scenarioData.TerrainTankAttack[tt2] * s.scenarioData.FormationTankDefence[unit2.Formation] * s.scenarioData.Data16Low[unit2.Type] / 2 * unit2.EquipCount / 64
		defenderScore = (menCoeff + tankCoeff) * unit2.Morale / 256 * (240 - unit2.Fatigue/2) / 128
		defenderScore = defenderScore * unit2.General.Defence / 16
		if unit2.SupplyLevel == 0 {
			defenderScore = defenderScore * s.scenarioData.Data167 / 8
		}
		defenderScore = defenderScore * s.ai.NeighbourScore(&s.hexes.Arr144, unit2.X, unit2.Y, 1-unit.Side) / 8
		defenderScore++
	}

	arg1 := defenderScore * 16 / attackerScore
	if s.scenarioData.UnitMask[unit.Type]&4 == 0 {
		arg1 += weather
	}
	arg1 = Clamp(arg1, 0, 63)
	if !unit.FormationTopBit || s.scenarioData.Data32[unit.Type]&128 == 0 {
		menLost := Clamp((Rand(unit.MenCount*arg1, s.rand)+255)/512, 0, unit.MenCount)
		s.menLost[unit.Side] += menLost
		unit.MenCount -= menLost
		tanksLost := Clamp((Rand(unit.EquipCount*arg1, s.rand)+255)/512, 0, unit.EquipCount)
		s.tanksLost[unit.Side] += tanksLost
		unit.EquipCount -= tanksLost
		if arg1 < 24 {
			unit.Morale = Clamp(unit.Morale+1, 0, 250)
		}
		unit2.IsUnderAttack = true //  |= 2
		if arg1 > 32 {
			unit.Order = Defend // ? ^48
			*message = WeHaveMetStrongResistance{*unit}
			unit.Morale = Abs(unit.Morale - 2)
		}
	}
	unit.Fatigue = Clamp(unit.Fatigue+arg1, 0, 255)
	unit.SupplyLevel = Clamp(unit.SupplyLevel-s.scenarioData.Data162, 0, 255)

	arg1 = attackerScore*16/defenderScore - weather
	if s.game == Crusade {
		arg1 = Clamp(arg1, 0, 63)
	} else {
		arg1 = Clamp(arg1, 0, 128)
	}
	// function13(sx, sy)
	// function4(arg1)
	s.sync.SendUpdate(UnitAttack{sx, sy, arg1})

	menLost2 := Clamp((Rand(unit2.MenCount*arg1, s.rand)+500)/512, 0, unit2.MenCount)
	s.menLost[1-unit.Side] += menLost2
	unit2.MenCount -= menLost2
	tanksLost2 := Clamp((Rand(unit2.EquipCount*arg1, s.rand)+255)/512, 0, unit2.EquipCount)
	s.tanksLost[1-unit.Side] += tanksLost2
	unit2.EquipCount -= tanksLost2
	unit2.SupplyLevel = Clamp(unit2.SupplyLevel-s.scenarioData.Data163, 0, 255)
	if s.scenarioData.UnitCanMove[unit2.Type] &&
		((s.game != Conflict && !unit.FormationTopBit) ||
			(s.game == Conflict && s.scenarioData.UnitMask[unit2.Type]&2 == 0)) &&
		arg1-s.scenarioData.Data0Low[unit2.Type]*2+unit2.Fatigue/4 > 36 {
		unit2.Morale = Abs(unit2.Morale - 1)
		oldX, oldY := unit2.X, unit2.Y
		bestX, bestY := unit2.X, unit2.Y
		s.terrainTypes.hideUnit(unit2)
		if unit2.Fatigue > 128 {
			unit2SupplyUnit := s.units[unit2.Side][unit2.SupplyUnit]
			if unit2SupplyUnit.IsInGame {
				unit2.Morale = Abs(unit2.Morale - s.units.NeighbourUnitCount(unit2.X, unit2.Y, unit.Side)*4)
				unit2.X, unit2.Y = unit2SupplyUnit.X, unit2SupplyUnit.Y
				unit2.ClearState()
				unit2.HalfDaysUntilAppear = 6
				unit2.InvAppearProbability = 6
				if s.game != Crusade {
					unit2.HalfDaysUntilAppear = 4
					unit2.InvAppearProbability = 4
					if s.game == Decision {
						unit2.Fatigue = 130
					} else {
						unit2.Fatigue = 120
					}
				}
				*message = WeHaveBeenOverrun{unit2}
			}
		}
		bestDefence := -128
		for i := 0; i <= 6; i++ {
			nx, ny := s.generic.IthNeighbour(unit2.X, unit2.Y, i)
			if !s.areUnitCoordsValid(nx, ny) || s.units.IsUnitAt(nx, ny) || s.terrain.IsCityAtUnitCoords(nx, ny) {
				continue
			}
			tt := s.terrainTypes.terrainOrUnitTypeAt(nx/2, ny)
			if s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit2.Type] == 0 {
				continue
			}
			r := s.scenarioData.TerrainMenDefence[tt] +
				s.ai.NeighbourScore(&s.hexes.Arr96, nx, ny, 1-unit.Side)*4
			if r > 11 && r >= bestDefence {
				bestDefence = r
				bestX, bestY = nx, ny
			}
		}
		unit2.X, unit2.Y = bestX, bestY // moved this up comparing to the original code
		if _, ok := (*message).(WeHaveBeenOverrun); !ok {
			if s.game != Conflict {
				s.terrainTypes.showUnit(unit2)
				unit.ObjectiveX, unit.ObjectiveY = unit2.X, unit2.Y
			} else {
				if s.commanderFlags.PlayerCanSeeUnits[1-unit.Side] {
					s.terrainTypes.showUnit(unit2)
				}
				unit2.InContactWithEnemy = false
				unit2.SeenByEnemy = false // &= 190
			}
		}
		if bestX != oldX || bestY != oldY {
			// unit2 is retreating, unit one is chasing (and maybe capturing a city)
			if _, ok := (*message).(WeHaveBeenOverrun); !ok {
				*message = WeAreRetreating{unit2}
			}
			tt := s.terrainTypes.terrainOrUnitTypeAt(oldX/2, oldY)
			if arg1 > 60 && (s.game != Conflict || !unit.FormationTopBit) &&
				s.ai.NeighbourScore(&s.hexes.Arr96, oldX, oldY, unit.Side) > -4 &&
				s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit.Type] > 0 {
				s.terrainTypes.hideUnit(*unit)
				unit.X, unit.Y = oldX, oldY
				s.terrainTypes.showUnit(*unit)
				if city, captured := s.function16(*unit); captured {
					*message = WeHaveCaptured{*unit, city}
				}
			}
		} else {
			*message = nil
		}
		unit2.Formation = s.scenarioData.Data176[1][0]
		unit2.Order = OrderType((s.scenarioData.Data176[1][0] + 1) % 4)
		unit2.HasSupplyLine = false // |= 32
	}

	a := arg1
	if _, ok := (*message).(WeAreRetreating); ok { // are retreating
		a /= 2
	}
	unit2.Fatigue = Clamp(unit2.Fatigue+a, 0, 255)

	if arg1 < 24 {
		unit2.Morale = Clamp(unit2.Morale+1, 0, 250)
	}
	s.units[unit2.Side][unit2.Index] = unit2
	if attack, ok := (*message).(WeAreAttacking); ok {
		// update arg1 value if the message is still WeAreAttacking
		*message = WeAreAttacking{attack.unit, attack.enemy, arg1, attack.formationNames}
	}

}

// Has unit captured a city
func (s *GameState) function16(unit Unit) (City, bool) {
	if city, ok := s.terrain.FindCityAtUnitCoords(unit.X, unit.Y); ok {
		if city.Owner != unit.Side {
			// msg = 5
			city.Owner = unit.Side
			s.SaveCity(city)
			s.citiesHeld[unit.Side] += city.VictoryPoints
			s.citiesHeld[1-unit.Side] -= city.VictoryPoints
			s.criticalLocationsCaptured[unit.Side] += city.VictoryPoints & 1
			return city, true
		}
	}
	return City{}, false
}

func (s *GameState) function29_showUnit(unit Unit) {
	if unit.InContactWithEnemy || unit.SeenByEnemy /* &65 != 0 */ ||
		s.commanderFlags.PlayerCanSeeUnits[unit.Side] {
		s.terrainTypes.showUnit(unit)
	}
}

func (s *GameState) everyHour() bool {
	sunriseOffset := Abs(6-s.month) / 2
	s.isNight = s.hour < 5+sunriseOffset || s.hour > 20-sunriseOffset

	if s.hour == 12 {
		if !s.every12Hours() {
			return false
		}
	}

	if s.scenarioData.AvgDailySupplyUse > Rand(24, s.rand) {
		for _, sideUnits := range s.units {
			for i, unit := range sideUnits {
				if !unit.IsInGame ||
					!s.scenarioData.UnitUsesSupplies[unit.Type] ||
					unit.SupplyLevel <= 0 {
					continue
				}
				unit.SupplyLevel--
				sideUnits[i] = unit
			}
		}

	}
	return true
}

func (s *GameState) every12Hours() bool {
	var reinforcements [2]bool
	s.supplyLevels[0] += s.scenarioData.ResupplyRate[0] * 2
	s.supplyLevels[1] += s.scenarioData.ResupplyRate[1] * 2
	s.HideAllUnits()
	// In CiE and DiD resupply at midnight, in CiV resupply at midday.
	resupply := (s.game != Conflict && s.isNight) || (s.game == Conflict && !s.isNight)
	if resupply {
		s.sync.SendUpdate(SupplyDistributionStart{})
	}
	for _, sideUnits := range s.units {
		for i, unit := range sideUnits {
			if unit.IsInGame {
				if resupply {
					unit = s.resupplyUnit(unit)
				}
			} else {
				if unit.HalfDaysUntilAppear == 0 {
					continue
				}
				unit.HalfDaysUntilAppear--
				if unit.HalfDaysUntilAppear == 0 {
					shouldSpawnUnit := !s.units.IsUnitAt(unit.X, unit.Y) &&
						Rand(unit.InvAppearProbability, s.rand) == 0
					if city, ok := s.terrain.FindCityAtUnitCoords(unit.X, unit.Y); ok && city.Owner != unit.Side {
						shouldSpawnUnit = false
					}
					if shouldSpawnUnit {
						unit.IsInGame = true
						// Unit will be shown if needed inside ShowAllUnits at the end of the function.
						reinforcements[unit.Side] = true
					} else {
						unit.HalfDaysUntilAppear = 1
					}
				}
			}
			sideUnits[i] = unit
			//[53249] = 0
		}
	}

	for _, sideUnits := range s.units {
		for i, unit := range sideUnits {
			if unit.HasSupplyLine { // (has supply line)
				if unit.MenCount <= s.scenarioData.MenCountLimit[unit.Type] {
					unit.MenCount += Rand(s.scenarioData.MenReplacementRate[unit.Side]+32, s.rand) / 32
				}
				if unit.EquipCount <= s.scenarioData.EquipCountLimit[unit.Type] {
					unit.EquipCount += Rand(s.scenarioData.EquipReplacementRate[unit.Side]+32, s.rand) / 32
				}
			}
			sideUnits[i] = unit
		}
	}
	s.ShowAllVisibleUnits()
	if resupply {
		s.sync.SendUpdate(SupplyDistributionEnd{})
	}
	if reinforcements[0] || reinforcements[1] {
		if !s.sync.SendUpdate(Reinforcements{Sides: reinforcements}) {
			return false
		}
	}
	return true
}

func (s *GameState) resupplyUnit(unit Unit) Unit {
	unitVisible := s.commanderFlags.PlayerCanSeeUnits[unit.Side]
	unit.OrderBit4 = false
	if !s.scenarioData.UnitUsesSupplies[unit.Type] ||
		!s.scenarioData.UnitCanMove[unit.Type] {
		return unit
	}
	// Mark initially that there's no supply line.
	unit.HasSupplyLine = false
	minSupplyType := s.scenarioData.MinSupplyType & 15
	if unit.Type >= minSupplyType {
		// headquarters can only gain supply from supply depots,
		//  not other headquarters
		minSupplyType++
	}
	if unitVisible {
		s.terrainTypes.showUnit(unit)
	}
	// keep the last friendly unit so that we can use it outside of the loop
	var supplyUnit Unit
outerLoop:
	for j := 0; j < len(s.units[unit.Side]); j++ {
		supplyUnit = s.units[unit.Side][j]
		if supplyUnit.Type < minSupplyType ||
			!supplyUnit.IsInGame || supplyUnit.SupplyLevel == 0 {
			continue
		}
		supplyX, supplyY := supplyUnit.X, supplyUnit.Y
		if unitVisible {
			s.terrainTypes.showUnit(supplyUnit)
		}
		supplyTransportBudget := s.scenarioData.MaxSupplyTransportCost
		if unit.Type == s.scenarioData.MinSupplyType&15 {
			supplyTransportBudget *= 2
		}
		for supplyTransportBudget > 0 {
			dx, dy := unit.X-supplyX, unit.Y-supplyY
			if Abs(dx)+Abs(dy) < 3 {
				supplyLevel := s.supplyLevels[unit.Side]
				if supplyLevel > 0 {
					maxResupply := Clamp(
						(supplyLevel-unit.SupplyLevel*2)/16,
						0,
						s.scenarioData.MaxResupplyAmount)
					unitResupply := s.scenarioData.UnitResupplyPerType[unit.Type]
					unitResupply = Clamp(unitResupply, 0, maxResupply)
					unit.SupplyLevel += unitResupply
					s.supplyLevels[unit.Side] = supplyLevel - unitResupply
					unit.HasSupplyLine = true
				} else {
					// not sure if it's needed...
					s.supplyLevels[unit.Side] = 0
				}
				s.terrainTypes.hideUnit(supplyUnit)
				break outerLoop
			} else {
				var x, y, speed int
				// TODO: why changing variant < 2 to variant < 1 has no effect (cost never 0? at least in dday?)
				for variant := 0; variant < 2; variant++ {
					x, y, speed = s.FindBestMoveFromTowards(supplyX, supplyY, unit.X, unit.Y, s.scenarioData.MinSupplyType, variant)
					if speed != 0 {
						break
					}
				}
				if unitVisible {
					s.sync.SendUpdate(SupplyTruckMove{supplyX / 2, supplyY, x / 2, y})
					//  function13(x, y) (show truck icon at x, y)
				}
				supplyX, supplyY = x, y
				if s.units.IsUnitOfSideAt(supplyX, supplyY, 1-unit.Side) {
					break
				}
				supplyTransportBudget -= 256 / (speed + 1)
			}
		}
		s.terrainTypes.hideUnit(supplyUnit)
		// function20: change text display mode
	}
	if unit.SupplyLevel == 0 {
		unit.Fatigue = Clamp(unit.Fatigue+64, 0, 255)
		// todo: does it really work? Aren't the last units on the list all zeroes...
		if supplyUnit.X != 0 {
			unit.ObjectiveX, unit.ObjectiveY = supplyUnit.X, supplyUnit.Y
		}
	}
	s.terrainTypes.hideUnit(unit)
	return unit
}
func (s *GameState) SaveCity(newCity City) {
	for i, city := range s.terrain.Cities {
		if city.X == newCity.X && city.Y == newCity.Y {
			s.terrain.Cities[i] = newCity
			return
		}
	}
	panic(fmt.Errorf("Cannot find city at %d,%d", newCity.X, newCity.Y))
}

func (s *GameState) HideAllUnits() {
	s.allUnitsHidden = true
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if unit.IsInGame {
				s.terrainTypes.hideUnit(unit)
			}
		}
	}
}

func (s *GameState) IsUnitVisible(unit Unit) bool {
	return unit.IsInGame && (unit.InContactWithEnemy || unit.SeenByEnemy || s.commanderFlags.PlayerCanSeeUnits[unit.Side])
}
func (s *GameState) ShowAllVisibleUnits() {
	s.allUnitsHidden = false
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if !unit.IsInGame {
				continue
			}
			if s.IsUnitVisible(unit) {
				s.terrainTypes.showUnit(unit)
			}
		}
	}
}

func (s *GameState) areUnitCoordsValid(x, y int) bool {
	// When x ==-1 x/2 is 0, which is a valid tile coordinate.
	return x >= 0 && s.terrainTypes.AreCoordsValid(x/2, y)
}

// function6
// Finds best position to move if you want to move from unitX0,unitY0 to unitX1, unitY1 with unit
// of type unitType. If variant == 0 consider only neighbour fields directly towards the goal,
// if variant == 1 look at neighbour two fields "more to the side"
func (s *GameState) FindBestMoveFromTowards(unitX0, unitY0, unitX1, unitY1, unitType, variant int) (int, int, int) {
	candX1, candY1 := s.generic.FirstNeighbourFromTowards(
		unitX0, unitY0, unitX1, unitY1, 2*variant)
	var speed1 int
	if !s.areUnitCoordsValid(candX1, candY1) {
		candX1, candY1 = unitX0, unitY0
	} else {
		terrainType1 := s.terrainTypes.terrainOrUnitTypeAt(candX1/2, candY1)
		speed1 = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[terrainType1][unitType]
	}

	candX2, candY2 := s.generic.FirstNeighbourFromTowards(
		unitX0, unitY0, unitX1, unitY1, 2*variant+1)
	var speed2 int
	if !s.areUnitCoordsValid(candX2, candY2) {
		candX2, candY2 = unitX0, unitY0
	} else {
		terrainType2 := s.terrainTypes.terrainOrUnitTypeAt(candX2/2, candY2)
		speed2 = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[terrainType2][unitType]
	}

	if speed2 > speed1-Rand(2, s.rand) {
		return candX2, candY2, speed2
	}
	return candX1, candY1, speed1
}

func (s *GameState) everyDay() bool {
	s.daysElapsed++
	var flashback FlashbackUnits
	numActiveUnits := 0
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if unit.IsInGame || !unit.HasSupplyLine {
				numActiveUnits++
			}
			if unit.IsInGame {
				flashback = append(flashback, FlashbackUnit{
					X: unit.X, Y: unit.Y, ColorPalette: unit.ColorPalette, Type: unit.Type})
			}
		}
	}
	s.numUnitsToUpdatePerTimeIncrement = (numActiveUnits*s.scenarioData.UnitUpdatesPerTimeIncrement)/128 + 1

	s.flashback = append(s.flashback, flashback)
	rnd := Rand(256, s.rand)
	if rnd < 140 {
		s.weather = int(s.scenarioData.PossibleWeather[4*(s.month/3)+rnd/35])
	}
	s.sync.SendUpdate(WeatherForecast{s.weather})
	if !s.every12Hours() {
		return false
	}
	for _, update := range s.scenarioData.DataUpdates {
		if update.Day == s.daysElapsed {
			s.scenarioData.UpdateData(update.Offset, update.Value)
		}
	}
	s.sync.SendUpdate(DailyUpdate{
		DaysRemaining: s.variants[s.selectedVariant].LengthInDays - s.daysElapsed + 1,
		SupplyLevel:   Clamp(s.supplyLevels[s.playerSide]/256, 0, 2)})
	s.ai.update = 3
	return true
}

func monthLength(month, year int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%400 == 0 {
			return 29
		}
		if year%100 == 0 {
			return 28
		}
		if year%4 == 0 {
			return 29
		}
		return 28
	}
	panic(fmt.Errorf("Unexpected month number %d", month))
}
func (s *GameState) WinningSideAndAdvantage() (winningSide int, advantage int) {
	side0Score := (1 + s.menLost[1] + s.tanksLost[1]) * s.variants[s.selectedVariant].Data3 / 8
	side1Score := 1 + s.menLost[0] + s.tanksLost[0]
	if s.game != Conflict {
		side0Score += s.citiesHeld[0] * 3
		side1Score += s.citiesHeld[1] * 3
	} else {
		side0Score += s.citiesHeld[0] * 6 / (s.scenarioData.Data174 + 1)
		side1Score += s.citiesHeld[1] * 6 / (s.scenarioData.Data174 + 1)
	}
	var score int
	if side0Score < side1Score {
		score = side1Score * 3 / side0Score
		winningSide = 1
	} else {
		score = side0Score * 3 / side1Score
		winningSide = 0
	}
	advantage = 4
	if score >= 3 {
		advantage = Clamp(score-3, 0, 4)
	}
	return
}

func (s *GameState) FinalResults() (int, int, int) {
	variant := s.variants[s.selectedVariant]
	winningSide, advantage := s.WinningSideAndAdvantage()
	var absoluteAdvantage int // a number from [1..10]
	if winningSide == 0 {
		absoluteAdvantage = advantage + 6
	} else {
		absoluteAdvantage = 5 - advantage
	}
	v73 := s.playerSide
	if s.commanderFlags.PlayerControlled[0] && s.commanderFlags.PlayerControlled[1] {
		if advantage < 6 {
			v73 = 1
		} else {
			v73 = 0
		}
	}
	var v74 int
	if v73 == 0 {
		v74 = absoluteAdvantage
	} else {
		v74 = 11 - absoluteAdvantage
	}

	criticalLocationBalance := s.criticalLocationsCaptured[0] - s.criticalLocationsCaptured[1]
	if criticalLocationBalance >= variant.CriticalLocations[0] {
		v74 = 1 + 9*(1-v73)
	}
	if -criticalLocationBalance >= variant.CriticalLocations[1] {
		v74 = 1 + 9*v73
	}
	var difficulty int
	if v73 == 0 {
		difficulty = s.options.GameBalance
	} else {
		difficulty = 4 - s.options.GameBalance
	}
	rank := Min(v74-2*difficulty+4, 12)
	return v74 - 1, difficulty, rank - 1
}
func (s *GameState) isGameOver() bool {
	variant := s.variants[s.selectedVariant]
	if s.daysElapsed >= variant.LengthInDays {
		return true
	}
	criticalLocationBalance := s.criticalLocationsCaptured[0] - s.criticalLocationsCaptured[1]
	if criticalLocationBalance >= variant.CriticalLocations[0] {
		return true
	}
	if -criticalLocationBalance >= variant.CriticalLocations[1] {
		return true
	}
	return false
}

func (s *GameState) Minute() int {
	return s.minute
}
func (s *GameState) Hour() int {
	return s.hour
}
func (s *GameState) IsNight() bool {
	return s.isNight
}
func (s *GameState) Day() int {
	return s.day
}
func (s *GameState) Month() string {
	return s.scenarioData.Months[s.month]
}
func (s *GameState) Year() int {
	return s.year
}
func (s *GameState) Weather() string {
	return s.scenarioData.Weather[s.weather]
}
func (s *GameState) MenLost(side int) int {
	return s.menLost[side] * s.scenarioData.MenMultiplier
}
func (s *GameState) TanksLost(side int) int {
	return s.tanksLost[side] * s.scenarioData.TanksMultiplier
}
func (s *GameState) CitiesHeld(side int) int {
	return s.citiesHeld[side]
}
func (s *GameState) Flashback() FlashbackHistory {
	return s.flashback
}
func (s *GameState) TerrainTypeMap() *TerrainTypeMap {
	return s.terrainTypes
}
