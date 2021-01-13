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
	commanderMask                    int // a bitmask defining which units are visible for which side etc.
	unitsUpdated                     int
	numUnitsToUpdatePerTimeIncrement int
	lastUpdatedUnit                  int

	menLost                   [2]int // 29927 + side*2
	tanksLost                 [2]int // 29927 + 4 + side*2
	citiesHeld                [2]int // 29927 + 13 + side*2
	criticalLocationsCaptured [2]int // 29927 + 21 + side*2
	flashback                 FlashbackHistory

	map0 [2][16][16]int // Location of troops
	map1 [2][16][16]int // Location of important objects (supply units, air wings, important cities...)
	map3 [2][16][16]int
	// Aggregated versions of map0, map1 to 4 times lower resolution.
	map2_0, map2_1 [2][4][4]int // 0x400 - two byte values

	// Side of the most recently updated unit. Used for detecting moment when we switch analysing sides.
	update int

	scenarioData    *Data
	terrain         *Terrain
	terrainMap      *Map
	generic         *Generic
	hexes           *Hexes
	units           *Units
	generals        *Generals
	variants        []Variant
	selectedVariant int
	options         *Options

	sync *MessageSync
}

func NewGameState(rand *rand.Rand, gameData *GameData, scenarioData *ScenarioData, scenarioNum, variantNum int, playerSide int, options *Options, sync *MessageSync) *GameState {
	scenario := &gameData.Scenarios[scenarioNum]
	variant := &scenarioData.Variants[variantNum]
	sunriseOffset := Abs(6-scenario.StartMonth) / 2
	s := &GameState{}
	s.rand = rand
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
	s.scenarioData = &scenarioData.Data
	s.units = &scenarioData.Units
	s.terrain = &scenarioData.Terrain
	s.terrainMap = &gameData.Map
	s.generic = &gameData.Generic
	s.hexes = &gameData.Hexes
	s.generals = &scenarioData.Generals
	s.variants = scenarioData.Variants
	s.selectedVariant = variantNum
	s.playerSide = playerSide
	s.commanderMask = calculateCommanderMask(*options)
	s.options = options
	s.sync = sync

	s.update = 3

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

// Three two-bit parts telling which side is controlled by player, can see its units, and sth.
func calculateCommanderMask(o Options) int {
	n := o.AlliedCommander.Int() + 2*o.GermanCommander.Int()
	if o.Intelligence == Limited {
		n += 56 - 4*(o.AlliedCommander.Int()*o.GermanCommander.Int()+o.AlliedCommander.Int())
	}
	return n
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

	PlayerSide    uint8
	CommanderMask uint8

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
	saveData.CommanderMask = uint8(s.commanderMask)
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
	saveData.Update = uint8(s.update)

	for i := 0; i < 2; i++ {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				saveData.Map0[i][x][y] = int16(s.map0[i][x][y])
				saveData.Map1[i][x][y] = int16(s.map1[i][x][y])
				saveData.Map3[i][x][y] = int16(s.map3[i][x][y])
			}
		}
		for x := 0; x < 4; x++ {
			for y := 0; y < 4; y++ {
				saveData.Map2_0[i][x][y] = int16(s.map2_0[i][x][y])
				saveData.Map2_1[i][x][y] = int16(s.map2_1[i][x][y])
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
	// 8: "commander mask"
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
	units, err := ParseUnits(reader, s.scenarioData.UnitTypes, s.scenarioData.UnitNames, *s.generals)
	if err != nil {
		return err
	}
	*s.units = units
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
	s.commanderMask = int(saveData.CommanderMask)
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
	s.update = int(saveData.Update)

	for i := 0; i < 2; i++ {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				s.map0[i][x][y] = int(saveData.Map0[i][x][y])
				s.map1[i][x][y] = int(saveData.Map1[i][x][y])
				s.map3[i][x][y] = int(saveData.Map3[i][x][y])
			}
		}
		for x := 0; x < 4; x++ {
			for y := 0; y < 4; y++ {
				s.map2_0[i][x][y] = int(saveData.Map2_0[i][x][y])
				s.map2_1[i][x][y] = int(saveData.Map2_1[i][x][y])
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
	// swap adjacent pairs of bits
	s.commanderMask = ((s.commanderMask & 21) << 1) + ((s.commanderMask & 42) >> 1)
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

func (s *GameState) resetMaps() {
	for side := 0; side < 2; side++ {
		for sx := 0; sx < 16; sx++ {
			for sy := 0; sy < 16; sy++ {
				s.map0[side][sx][sy] = 0
				s.map1[side][sx][sy] = 0
				s.map3[side][sx][sy] = 0
			}
		}
	}
	for side := 0; side < 2; side++ {
		for tx := 0; tx < 4; tx++ {
			for ty := 0; ty < 4; ty++ {
				s.map2_0[side][tx][ty] = 0
				s.map2_1[side][tx][ty] = 0
			}
		}
	}
}

func (s *GameState) updateUnit() (message MessageFromUnit, quit bool) {
	var mode OrderType
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
	if unit.Terrain%64 >= 48 {
		panic(fmt.Errorf("%v", unit))
	}
	var v9 int
	var arg1 int
	if unit.MenCount+unit.EquipCount < 7 ||
		unit.Fatigue == 255 {
		s.hideUnit(unit)
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
	v9 = s.countNeighbourUnits(unit.X, unit.Y, 1-unit.Side)
	if v9 == 0 {
		unit.State4 = false // &= 239
	}

	if ((unit.Side + 1) & s.commanderMask) == 0 {
		s.update = unit.Side
		// If not a local command and either objective is specified, or order is defend or move).
		if !unit.HasLocalCommand && (unit.Order == Defend || unit.Order == Move || unit.ObjectiveX != 0) { // ... maybe?
			goto l21
		} else {
			mode = unit.Order
			unit.HasLocalCommand = true // |= 32
			goto l24
		}
	} else {
		if unit.OrderBit4 {
			mode = unit.Order
			goto l24
		}
	}
	if s.update != unit.Side {
		s.reinitSmallMapsAndSuch(unit.Side)
	}
	// If unit is commanded by the computer find the best order to execute by the unit.
	{
		// v57 := sign(sign_extend([29927 + 10 + unit.side])/16)*4
		sx, sy := unit.X/8, unit.Y/4
		// Num enemy troops nearby (neighbouring "small" map fields).
		numEnemyTroops := 0
		for i := 0; i < 9; i++ {
			dx, dy := s.generic.SmallMapOffsets(i)
			if InRange(sx+dx, 0, 16) && InRange(dy+sy, 0, 16) {
				numEnemyTroops += s.map0[1-unit.Side][sx+dx][sy+dy]
			}
		}
		// If there are no enemy units in neaby "small" map and there is a supply line to unit and sth (not a special unit?) then look at the "tiny" map.
		if numEnemyTroops == 0 && unit.HasSupplyLine &&
			((s.game != Conflict && s.scenarioData.UnitScores[unit.Type]&248 == 0) ||
				(s.game == Conflict && s.scenarioData.UnitMask[unit.Type&1] == 0)) {
			tx, ty := unit.X/32, unit.Y/16
			arg1 = -17536 // 48000
			bestI := 0
			bestX, bestY := 0, 0
			for i := 0; i < 9; i++ {
				dx, dy := s.generic.TinyMapOffsets(i)
				x, y := tx+dx, ty+dy
				if !InRange(x, 0, 4) || !InRange(y, 0, 4) {
					continue
				}
				// Coords are a good target if there are more high importance objects (supply units, air wings, cities with high vp), and less good target if there are already many friendly units.
				val := (s.map2_1[unit.Side][x][y] + s.map2_1[1-unit.Side][x][y]) * 16 / Clamp(s.map2_0[unit.Side][x][y]-s.map2_0[1-unit.Side][x][y], 10, 9999)
				tmp := val * s.function26(unit.X/4, unit.Y/4, i) / 8
				if i == 0 {
					// Prioritize staying withing the same square.
					// But why, if we seem to ignore the i==0 solution later..?
					tmp *= 2
				}
				if tmp > arg1 {
					arg1 = tmp
					bestI = i
					bestX, bestY = x, y
				}
			}
			// Set unit objective to the center of the target square.
			if bestI > 0 {
				unit.TargetFormation = 0
				unit.OrderBit4 = false
				unit.Order = Reserve
				count := (unit.MenCount + unit.EquipCount + 8) / 16
				s.map2_0[unit.Side][tx][ty] = Abs(s.map2_0[unit.Side][bestX][bestY] - count)
				s.map2_0[unit.Side][bestX][bestY] += count
				unit.ObjectiveX = bestX*32 + 16 // ((v20&6)*16)|16
				if s.game == Conflict {
					unit.ObjectiveX += Rand(3, s.rand) * 2
				}
				unit.ObjectiveY = bestY*16 + 8 // ((v20&24)*2)| 8
				goto l21
			}
		}
		{
			generalMask := unit.General.Data0
			arg1 = -17536 // 48000
			//var bestI int
			var bestDx, bestDy int
			var v63 int
			temp2 := (unit.MenCount + unit.EquipCount + 4) / 8
			v61 := temp2 * Clamp(s.scenarioData.FormationMenDefence[unit.Formation], 8, 99) / 8 * s.scenarioData.TerrainMenDefence[s.terrainType(unit.Terrain)] / 8
			if s.scenarioData.UnitScores[unit.Type] > 7 {
				// special units - air wings or supply units
				temp2 = 1
				v61 = 1
			}
			// Subtract impact of the unit itself.
			s.map0[unit.Side][sx][sy] = Clamp(s.map0[unit.Side][sx][sy]-temp2, 0, 255)
			s.map3[unit.Side][sx][sy] = Clamp(s.map3[unit.Side][sx][sy]-v61, 0, 255)
			// save a copy of the unit, as we're going to modify it.
			unitCopy := unit
			for i := 1; i <= 9; i++ {
				dx, dy := s.generic.SmallMapOffsets(i - 1)
				if !InRange(sx+dx, 0, 16) || !InRange(sy+dy, 0, 16) {
					continue
				}
				v54 := 0
				v49 := 0
				v50 := 0
				v53 := 0
				unit.MenCount = s.map0[unit.Side][sx+dx][sy+dy]
				unit.EquipCount = (unit.MenCount + s.map3[unit.Side][sx+dx][sy+dy]) / 2
				v16 := s.map0[1-unit.Side][sx+dx][sy+dy] / 2
				for i := 0; i <= 7; i++ {
					ddx, ddy := s.generic.SmallMapOffsets(i + 1)
					if !InRange(sx+dx+ddx, 0, 16) || !InRange(sy+dy+ddy, 0, 16) {
						continue
					}
					v := s.map0[1-unit.Side][sx+dx+ddx][sy+dy+ddy]
					if i&4 > 0 { // diagonals(?)
						v /= 2
					}
					v16 += v
				}
				v51 := s.map0[1-unit.Side][sx+dx][sy+dy]
				temp := Reserve
				if s.map3[1-unit.Side][sx+dx][sy+dy] > 0 {
					temp = Attack
				}
				v52 := (v51 + s.map3[1-unit.Side][sx+dx][sy+dy]) / 2
				for j := 0; j < 2; j++ {
					var v48 int
					if unit.MenCount > v52 {
						v48 = Clamp((unit.MenCount+1)*8/(v52+1)-7, 0, 16)
					} else {
						v48 = -Clamp((v52+1)*8/(unit.MenCount+1)-8, 0, 16)
					}
					v48 += unit.General.Data1High + s.scenarioData.Data0High[unit.Type]
					var v55 int
					if unit.EquipCount > v16 {
						v55 = Clamp((unit.EquipCount+1)*8/(v16+1)-7, 0, 16)
					} else {
						v55 = -Clamp((v16+1)*8/(unit.EquipCount+1)-8, 0, 16)
					}
					if v48 > 0 {
						v := v48 * s.map1[1-unit.Side][sx+dx][sy+dy]
						if unit.SeenByEnemy {
							v /= 2 /* logical shift not the arithmetic one, actually) */
						}
						if generalMask&4 > 0 {
							v *= 2
						}
						if generalMask&64 > 0 {
							v /= 2
						}
						if j > 0 {
							v += s.map1[unit.Side][sx+dx][sy+dy] * 8 / unit.MenCount
						}
						v54 += v
					}
					if v55 < 0 {
						temp = Reserve
						if v51 > 0 {
							v := s.map1[unit.Side][sx+dx][sy+dy] * v55
							if generalMask&2 > 0 {
								v *= 2
							}
							if generalMask&32 > 0 {
								v /= 2
							}
							v53 += v
						}
					}
					if v48 > 0 {
						if v9 > 0 {
							temp = Attack
						}
						if v51 > 0 {
							v := v48
							if generalMask&8 > 0 {
								v *= 2
							}
							if generalMask&128 > 0 {
								v /= 2
							}
							v *= v51
							v49 += v
						}
					}
					if v55 < 0 {
						if unit.MenCount > 0 {
							temp = Defend
							v := unit.MenCount * v55
							if generalMask&1 > 0 {
								v *= 2
							}
							if generalMask&16 > 0 {
								v /= 2
							}
							v50 += v
						}
						if v55+unit.General.Data2High+s.scenarioData.Data0Low[unit.Type] < -9 {
							if j == i {
								unit.Fatigue = unit.Fatigue + 256
							}
						}
					}
					if j == 0 {
						v54 = -v54
						v53 = -v53
						v49 = -v49
						v50 = -v50
						unit.MenCount += temp2
						unit.EquipCount += v61
					}
				}
				t := v54 + v53 + v49 + v50
				if i == 1 {
					if _, ok := s.FindCity(unit.X, unit.Y); ok {
						if v51 > 0 {
							v9 = 2
						}
					}
				}
				v := s.scenarioData.UnitScores[unit.Type] & 248
				if unit.InContactWithEnemy {
					v += (unit.Fatigue/16 + unit.Fatigue/32)
				}
				if v > 7 {
					t = unit.EquipCount - v52*2
					v9 = -128
					temp = Reserve
					unit.Fatigue &= 255
				}
				t = t * s.function26(unit.X, unit.Y, i) / 8
				if i == 1 {
					v63 = t
					mode = temp
				}
				if t > arg1 {
					arg1 = t
					bestDx, bestDy = dx, dy
					//bestI = i
				}
				if i+1 > Sign(int(mode))+v9 {
					continue
				}
				break
			}
			// function18: potentially exit the whole update here
			unit = unitCopy // revert modified unit
			unit.OrderBit4 = true
			supplyUse := s.scenarioData.AvgDailySupplyUse
			if !unit.HasSupplyLine {
				supplyUse *= 2
			}
			if unit.SupplyLevel < supplyUse {
				unit2 := s.units[unit.Side][unit.SupplyUnit]
				if !unit2.IsInGame {
					unit2 = s.units[unit.Side][unit2.SupplyUnit]
				}
				unit.ObjectiveX = unit2.X
				unit.ObjectiveY = unit2.Y
				t := Move
				if v9 > 0 {
					t = Defend
				}
				unit.Order = t
				unit.TargetFormation = 0
				unit.OrderBit4 = false
				goto l21
			}
			if s.game == Conflict && s.scenarioData.UnitMask[unit.Type]&1 != 0 {
				bestDx, bestDy = 0, 0
			}
			if unit.Fatigue*4 > arg1-v63 {
				bestDx, bestDy = 0, 0
			}
			if bestDx == 0 && bestDy == 0 {
				if unit.Fatigue > 64 {
					mode = Defend
				}
				if mode == Reserve {
					mode = Defend
				}
				s.map0[unit.Side][sx][sy] += temp2
				s.map3[unit.Side][sx][sy] += v61
				// update = 13
			} else {
				if s.map0[unit.Side][sx+bestDx][sy+bestDy] > 0 {
					s.map0[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				}
				s.map3[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				unit.ObjectiveY = (((sy+bestDy)&240)/4 + Rand(2, s.rand) + 1) & 63
				unit.ObjectiveX = ((((sy+bestDy)&15)*4+Rand(2, s.rand)+1)*2 + (unit.ObjectiveY & 1)) & 127
				mode = Move
				if v9 != 0 {
					unit.Order = Defend
					goto l24
				}
			}
			unit.Order = mode
		}
	}
l24:
	unit.TargetFormation = s.function10(unit.Order, 1)
	// Find the best objective for attack.
	if mode == Attack {
		arg1 = 16000
		terrainType := s.terrainType(unit.Terrain)
		menCoeff := s.scenarioData.TerrainMenAttack[terrainType] * unit.MenCount
		equipCoeff := s.scenarioData.TerrainTankAttack[terrainType] * unit.EquipCount * s.scenarioData.Data16High[unit.Type] / 4
		coeff := (menCoeff + equipCoeff) / 8 * (255 - unit.Fatigue) / 256 * (unit.Morale + s.scenarioData.Data0High[unit.Type]*16) / 128
		temp2 := coeff * s.magicCoeff(&s.hexes.Arr144, unit.X, unit.Y, unit.Side) / 8
		v := 0
		if v9 > 0 && s.scenarioData.Data200Low[unit.Type] < 3 {
			v = 12
		}
		for i := v; i <= 18; i++ {
			arg2 := 16001
			nx := unit.X + s.generic.Dx152[i]
			ny := unit.Y + s.generic.Dy153[i]
			if unit2, ok := s.FindUnitOfSide(nx, ny, 1-unit.Side); ok {
				terrainType := s.terrainType(unit2.Terrain)
				menCoeff := s.scenarioData.TerrainMenDefence[terrainType] * unit2.MenCount
				equipCoeff := s.scenarioData.TerrainTankDefence[terrainType] * unit2.EquipCount * s.scenarioData.Data16Low[unit2.Type] / 4
				t := (menCoeff + equipCoeff) * s.scenarioData.FormationMenDefence[unit2.Formation] / 8
				w := weather
				if s.game != Conflict && s.scenarioData.UnitMask[unit.Type]&4 != 0 {
					w /= 2
				}
				if s.game == Conflict && s.scenarioData.UnitMask[unit.Type]&4 == 0 {
					w *= 2
				}
				d := s.scenarioData.UnitScores[unit2.Type] + 14 - w
				if unit2.IsUnderAttack {
					d += 4
				}
				if unit2.State2 {
					d += 8
				}
				n := t / Clamp(d, 1, 32)
				arg2 = n * s.magicCoeff(&s.hexes.Arr144, unit2.X, unit2.Y, unit2.Side) / 8 * (255 - unit2.Fatigue) / 256 * unit2.Morale / 128
			} else {
				t := s.terrainAt(nx, ny)
				if i == 18 {
					t = unit.Terrain
				}
				tt := s.terrainType(t)
				var v int
				if unit.MenCount > unit.EquipCount {
					v = s.scenarioData.TerrainMenAttack[tt]
				} else {
					v = s.scenarioData.TerrainTankAttack[tt]
				}
				if tt < 7 {
					// temporarily hide the unit while we compute sth
					s.units[unit.Side][unit.Index].IsInGame = false
					arg2 = temp2 - s.magicCoeff(&s.hexes.Arr48, nx, ny, unit.Side)*2 + v
					// unhide the unit
					s.units[unit.Side][unit.Index].IsInGame = true
				}
			}
			if i < 12 {
				arg2 *= 2
			}
			if city, ok := s.FindCity(nx, ny); ok {
				if city.Owner != unit.Side {
					if s.ContainsUnitOfSide(nx, ny, 1-unit.Side) {
						arg2 -= city.VictoryPoints
					} else {
						arg2 = -city.VictoryPoints
					}
				}
			}
			if arg2 <= arg1 {
				arg1 = arg2
				unit.ObjectiveX = nx
				unit.ObjectiveY = ny
			}
		}
	}
	if mode == Reserve {
		unit.ObjectiveX = 0
	}
	// Pick the best location to defend from among the neighbour locations.
	if mode == Defend {
		// Reset current objective.
		if unit.ObjectiveX > 0 {
			unit.ObjectiveX = unit.X
			unit.ObjectiveY = unit.Y
		}
		// temperarily hide the unit while we compute sth
		s.units[unit.Side][unit.Index].IsInGame = false
		arg1 = -17536 // 48000
		var bestI int
		// Score for i==6 (zero offset - the unit's position).
		var v_6 int
		for i := 0; i <= 6; i++ {
			ix := s.coordsToMapIndex(unit.X, unit.Y) + s.generic.MapOffsets[i]
			if !s.terrainMap.IsIndexValid(ix) {
				continue
			}
			ter := s.terrainAtIndex(ix)
			if i == 6 {
				ter = unit.Terrain
			}
			tt := s.terrainType(ter)
			var v int
			if tt == 7 {
				v = -128
			} else {
				r := s.scenarioData.TerrainMenDefence[tt]
				nx := unit.X + s.generic.Dx[i]
				ny := unit.Y + s.generic.Dy[i]
				if s.game != Conflict {
					v = r + s.magicCoeff(&s.hexes.Arr0, nx, ny, unit.Side)
				}
				if city, ok := s.FindCity(nx, ny); ok {
					if s.ContainsUnitOfSide(nx, ny, unit.Side) {
						v += city.VictoryPoints
					}
				}
				if s.scenarioData.UnitScores[unit.Type]&248 > 0 ||
					unit.Fatigue+unit.General.Data2High*4 > 96 {
					v = r + s.magicCoeff(&s.hexes.Arr96, nx, ny, unit.Side)
				}
			}
			if v >= arg1 {
				arg1 = v
				bestI = i
			}
			if i == 6 {
				v_6 = v
			}
		}
		// unhide unit
		s.units[unit.Side][unit.Index].IsInGame = true
		v := s.scenarioData.FormationMenDefence[unit.Formation] - 8
		if ((unit.Side + 1) & s.commanderMask) == 0 {
			v *= 2
		}
		if v+v_6 > arg1 {
			bestI = 6
		}
		if bestI < 6 {
			unit.ObjectiveX = unit.X + s.generic.Dx[bestI]
			unit.ObjectiveY = unit.Y + s.generic.Dy[bestI]
		} else {
			unit.TargetFormation = s.function10(unit.Order, 1)
		}
	}
	{
		// long range attack
		d32 := s.scenarioData.Data32[unit.Type]
		attackRange := (d32 & 31) * 2
		if attackRange > 0 &&
			((s.game != Conflict && (d32&8)+weather < 10) ||
				(s.game == Conflict && (d32&32)+weather < 34)) &&
			unit.Fatigue/4 < 32 {
			for i := 0; i <= 32-unit.Fatigue/4; i++ {
				unit2 := s.units[1-unit.Side][Rand(64, s.rand)]
				if ((s.game != Conflict && (unit2.IsUnderAttack || unit2.State2)) ||
					(s.game == Conflict && unit2.SeenByEnemy)) &&
					Abs(unit.X-unit2.X)/2+Abs(unit.Y-unit2.Y) <= attackRange {
					unit.ObjectiveX = unit2.X
					unit.ObjectiveY = unit2.Y
					unit.Order = Attack
					unit.Formation = s.scenarioData.Data176[0][2]
				}
			}
		}
	}
l21:
	s.update = unit.Side
	message = nil
	if unit.SupplyLevel == 0 {
		message = WeHaveExhaustedSupplies{unit}
	}
	{
		var distance int
		var sx, sy int
		// l22:
		for unitMoveBudget := 25; unitMoveBudget > 0; {
			if unit.ObjectiveX == 0 {
				break
			}
			distance = Function15_distanceToObjective(unit)
			d32 := s.scenarioData.Data32[unit.Type]
			attackRange := (d32 & 31) * 2
			if distance > 0 && distance <= attackRange && unit.Order == Attack {
				sx = unit.ObjectiveX
				sy = unit.ObjectiveY
				unit.FormationTopBit = true
				arg1 = 7
				break // goto l2
			}
			mvAdd := 0
		l5:
			if unit.ObjectiveX == unit.X && unit.ObjectiveY == unit.Y {
				unit.ObjectiveX = 0
				unit.TargetFormation = s.function10(unit.Order, 1)
				break // goto l2
			}
			unit.TargetFormation = s.function10(unit.Order, 0)
			// If unit is player controlled or its command is local
			if ((unit.Side+1)&s.commanderMask) != 0 || unit.HasLocalCommand {
				// If it's next to its objective to defend and it's in contact with enemy
				if distance == 1 && unit.Order == Defend && unit.InContactWithEnemy {
					unit.TargetFormation = s.function10(unit.Order, 1)
				}
			}
			// TODO: investigate if scope of sx, sy is not too large, and they're used where they're not supposed to.
			var moveSpeed int
			sx, sy, moveSpeed = s.FindBestMoveFromTowards(unit.X, unit.Y, unit.ObjectiveX, unit.ObjectiveY, unit.Type, mvAdd)
			if d32&64 > 0 { // in CiV artillery or mortars
				if s.game != Conflict || unit.Formation == 0 {
					sx = unit.ObjectiveX
					sy = unit.ObjectiveY
					tt := s.terrainTypeAt(sx, sy)
					moveSpeed = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit.Type]
					arg1 = tt // shouldn't have any impact
					mvAdd = 1 // it just means don't go back to l5
				} else if unit.Formation != 0 { /* Conflict */
					if s.scenarioData.UnitMask[unit.Type]&32 != 0 {
						break // goto l2
					}
				}
			}
			if s.ContainsUnitOfSide(sx, sy, unit.Side) {
				moveSpeed = 0
			}
			if s.ContainsUnitOfSide(sx, sy, 1-unit.Side) {
				moveSpeed = -1
			}
			if moveSpeed < 1 &&
				(unit.Order != Attack || moveSpeed != -1) &&
				Abs(unit.ObjectiveX-unit.X)+Abs(unit.ObjectiveY-unit.Y) > 2 &&
				mvAdd == 0 {
				mvAdd = 1
				goto l5
			}

			if moveSpeed < 1 {
				break
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
					break
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
				break
			}
			unitMoveBudget -= moveCost
			s.hideUnit(unit)
			if ((unit.Side+1)&(s.commanderMask>>2)) == 0 ||
				unit.InContactWithEnemy || unit.SeenByEnemy {
				if !s.sync.SendUpdate(UnitMove{unit, unit.X / 2, unit.Y, sx / 2, sy}) {
					quit = true
					return
				}
			}
			unit.X = sx
			unit.Y = sy
			unit.Terrain = s.terrainAt(unit.X, unit.Y)
			if unit.Terrain%64 >= 48 {
				panic(fmt.Errorf("%v", unit))
			}
			s.function29_showUnit(unit)
			if Function15_distanceToObjective(unit) == 0 {
				unit.ObjectiveX = 0
				unit.TargetFormation = s.function10(unit.Order, 1)
				if (unit.Order == Defend || unit.Order == Move) &&
					!unit.HasLocalCommand {
					message = WeHaveReachedOurObjective{unit}
				}
			}
			unit.Fatigue = Clamp(unit.Fatigue+s.scenarioData.Data173, 0, 255)
			if city, captured := s.function16(unit); captured {
				message = WeHaveCaptured{unit, city}
				break
			}
			if unitMoveBudget > 0 {
				if s.countNeighbourUnits(unit.X, unit.Y, 1-unit.Side) > 0 {
					unit.InContactWithEnemy = true
					unit.State4 = true // |= 17
				} else {
					unit.InContactWithEnemy = false // &= 254
				}
				s.function29_showUnit(unit)
			}
		}
		// l2:
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
			if unit2, ok := s.FindUnit(unit.X+s.generic.Dx[i], unit.Y+s.generic.Dy[i]); ok && unit2.Side == 1-unit.Side {
				unit2.InContactWithEnemy = true
				unit2.SeenByEnemy = true // |= 65
				s.showUnit(unit2)
				s.units[unit2.Side][unit2.Index] = unit2
				if s.scenarioData.UnitScores[unit2.Type] > 8 {
					if ((unit.Side + 1) & s.commanderMask) > 0 {
						sx = unit2.X
						sy = unit2.Y
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
		if distance == 1 && s.ContainsUnitOfSide(sx, sy, unit.Side) {
			unit.ObjectiveX = 0
			goto end
		}
		unit.TargetFormation = s.function10(unit.Order, 2)
		if unit.Fatigue > 64 || unit.SupplyLevel == 0 || !s.ContainsUnitOfSide(sx, sy, 1-unit.Side) ||
			unit.Formation != s.scenarioData.Data176[0][2] {
			goto end
		}
		if unit.FormationTopBit {
			s.hideUnit(unit)
			if !s.sync.SendUpdate(UnitMove{unit, unit.X / 2, unit.Y, sx / 2, sy}) {
				quit = true
				return
			}
			s.showUnit(unit)
			if s.game == Conflict {
				unit.InContactWithEnemy = true
				unit.SeenByEnemy = true // |= 65
			}
			// * function14 - play sound??
		} else {
			if s.scenarioData.Data32[unit.Type]&8 > 0 && weather > 3 {
				// [53767] = 0 sth with sound (silence???)
				goto end
			}
			// function27 - play some sound?
		}
		// [53767] = 0 // silence?
		if s.game != Conflict {
			unit.InContactWithEnemy = true
			unit.SeenByEnemy = true // |= 65
		}
		unit2, ok := s.FindUnitOfSide(sx, sy, 1-unit.Side)
		if !ok {
			panic("")
		}
		message = WeAreAttacking{unit, unit2, 0 /* placeholder value */, s.scenarioData.Formations}
		var attackerScore int
		{
			tt := s.terrainType(unit.Terrain)
			var menCoeff int
			if !unit.FormationTopBit {
				menCoeff = s.scenarioData.TerrainMenAttack[tt] * s.scenarioData.FormationMenAttack[unit.Formation] * unit.MenCount / 32
			}
			tankCoeff := s.scenarioData.TerrainTankAttack[tt] * s.scenarioData.FormationTankAttack[unit.Formation] * s.scenarioData.Data16High[unit.Type] / 2 * unit.EquipCount / 64
			if unit.FormationTopBit &&
				((s.game != Conflict && s.scenarioData.Data32[unit.Type]&8 > 0) ||
					(s.game == Conflict && s.scenarioData.Data32[unit.Type]&32 > 0)) {
				// long range unit
				if weather > 3 {
					goto end
				}
				tankCoeff = tankCoeff * (4 - weather) / 4
			}
			attackerScore = (menCoeff + tankCoeff) * unit.Morale / 256 * (255 - unit.Fatigue) / 128
			attackerScore = attackerScore * unit.General.Attack / 16
			attackerScore = attackerScore * s.magicCoeff(&s.hexes.Arr144, unit.X, unit.Y, unit.Side) / 8
			attackerScore++
		}

		var defenderScore int
		{
			tt2 := s.terrainType(unit2.Terrain)
			if s.scenarioData.UnitScores[unit2.Type]&248 > 0 {
				unit.State2 = true // |= 4
			}

			menCoeff := s.scenarioData.TerrainMenDefence[tt2] * s.scenarioData.FormationMenDefence[unit2.Formation] * unit2.MenCount / 32
			equipCoeff := s.scenarioData.TerrainTankAttack[tt2] * s.scenarioData.FormationTankDefence[unit2.Formation] * s.scenarioData.Data16Low[unit2.Type] / 2 * unit2.EquipCount / 64
			defenderScore = (menCoeff + equipCoeff) * unit2.Morale / 256 * (240 - unit2.Fatigue/2) / 128
			defenderScore = defenderScore * unit2.General.Defence / 16
			if unit2.SupplyLevel == 0 {
				defenderScore = defenderScore * s.scenarioData.Data167 / 8
			}
			defenderScore = defenderScore * s.magicCoeff(&s.hexes.Arr144, unit2.X, unit2.Y, 1-unit.Side) / 8
			defenderScore++
		}

		arg1 = defenderScore * 16 / attackerScore
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
				message = WeHaveMetStrongResistance{unit}
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
		tanksLost2 := Clamp((Rand(unit2.EquipCount*arg1, s.rand)+255)/512, 0, unit2.EquipCount)
		s.tanksLost[1-unit.Side] += tanksLost2
		unit2.SupplyLevel = Clamp(unit2.SupplyLevel-s.scenarioData.Data163, 0, 255)
		if s.scenarioData.UnitCanMove[unit2.Type] &&
			((s.game != Conflict && !unit.FormationTopBit) ||
				(s.game == Conflict && s.scenarioData.UnitMask[unit2.Type]&2 == 0)) &&
			arg1-s.scenarioData.Data0Low[unit2.Type]*2+unit2.Fatigue/4 > 36 {
			unit2.Morale = Abs(unit2.Morale - 1)
			oldX, oldY := unit2.X, unit2.Y
			s.hideUnit(unit2)
			if unit2.Fatigue > 128 {
				unit2SupplyUnit := s.units[unit2.Side][unit2.SupplyUnit]
				if unit2SupplyUnit.IsInGame {
					unit2.Morale = Abs(unit2.Morale - s.countNeighbourUnits(unit2.X, unit2.Y, unit.Side)*4)
					unit2.X = unit2SupplyUnit.X
					unit2.Y = unit2SupplyUnit.Y
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
					message = WeHaveBeenOverrun{unit2}
				}
			}
			bestDefence := -128
			bestX, bestY := unit2.X, unit2.Y
			for i := 0; i < 6; i++ {
				nx := unit2.X + s.generic.Dx[i]
				ny := unit2.Y + s.generic.Dy[i]
				tt := s.terrainTypeAt(nx, ny)
				r := s.scenarioData.TerrainMenDefence[tt]
				if s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit2.Type] > 0 {
					if !s.ContainsUnit(nx, ny) && !s.ContainsCity(nx, ny) {
						r += s.magicCoeff(&s.hexes.Arr96, nx, ny, 1-unit.Side) * 4
						if r > 11 && r >= bestDefence {
							bestDefence = r
							bestX, bestY = nx, ny
						}
					}
				}
			}
			if bestX != unit2.X || bestY != unit2.Y {
				unit2.X, unit2.Y = bestX, bestY // moved this up comparing to the original code

				unit2.Terrain = s.terrainAt(bestX, bestY)
				if unit2.Terrain%64 >= 48 {
					panic(fmt.Errorf("%v %d %d", unit2, bestX, bestY))
				}
			}
			if _, ok := message.(WeHaveBeenOverrun); !ok {
				if s.game != Conflict {
					s.showUnit(unit2)
					unit.ObjectiveX = unit2.X
					unit.ObjectiveY = unit2.Y
				} else {
					if ((2 - unit.Side) & (s.commanderMask >> 2)) == 0 {
						s.showUnit(unit2)
					}
					unit2.InContactWithEnemy = false
					unit2.SeenByEnemy = false // &= 190
				}
			}
			if bestX != oldX || bestY != oldY {
				// unit2 is retreating, unit one is chasing (and maybe capturing a city)
				if _, ok := message.(WeHaveBeenOverrun); !ok {
					message = WeAreRetreating{unit2}
				}
				if arg1 > 60 && (s.game != Conflict || !unit.FormationTopBit) &&
					s.magicCoeff(&s.hexes.Arr96, oldX, oldY, unit.Side) > -4 &&
					s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[s.terrainTypeAt(oldX, oldY)][unit.Type] > 0 {
					s.hideUnit(unit)
					unit.X = oldX
					unit.Y = oldY
					unit.Terrain = s.terrainAt(unit.X, unit.Y)
					if unit.Terrain%64 >= 48 {
						panic(fmt.Errorf("%v", unit.Terrain))
					}
					s.showUnit(unit)
					if city, captured := s.function16(unit); captured {
						message = WeHaveCaptured{unit, city}
					}
				}
			} else {
				message = nil
			}
			unit2.Formation = s.scenarioData.Data176[1][0]
			unit2.Order = OrderType((s.scenarioData.Data176[1][0] + 1) % 4)
			unit2.HasSupplyLine = false // |= 32
		}

		a := arg1
		if _, ok := message.(WeAreRetreating); ok { // are retreating
			a /= 2
		}
		unit2.Fatigue = Clamp(unit2.Fatigue+a, 0, 255)

		if arg1 < 24 {
			unit2.Morale = Clamp(unit2.Morale+1, 0, 250)
		}
		s.units[unit2.Side][unit2.Index] = unit2
		if attack, ok := message.(WeAreAttacking); ok {
			// update arg1 value if the message is still WeAreAttacking
			message = WeAreAttacking{attack.unit, attack.enemy, arg1, attack.formationNames}
		}
	}
end: // l3
	for unit.Formation != unit.TargetFormation {
		// changing to target formation???
		dif := Sign(unit.Formation - unit.TargetFormation)
		temp := s.scenarioData.Data216[(dif+1)*4+unit.Formation]
		if temp > Rand(15, s.rand) {
			unit.FormationTopBit = false
			unit.Formation -= dif
		}
		if temp&16 == 0 {
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

// Has unit captured a city
func (s *GameState) function16(unit Unit) (City, bool) {
	if city, ok := s.FindCity(unit.X, unit.Y); ok {
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
func (s *GameState) coordsToMapIndex(x, y int) int {
	return y*s.terrainMap.Width + x/2 - y/2
}

func (s *GameState) function29_showUnit(unit Unit) {
	if unit.InContactWithEnemy || unit.SeenByEnemy /* &65 != 0 */ ||
		((unit.Side+1)&(s.commanderMask>>2)) == 0 {
		s.showUnit(unit)
	}
}

// arr is one of arrays in Hexes
func (s *GameState) magicCoeff(arr *[6][8]int, x, y, side int) int {
	// bitmap0: has enemy unit in given direction
	// bitmap1: has friendly unit or impassable terrain in given direction
	// bitmap2: has enemy unit in neighbouring directions
	// bitmap3: has friendly unit in neighbouring directions
	// bitmap4: contains impassable terrain in given direction
	var bitmaps [5]byte
	for i := 5; i >= 0; i-- {
		bitmaps[0] <<= 1
		bitmaps[1] <<= 1
		bitmaps[4] <<= 1
		nx := x + s.generic.Dx[i]
		ny := y + s.generic.Dy[i]
		if s.ContainsUnitOfSide(nx, ny, 1-side) {
			bitmaps[0]++
		} else if s.ContainsUnitOfSide(nx, ny, side) {
			bitmaps[1]++
		} else if s.terrainTypeAt(nx, ny) >= 7 {
			bitmaps[4]++
		}
	}

	bitmaps[2] = bitmaps[0]
	bitmaps[3] = bitmaps[1]

	bitmaps[0] = rotateRight6Bits(bitmaps[0])
	bitmaps[1] = rotateRight6Bits(bitmaps[1])

	bitmaps[2] |= rotateRight6Bits(bitmaps[0])
	bitmaps[3] |= rotateRight6Bits(bitmaps[1])

	bitmaps[1] |= rotateRight6Bits(bitmaps[4])

	// Except for the first one, zero values are impossible.
	xA70B := [16]int{
		0, // 0) no unit ahead nor in neighbouring directions
		2, // 1) enemy ahead, nothing in neighbouring directions
		1, // 2) friendly unit or impassable terrain ahead, nothing in neighbouring directions
		0, // 3) (impossible)
		4, // 4) nothing ahead, enemy unit in a neighbouring direction
		2, // 5) enemy ahead and in a neighbouring direction
		1, // 6) friendly unit or impassable ahead and an enemy in a neighbouring direction
		0, // 7) (impossible)
		3, // 8) friendly unit in a neighbouring direction, nothing ahead
		2, // 9) enemy ehead, friendly unit in a neighbouring direction
		1, // 10) friendly unit/impassable terrain ahead, friendly unit in a neighbouring direction
		0, // 11) (impossible)
		5, // 12) nothing ahead and an enemy and a friendly unit in neighbouring directions
		2, // 13) enemy ahead and enemy and friendly unit in neighbouring directions
		1, // 14) friendly unit/impassable terrain ahead, enemy and friendly unit in neighbouring directions
		0, // 15) (impossible)
	}

	// xA705[i] - how many times xA80B[value of bitmaps[0-4)] occurs for all bitmap indices
	var xA705 [6]int
	for i := 0; i < 6; i++ {
		var xA6FE byte // concatenation of bitmaps[0:4) on position i
		for Y := 0; Y < 4; Y++ {
			xA6FE |= (bitmaps[Y] & 1) << Y
			bitmaps[Y] >>= 1
		}
		A := xA70B[xA6FE]
		if A == 0 && xA6FE != 0 {
			panic(fmt.Errorf("xA6FE==%d", int(xA6FE)))
		}
		xA705[A]++
	}
	xA704 := 0
	for i := 0; i < 6; i++ {
		xA704 += arr[i][xA705[i]]
	}
	return int(int8(xA704))
}

func rotateRight6Bits(num byte) byte {
	odd := num & 1
	num >>= 1
	if odd != 0 {
		num |= 0x20
	}
	return num
}

func (s *GameState) function10(order OrderType, offset int) int {
	if offset < 0 || offset >= 4 {
		panic(offset)
	}
	return s.scenarioData.Data176[int(order)][offset]
}

func Function15_distanceToObjective(unit Unit) int {
	dx := unit.ObjectiveX - unit.X
	dy := unit.ObjectiveY - unit.Y
	if Abs(dy) > Abs(dx)/2 {
		return Abs(dy)
	} else {
		return (Abs(dx) + Abs(dy) + 1) / 2
	}
}
func (s *GameState) function26(x, y int, index int) int {
	v := s.generic.Data214[((x/4)&1)*2+((y/2)&1)*18+index]
	if ((((x/2)&3)+1)&2)+((((y)&3)+1)&2) == 4 {
		v = ((index + 1) / 2) & 6
	}
	return v
}

// If the side is controlled by the computer create the strategy level maps aggregating locations and numbers of units, important locations and such.
func (s *GameState) reinitSmallMapsAndSuch(currentSide int) {
	s.resetMaps()
	// Those variables in the original code do not seem to play any role
	//v13 := 0
	//v15 := 0
	//v16 := 0
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if !unit.IsInGame || s.scenarioData.UnitMask[unit.Type]&16 != 0 {
				continue
			}
			sx, sy := unit.X/8, unit.Y/4
			if !InRange(sx, 0, 16) || !InRange(sy, 0, 16) {
				continue
			}
			if unit.Side == currentSide {
				//v15 += unit.MenCount + unit.EquipCount
				//v13 += 1
			} else {
				//v16 += unit.MenCount + unit.EquipCount
				if ((currentSide+1)&(s.commanderMask>>4)) > 0 && !unit.SeenByEnemy {
					continue
				}
			}
			v30 := unit.MenCount + unit.EquipCount
			tmp := v30 * Clamp(s.scenarioData.FormationMenDefence[unit.Formation], 8, 99) / 8
			v29 := tmp * s.scenarioData.TerrainMenDefence[s.terrainType(unit.Terrain)] / 8
			if s.scenarioData.UnitScores[unit.Type] > 7 {
				// special units - supply, air wings
				v29 = 4
				v30 = 4
			}
			s.map0[unit.Side][sx][sy] += (v30 + 4) / 8
			s.map3[unit.Side][sx][sy] = Clamp(s.map3[unit.Side][sx][sy]+(v29+4)/8, 0, 255)
			if unit.SupplyLevel-1 > s.scenarioData.AvgDailySupplyUse {
				// An "influence" of the unit on the surrounding squares on the "small" map.
				influence := s.scenarioData.UnitScores[unit.Type] / 4
				if influence > 0 {
					// Mark the "influence" of the unit on concentric circles around the unit position.
					// The influence gets smaller, further away we get.
					for radius := -1; radius <= influence; radius++ {
						// Last index on a "circle" with given radius.
						lastNeighbour := (Abs(radius) - Sign(Abs(radius))) * 4
						for i := 0; i <= lastNeighbour; i++ {
							dx, dy := s.generic.SmallMapOffsets(i)
							x, y := sx+dx, sy+dy
							if !InRange(x, 0, 16) || !InRange(y, 0, 16) {
								continue
							}
							s.map1[unit.Side][x][y] += 2
							if unit.IsUnderAttack {
								s.map1[unit.Side][x][y] += 2
							}
						}
					}
				}
			}
		}
	}
	// function18(): potentially exit the whole update here
	for _, city := range s.terrain.Cities {
		if city.Owner != 0 || city.VictoryPoints != 0 {
			sx, sy := city.X/8, city.Y/4
			v29 := city.VictoryPoints / 8
			if v29 > 0 {
				// Mark the "influence" of the city on concentric circles around the city position.
				// The influence gets smaller, further away we get.
				s.map3[city.Owner][sx][sy]++
				for i := 1; i <= v29; i++ {
					for j := 0; j <= (i-1)*4; j++ {
						dx, dy := s.generic.SmallMapOffsets(j)
						x, y := sx+dx, sy+dy
						if !InRange(x, 0, 16) || !InRange(y, 0, 16) {
							continue
						}
						s.map1[city.Owner][x][y] += 2
					}
				}
			}
		}
	}
	// function18(): potentially exit the whole update here
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			s.map1[0][x][y] = s.map1[0][x][y] * s.terrain.Coeffs[x][y] / 8
			s.map1[1][x][y] = s.map1[1][x][y] * s.terrain.Coeffs[x][y] / 8
		}
	}
	// function18(): potentially exit the whole update here
	for side := 0; side < 2; side++ {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				s.map2_0[side][x/4][y/4] += s.map0[side][x][y]
				s.map2_1[side][x/4][y/4] += s.map1[side][x][y]
			}
		}
	}
	// function18(): potentially exit the whole update here
}

func (s *GameState) countNeighbourUnits(x, y, side int) int {
	num := 0
	for _, unit := range s.units[side] {
		if !unit.IsInGame {
			continue
		}
		if Abs(unit.X-x)+Abs(2*(unit.Y-y)) < 4 { // TODO: double check it
			num++
		}
	}
	return num
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
					shouldSpawnUnit := !s.ContainsUnit(unit.X, unit.Y) &&
						Rand(unit.InvAppearProbability, s.rand) == 0
					if city, ok := s.FindCity(unit.X, unit.Y); ok && city.Owner != unit.Side {
						shouldSpawnUnit = false
					}
					if shouldSpawnUnit {
						unit.IsInGame = true
						unit.Terrain = s.terrainAt(unit.X, unit.Y)
						if unit.Terrain%64 >= 48 {
							panic(fmt.Errorf("%v", unit))
						}
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
	unitVisible := ((unit.Side + 1) & (s.commanderMask >> 2)) == 0
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
		s.showUnit(unit)
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
			s.showUnit(supplyUnit)
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
				s.hideUnit(supplyUnit)
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
				if s.ContainsUnitOfSide(supplyX, supplyY, 1-unit.Side) {
					break
				}
				supplyTransportBudget -= 256 / (speed + 1)
			}
		}
		s.hideUnit(supplyUnit)
		// function20: change text display mode
	}
	if unit.SupplyLevel == 0 {
		unit.Fatigue = Clamp(unit.Fatigue+64, 0, 255)
		// todo: does it really work? Aren't the last units on the list all zeroes...
		if supplyUnit.X != 0 {
			unit.ObjectiveX = supplyUnit.X
			unit.ObjectiveY = supplyUnit.Y
		}
	}
	s.hideUnit(unit)
	return unit
}

func (s *GameState) ContainsUnit(x, y int) bool {
	return s.ContainsUnitOfSide(x, y, 0) ||
		s.ContainsUnitOfSide(x, y, 1)
}
func (s *GameState) ContainsUnitOfSide(x, y, side int) bool {
	sideUnits := s.units[side]
	for i := range sideUnits {
		if sideUnits[i].IsInGame && sideUnits[i].X == x && sideUnits[i].Y == y {
			return true
		}
	}
	return false
}
func (s *GameState) ContainsCity(x, y int) bool {
	for _, city := range s.terrain.Cities {
		if city.VictoryPoints > 0 && city.X == x && city.Y == y {
			return true
		}
	}
	return false
}

func (s *GameState) FindUnit(x, y int) (Unit, bool) {
	return s.FindUnitAtMapCoords(x/2, y)
}
func (s *GameState) FindUnitAtMapCoords(x, y int) (Unit, bool) {
	for _, sideUnits := range s.units {
		for i := range sideUnits {
			if sideUnits[i].IsInGame && sideUnits[i].X/2 == x && sideUnits[i].Y == y {
				return sideUnits[i], true
			}
		}
	}
	return Unit{}, false
}
func (s *GameState) FindUnitOfSide(x, y, side int) (Unit, bool) {
	sideUnits := s.units[side]
	for i := range sideUnits {
		if sideUnits[i].IsInGame && sideUnits[i].X == x && sideUnits[i].Y == y {
			return sideUnits[i], true
		}
	}
	return Unit{}, false
}
func (s *GameState) FindCity(x, y int) (City, bool) {
	return s.FindCityAtMapCoords(x/2, y)
}
func (s *GameState) FindCityAtMapCoords(x, y int) (City, bool) {
	for _, city := range s.terrain.Cities {
		if city.VictoryPoints > 0 && city.X/2 == x && city.Y == y {
			return city, true
		}
	}
	return City{}, false
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

// function17
func (s *GameState) terrainType(terrain byte) int {
	return s.generic.TerrainTypes[terrain&63]
}

func (s *GameState) terrainTypeAt(x, y int) int {
	return s.terrainType(s.terrainAt(x, y))
}
func (s *GameState) terrainTypeAtIndex(ix int) int {
	return s.terrainType(s.terrainAtIndex(ix))
}
func (s *GameState) terrainAtIndex(ix int) byte {
	return s.terrainMap.GetTileAtIndex(ix)
}
func (s *GameState) terrainAt(x, y int) byte {
	ix := s.coordsToMapIndex(x, y)
	if !s.terrainMap.IsIndexValid(ix) {
		return 0
	}
	return s.terrainMap.GetTileAtIndex(ix)
}

func (s *GameState) showUnit(unit Unit) {
	ix := s.coordsToMapIndex(unit.X, unit.Y)
	s.terrainMap.SetTileAtIndex(ix, byte(unit.Type+unit.ColorPalette*16))
}
func (s *GameState) hideUnit(unit Unit) {
	ix := s.coordsToMapIndex(unit.X, unit.Y)
	if unit.Terrain%64 >= 48 {
		panic(fmt.Errorf("%v", unit))
	}
	s.terrainMap.SetTileAtIndex(ix, unit.Terrain)
}
func (s *GameState) HideAllUnits() {
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if unit.IsInGame {
				s.hideUnit(unit)
			}
		}
	}
}

func (s *GameState) IsUnitVisible(unit Unit) bool {
	return unit.IsInGame && (unit.InContactWithEnemy || unit.SeenByEnemy || ((unit.Side+1)&(s.commanderMask>>2)) == 0)
}
func (s *GameState) ShowAllVisibleUnits() {
	for _, sideUnits := range s.units {
		for i, unit := range sideUnits {
			if !unit.IsInGame {
				continue
			}
			sideUnits[i].Terrain = s.terrainAt(unit.X, unit.Y)
			if sideUnits[i].Terrain%64 >= 48 {
				panic(fmt.Errorf("%v", sideUnits[i]))
			}
			if s.IsUnitVisible(unit) {
				s.showUnit(unit)
			}
		}
	}
}

// function6
// Finds best position to move if you want to move from unitX0,unitY0 to unitX1, unitY1 with unit
// of type unitType. If variant == 0 consider only neighbour fields directly towards the goal,
// if variant == 1 look at neighbour two fields "more to the side"
func (s *GameState) FindBestMoveFromTowards(unitX0, unitY0, unitX1, unitY1, unitType, variant int) (int, int, int) {
	dx, dy := unitX1-unitX0, unitY1-unitY0

	neighbour1 := s.generic.DxDyToNeighbour(dx, dy, 2*variant)
	candX1 := unitX0 + s.generic.Dx[neighbour1]
	candY1 := unitY0 + s.generic.Dy[neighbour1]
	var speed1 int
	if !s.terrainMap.AreCoordsValid(candX1/2, candY1) {
		candX1, candY1 = unitX0, unitY0
	} else {
		terrainType1 := s.terrainTypeAt(candX1, candY1)
		speed1 = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[terrainType1][unitType]
	}

	neighbour2 := s.generic.DxDyToNeighbour(dx, dy, 2*variant+1)
	candX2 := unitX0 + s.generic.Dx[neighbour2]
	candY2 := unitY0 + s.generic.Dy[neighbour2]
	var speed2 int
	if !s.terrainMap.AreCoordsValid(candX2/2, candY2) {
		candX2, candY2 = unitX0, unitY0
	} else {
		terrainType2 := s.terrainTypeAt(candX2, candY2)
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
					X: unit.X, Y: unit.Y, ColorPalette: unit.ColorPalette, Type: unit.Type, Terrain: unit.Terrain})
			}
		}
	}
	s.numUnitsToUpdatePerTimeIncrement = (numActiveUnits*s.scenarioData.UnitUpdatesPerTimeIncrement)/128 + 1

	s.flashback = append(s.flashback, flashback)
	// todo: save today's map for flashback
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
	s.update = 3
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
	if s.commanderMask%4 == 0 { // if a two-player game?
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
