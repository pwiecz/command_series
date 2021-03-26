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

	commanderFlags                   *CommanderFlags
	unitsUpdated                     int
	numUnitsToUpdatePerTimeIncrement int

	flashback FlashbackHistory

	score *Score
	ai    *AI

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

func NewGameState(rand *rand.Rand, gameData *GameData, scenarioData *ScenarioData, scenarioNum, variantNum int, options *Options, sync *MessageSync) *GameState {
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
	s.scenarioData = scenarioData.Data
	s.units = scenarioData.Units
	s.terrain = scenarioData.Terrain
	s.terrainTypes = gameData.TerrainTypeMap
	s.generic = gameData.Generic
	s.hexes = gameData.Hexes
	s.generals = scenarioData.Generals
	s.variants = scenarioData.Variants
	s.selectedVariant = variantNum
	s.commanderFlags = newCommanderFlags(options)
	s.score = newScore(s.game, *variant, scenarioData.Data, s.commanderFlags, options)
	s.ai = newAI(rand, s.commanderFlags, gameData, scenarioData, s.score)
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
	saveData.CommanderFlags = s.commanderFlags.Serialize()
	saveData.SupplyLevels = [2]uint16{uint16(s.supplyLevels[0]), uint16(s.supplyLevels[1])}
	saveData.MenLost = [2]uint16{uint16(s.score.MenLost[0]), uint16(s.score.MenLost[1])}
	saveData.TanksLost = [2]uint16{uint16(s.score.TanksLost[0]), uint16(s.score.TanksLost[1])}
	saveData.CitiesHeld = [2]uint16{uint16(s.score.CitiesHeld[0]), uint16(s.score.CitiesHeld[1])}
	saveData.CriticalLocationsCaptured = [2]uint8{
		uint8(s.score.CriticalLocationsCaptured[0]),
		uint8(s.score.CriticalLocationsCaptured[1])}
	saveData.SelectedVariant = uint8(s.selectedVariant)
	saveData.UnitsUpdated = uint8(s.unitsUpdated)
	saveData.NumUnitsToUpdatePerTimeIncrement = uint8(s.numUnitsToUpdatePerTimeIncrement)
	saveData.LastUpdatedUnit = uint8(s.ai.lastUpdatedUnit)
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
	s.commanderFlags.Deserialize(saveData.CommanderFlags)
	s.supplyLevels = [2]int{int(saveData.SupplyLevels[0]), int(saveData.SupplyLevels[1])}
	s.score.MenLost = [2]int{int(saveData.MenLost[0]), int(saveData.MenLost[1])}
	s.score.TanksLost = [2]int{int(saveData.TanksLost[0]), int(saveData.TanksLost[1])}
	s.score.CitiesHeld = [2]int{int(saveData.CitiesHeld[0]), int(saveData.CitiesHeld[1])}
	s.score.CriticalLocationsCaptured = [2]int{
		int(saveData.CriticalLocationsCaptured[0]),
		int(saveData.CriticalLocationsCaptured[1])}
	s.selectedVariant = int(saveData.SelectedVariant)
	s.unitsUpdated = int(saveData.UnitsUpdated)
	s.numUnitsToUpdatePerTimeIncrement = int(saveData.NumUnitsToUpdatePerTimeIncrement)
	s.ai.lastUpdatedUnit = int(saveData.LastUpdatedUnit)
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
	s.commanderFlags.SwitchSides()
}
func (s *GameState) Update() bool {
	s.unitsUpdated++
	for ; s.unitsUpdated <= s.numUnitsToUpdatePerTimeIncrement; s.unitsUpdated++ {
		message, quit := s.ai.UpdateUnit(s.weather, s.isNight, s.sync)
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
					unit = s.ai.ResupplyUnit(unit, &s.supplyLevels, s.sync)
				}
			} else {
				if unit.HalfDaysUntilAppear == 0 {
					continue
				}
				unit.HalfDaysUntilAppear--
				if unit.HalfDaysUntilAppear == 0 {
					shouldSpawnUnit := !s.units.IsUnitAt(unit.XY) &&
						Rand(unit.InvAppearProbability, s.rand) == 0
					if city, ok := s.terrain.FindCityAt(unit.XY); ok && city.Owner != unit.Side {
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
			if unit.HasSupplyLine { // (& 136) ^ 136 > 0
				if unit.MenCount < s.scenarioData.MenCountLimit[unit.Type] {
					unit.MenCount += Rand(s.scenarioData.MenReplacementRate[unit.Side]+32, s.rand) / 32
				}
				if unit.TankCount < s.scenarioData.TankCountLimit[unit.Type] {
					unit.TankCount += Rand(s.scenarioData.TankReplacementRate[unit.Side]+32, s.rand) / 32
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
					XY: unit.XY, ColorPalette: unit.ColorPalette, Type: unit.Type})
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
		SupplyLevels:  s.supplyLevels})
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
	return s.score.WinningSideAndAdvantage()
}

func (s *GameState) FinalResults(playerSide int) (int, int, int) {
	return s.score.FinalResults(playerSide)
}
func (s *GameState) isGameOver() bool {
	variant := s.variants[s.selectedVariant]
	if s.daysElapsed >= variant.LengthInDays {
		return true
	}
	criticalLocationBalance := s.score.CriticalLocationsCaptured[0] - s.score.CriticalLocationsCaptured[1]
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
	return s.score.MenLost[side] * s.scenarioData.MenMultiplier
}
func (s *GameState) TanksLost(side int) int {
	return s.score.TanksLost[side] * s.scenarioData.TanksMultiplier
}
func (s *GameState) CitiesHeld(side int) int {
	return s.score.CitiesHeld[side]
}
func (s *GameState) Flashback() FlashbackHistory {
	return s.flashback
}
func (s *GameState) TerrainTypeMap() *TerrainTypeMap {
	return s.terrainTypes
}
