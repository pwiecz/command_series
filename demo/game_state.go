package main

import "fmt"
import "strings"

import "github.com/pwiecz/command_series/data"

type GameState struct {
	minute       int
	hour         int
	daysElapsed  int
	day          int /* 0-based */
	month        int /* 0-based */
	year         int
	weather      int
	supplyLevels [2]int

	playerSide                       int // remove it from here
	unitsUpdated                     int
	numUnitsToUpdatePerTimeIncrement int
	isNight                          bool
	lastUpdatedUnit                  int

	menLost                   [2]int // 29927 + side*2
	tanksLost                 [2]int // 29927 + 4 + side*2
	citiesHeld                [2]int // 29927 + 13 + side*2
	criticalLocationsCaptured [2]int // 29927 + 21 + side*2
	flashback                 [][]data.FlashbackUnit

	map0           [2][16][16]int // 0
	map1           [2][16][16]int // 0x200
	map2_0, map2_1 [2][4][4]int   // 0x400 - two byte values
	map2_2, map2_3 [2][16]int
	map3           [2][16][16]int // 0x600
	update         int

	scenarioData *data.ScenarioData
	terrain      *data.Terrain
	terrainMap   *data.Map
	generic      *data.Generic
	hexes        *data.Hexes
	units        [2][]data.Unit
	generals     [2][]data.General
	variant      *data.Variant
	options      Options

	sync *MessageSync
}

func NewGameState(scenario *data.Scenario, scenarioData *data.ScenarioData, variant *data.Variant, variantNum int, units [2][]data.Unit, terrain *data.Terrain, terrainMap *data.Map, generic *data.Generic, hexes *data.Hexes, generals [2][]data.General, gameOptions Options, sync *MessageSync) *GameState {
	s := &GameState{}
	s.minute = scenario.StartMinute
	s.hour = scenario.StartHour
	s.day = scenario.StartDay
	s.month = scenario.StartMonth
	s.year = scenario.StartYear
	s.weather = scenario.StartWeather
	s.supplyLevels = scenario.StartSupplyLevels
	s.numUnitsToUpdatePerTimeIncrement = scenarioData.UnitUpdatesPerTimeIncrement / 2
	s.lastUpdatedUnit = 127
	s.citiesHeld = variant.CitiesHeld
	s.scenarioData = scenarioData
	s.units = units
	s.terrain = terrain
	s.terrainMap = terrainMap
	s.generic = generic
	s.hexes = hexes
	s.generals = generals
	s.variant = variant
	s.options = gameOptions
	s.sync = sync

	for side, sideUnits := range s.units {
		for i, unit := range sideUnits {
			if unit.VariantBitmap&(1<<variantNum) != 0 {
				unit.State = 0
				unit.HalfDaysUntilAppear = 0
			}
			unit.VariantBitmap = 0 // not really needed
			if side == 0 && gameOptions.GameBalance > 2 {
				unit.Morale = (3 + gameOptions.GameBalance) * unit.Morale / 5
			} else if side == 1 && gameOptions.GameBalance < 2 {
				unit.Morale = (7 - gameOptions.GameBalance) * unit.Morale / 5
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
	s.showAllVisibleUnits()

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
		if !s.everyHour() {
			return false
		}
	}
	if s.hour >= 24 {
		s.hour = 0
		s.day++
		if !s.everyDay() {
			return false
		}
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
	if s.hour == 18 && s.minute == 0 {
		fmt.Println(s.dateTimeString())
		fmt.Println(s.statusReport())
		if s.isGameOver() {
			s.sync.SendUpdate(GameOver{s.finalResults()})
			return false
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
	var mode data.OrderType
	weather := s.weather
	if s.isNight {
		weather += 8
	}
nextUnit:
	s.lastUpdatedUnit = (s.lastUpdatedUnit + 1) % 128
	unit := s.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2]
	if unit.State&128 == 0 {
		goto nextUnit
	}
	var v9 int
	var arg1 int
	if unit.MenCount+unit.EquipCount < 7 ||
		unit.Fatigue == 255 {
		s.hideUnit(unit)
		message = WeMustSurrender{unit}
		unit.State = 0
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
		unit.State &= 239
	}

	if s.options.IsPlayerControlled(unit.Side) {
		s.update = unit.Side
		if unit.Order == data.Defend || unit.Order == data.Move || unit.ObjectiveX != 0 || unit.State&32 != 0 { // ... maybe?
			goto l21
		} else {
			mode = unit.Order
			unit.State |= 32
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
	{
		// v57 := sign(sign_extend([29927 + 10 + unit.side])/16)*4
		sx, sy := unit.X/8, unit.Y/4
		temp := 0
		for i := 0; i < 9; i++ {
			dx, dy := s.generic.SmallMapOffsets(i)
			if InRange(sx+dx, 0, 16) && InRange(dy+sy, 0, 16) {
				temp += s.map0[1-unit.Side][sx+dx][sy+dy]
			}
		}
		// in CiV the second term is s.scenarioData.UnitMask[unit.Type]&1 == 0
		if temp == 0 && s.scenarioData.UnitScores[unit.Type]&248 == 0 && unit.State&8 == 0 {
			tx, ty := unit.X/32, unit.Y/16
			//unit.X /= 4
			//unit.Y /= 4
			arg1 = -17536 // 48000
			bestI := 0
			bestX, bestY := 0, 0
			for i := 0; i < 9; i++ {
				//t := s.generic.Data44[i]
				//if !InRange(Sign(int(int8((t&6)*32)))*8+unit.X+1, 1, 33) {
				//	panic("")
				//}
				//if !InRange(Sign((int(int8(t))+2)/8)*4+unit.Y+1, 1, 17) {
				//	panic("")
				//}
				dx, dy := s.generic.TinyMapOffsets(i)
				x, y := tx+dx, ty+dy
				if !InRange(x, 0, 4) || !InRange(y, 0, 4) {
					continue
				}
				val := (s.map2_1[unit.Side][x][y] + s.map2_1[1-unit.Side][x][y]) * 16 / Clamp(s.map2_0[unit.Side][x][y]-s.map2_0[1-unit.Side][x][y], 10, 9999)
				tmp := val * s.function26(unit.X/4, unit.Y/4, i) / 8
				if i == 0 {
					tmp *= 2
				}
				if tmp > arg1 {
					arg1 = tmp
					bestI = i
					bestX, bestY = x, y
				}
			}
			// reload the unit as its coords have been overwritten
			//unit = s.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2]
			if bestI > 0 {
				unit.TargetFormation = 0
				unit.OrderBit4 = false
				unit.Order = data.Reserve
				temp = (unit.MenCount + unit.EquipCount + 8) / 16
				s.map2_0[unit.Side][tx][ty] = Abs(s.map2_0[unit.Side][bestX][bestY] - temp)
				s.map2_0[unit.Side][bestX][bestY] += temp
				unit.ObjectiveX = bestX*32 + 16 // ((v20&6)*16)|16
				if false /* CiV */ {
					unit.ObjectiveX += Rand(3) * 2
				}
				unit.ObjectiveY = bestY*16 + 8 // ((v20&24)*2)| 8
				goto l21
			}
		}
		{
			v58 := s.generals[unit.Side][unit.GeneralIndex].Data0
			arg1 = -17536 // 0xBB80
			//var bestI int
			var bestDx, bestDy int
			var v63 int
			temp2 := (unit.MenCount + unit.EquipCount + 4) / 8
			v61 := temp2 * Clamp(s.scenarioData.FormationMenDefence[unit.Formation], 8, 99) / 8 * s.scenarioData.TerrainMenDefence[s.terrainTypeAt(unit.X, unit.Y)] / 8
			if s.scenarioData.UnitScores[unit.Type] > 7 {
				// special units - air wings or supply units
				temp2 = 1
				v61 = 1
			}
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
				temp := data.Reserve
				if s.map3[1-unit.Side][sx+dx][sy+dy] > 0 {
					temp = data.Attack
				}
				v52 := (v51 + s.map3[1-unit.Side][sx+dx][sy+dy]) / 2
				for j := 0; j < 2; j++ {
					var v48 int
					if unit.MenCount > v52 {
						v48 = Clamp((unit.MenCount+1)*8/(v52+1)-7, 0, 16)
					} else {
						v48 = -Clamp((v52+1)*8/(unit.MenCount+1)-8, 0, 16)
					}
					v48 += s.generals[unit.Side][unit.GeneralIndex].Data1High + s.scenarioData.Data0High[unit.Type]
					var v55 int
					if unit.EquipCount > v16 {
						v55 = Clamp((unit.EquipCount+1)*8/(v16+1)-7, 0, 16)
					} else {
						v55 = -Clamp((v16+1)*8/(unit.EquipCount+1)-8, 0, 16)
					}
					if v48 > 0 {
						v := v48 * s.map1[1-unit.Side][sx+dx][sy+dy]
						if unit.State&64 > 0 {
							v /= 2 /* logical shift not the arithmetic one, actually) */
						}
						if v58&4 > 0 {
							v *= 2
						}
						if v58&64 > 0 {
							v /= 2
						}
						if j > 0 {
							v += s.map1[unit.Side][sx+dx][sy+dy] * 8 / unit.MenCount
						}
						v54 += v
					}
					if v55 < 0 {
						temp = data.Reserve
						if v51 > 0 {
							v := s.map1[unit.Side][sx+dx][sy+dy] * v55
							if v58&2 > 0 {
								v *= 2
							}
							if v58&32 > 0 {
								v /= 2
							}
							v53 += v
						}
					}
					if v48 > 0 {
						if v9 > 0 {
							temp = data.Attack
						}
						if v51 > 0 {
							v := v48
							if v58&8 > 0 {
								v *= 2
							}
							if v58&128 > 0 {
								v /= 2
							}
							v *= v51
							v49 += v
						}
					}
					if v55 < 0 {
						if unit.MenCount > 0 {
							temp = data.Defend
							v := unit.MenCount * v55
							if v58&1 > 0 {
								v *= 2
							}
							if v58&16 > 0 {
								v /= 2
							}
							v50 += v
						}
						if v55+s.generals[unit.Side][unit.GeneralIndex].Data2High+s.scenarioData.Data0Low[unit.Type] < -9 {
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
					if city, ok := s.FindCity(unit.X, unit.Y); ok {
						if city.VictoryPoints > 0 {
							if v51 > 0 {
								v9 = 2
							}
						}
					}
				}
				v := s.scenarioData.UnitScores[unit.Type] & 248
				if unit.State&1 > 0 {
					v += (unit.Fatigue/16 + unit.Fatigue/32)
				}
				if v > 7 {
					t = unit.EquipCount - v52*2
					v9 = -128
					temp = data.Reserve
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
			// there is no supply line to unit
			if unit.State&8 > 0 {
				supplyUse *= 2
			}
			if unit.SupplyLevel < supplyUse {
				unit2 := s.units[unit.Side][unit.SupplyUnit]
				if unit2.State&128 == 0 {
					unit2 = s.units[unit.Side][unit2.SupplyUnit]
				}
				unit.ObjectiveX = unit2.X
				unit.ObjectiveY = unit2.Y
				t := data.Move
				if v9 > 0 {
					t = data.Defend
				}
				unit.Order = t
				unit.TargetFormation = 0
				unit.OrderBit4 = false
				goto l21
			}
			if false /* CiV */ && s.scenarioData.UnitMask[unit.Type]&1 != 0 {
				bestDx, bestDy = 0, 0
			}
			if unit.Fatigue*4 > arg1-v63 {
				bestDx, bestDy = 0, 0
			}
			if bestDx == 0 && bestDy == 0 {
				if unit.Fatigue > 64 {
					mode = data.Defend
				}
				if mode == data.Reserve {
					mode = data.Defend
				}
				s.map0[unit.Side][sx][sy] += temp2
				s.map3[unit.Side][sx][sy] += v61
				// update = 13
			} else {
				if s.map0[unit.Side][sx+bestDx][sy+bestDy] > 0 {
					s.map0[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				}
				s.map3[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				unit.ObjectiveY = (((sy+bestDy)&240)/4 + Rand(2) + 1) & 63
				unit.ObjectiveX = ((((sy+bestDy)&15)*4+Rand(2)+1)*2 + (unit.ObjectiveY & 1)) & 127
				mode = data.Move
				if v9 != 0 {
					unit.Order = data.Defend
					goto l24
				}
			}
			unit.Order = mode
		}
	}
l24:
	unit.TargetFormation = s.function10(unit.Order, 1)
	if mode == data.Attack {
		arg1 = 16000
		terrainType := s.terrainType(unit.Terrain)
		menCoeff := s.scenarioData.TerrainMenAttack[terrainType] * unit.MenCount
		equipCoeff := s.scenarioData.TerrainTankAttack[terrainType] * unit.EquipCount * s.scenarioData.Data16High[unit.Type] / 4
		coeff := (menCoeff + equipCoeff) / 8 * (255 - unit.Fatigue) / 256 * (unit.Morale + s.scenarioData.Data0High[unit.Type]*16) / 128
		temp2 := coeff * s.magicCoeff(s.hexes.Arr144[:], unit.X, unit.Y, unit.Side) / 8
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
				if true /* !CiV */ && s.scenarioData.UnitMask[unit.Type]&4 != 0 {
					w /= 2
				}
				if false /* CiV */ && s.scenarioData.UnitMask[unit.Type]&4 == 0 {
					w *= 2
				}
				d := s.scenarioData.UnitScores[unit2.Type] + int((unit2.State&6)*2) + 14 - w
				n := t / Clamp(d, 1, 32)
				arg2 = n * s.magicCoeff(s.hexes.Arr144[:], unit2.X, unit2.Y, unit2.Side) / 8 * (255 - unit2.Fatigue) / 256 * unit2.Morale / 128
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
					// temperarily hide the unit while we compute sth
					s.units[unit.Side][unit.Index].State = unit.State & 127
					arg2 = temp2 - s.magicCoeff(s.hexes.Arr48[:], nx, ny, unit.Side)*2 + v
					// unhide the unit
					s.units[unit.Side][unit.Index].State = unit.State
				}
			}
			if i < 12 {
				arg2 *= 2
			}
			if city, ok := s.FindCity(nx, ny); ok {
				if city.Owner != unit.Side && city.VictoryPoints > 0 {
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
	if mode == data.Reserve {
		unit.ObjectiveX = 0
	}
	if mode == data.Defend {
		if unit.ObjectiveX > 0 {
			unit.ObjectiveX = unit.X
			unit.ObjectiveY = unit.Y
		}
		// temperarily hide the unit while we compute sth
		s.units[unit.Side][unit.Index].State &= 127
		arg1 = -17536 // 48000
		var bestI int
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
				if true /* !CiV */ {
					v = r + s.magicCoeff(s.hexes.Arr0[:], nx, ny, unit.Side)
				}
				if city, ok := s.FindCity(nx, ny); ok {
					if s.ContainsUnitOfSide(nx, ny, unit.Side) {
						v += city.VictoryPoints
					}
				}
				if (s.scenarioData.UnitScores[unit.Type]&248)+Sign(unit.Fatigue-96+s.generals[unit.Side][unit.GeneralIndex].Data2High*4) > 0 {
					v = r + s.magicCoeff(s.hexes.Arr96[:], nx, ny, unit.Side)
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
		s.units[unit.Side][unit.Index].State |= 128
		v := s.scenarioData.FormationMenDefence[unit.Formation] - 8
		if s.options.IsPlayerControlled(unit.Side) {
			v *= 2
		}
		if v > arg1-v_6 {
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
		// in CiV the weather term is (d32&32)+weather < 34
		if attackRange > 0 && (d32&8)+weather < 10 && unit.Fatigue/4 < 32 {
			for i := 0; i <= 32-unit.Fatigue/4; i++ {
				unit2 := s.units[1-unit.Side][Rand(64)]
				// in CiV first term says unit2.State&64 > 0
				if unit2.State&6 > 0 && Abs(unit.X-unit2.X)/2+Abs(unit.Y-unit2.Y) <= attackRange {
					unit.ObjectiveX = unit2.X
					unit.ObjectiveY = unit2.Y
					unit.Order = unit.Order | 2
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
		v57 := 25 // unit's move "budget"
		var distance int
		var sx, sy int
		for { // l22:
			if unit.ObjectiveX == 0 {
				break
			}
			distance = Function15_distanceToObjective(unit)
			d32 := s.scenarioData.Data32[unit.Type]
			attackRange := (d32 & 31) * 2
			if distance > 0 && distance <= attackRange && unit.Order == data.Attack {
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
			if s.options.IsPlayerControlled(unit.Side) || unit.State&32 > 0 {
				if distance == 1 && unit.Order == data.Defend && unit.State&1 > 0 {
					unit.TargetFormation = s.function10(unit.Order, 1)
				}
			}
			temp := function8(unit.ObjectiveX-unit.X, unit.ObjectiveY-unit.Y)
			offset, moveCost := s.function6(temp, mvAdd, unit.X, unit.Y, unit.Type)
			sx = unit.X + s.generic.Dx[offset]
			sy = unit.Y + s.generic.Dy[offset]
			if d32&64 > 0 { // in CiV artillery or mortars
				if true /* !CiV */ || unit.Formation == 0 {
					sx = unit.ObjectiveX
					sy = unit.ObjectiveY
					tt := s.terrainTypeAt(sx, sy)
					moveCost = s.scenarioData.MoveCostPerTerrainTypesAndUnit[tt][unit.Type]
					mvAdd = 1
				} else if unit.Formation != 0 { /* CiV */
					if s.scenarioData.UnitMask[unit.Type]&32 != 0 {
						break // goto l2
					}
				}
			}
			if s.ContainsUnitOfSide(sx, sy, unit.Side) {
				moveCost = 0
			}
			if s.ContainsUnitOfSide(sx, sy, 1-unit.Side) {
				moveCost = -1
			}
			if moveCost < 1 &&
				(unit.Order != data.Attack || moveCost != -1) &&
				Abs(unit.ObjectiveX-unit.X)+Abs(unit.ObjectiveY-unit.Y) > 2 &&
				mvAdd == 0 {
				mvAdd = 2
				goto l5
			}

			if moveCost < 1 {
				break
			}
			v := s.scenarioData.Data192[unit.Formation] * moveCost / 8
			if unit.State&16 != 0 {
				v *= s.scenarioData.Data200Low[unit.Type]
				v /= 8
			}
			v *= (512 - unit.Fatigue) / 32
			v = v * s.generals[unit.Side][unit.GeneralIndex].Data3Low / 16
			if unit.SupplyLevel == 0 {
				v /= 2
			}
			if false /* DitD || CiV */ {
				temp = v
				if v == 0 {
					break
				}
			}
			w := 1024
			if false /* CiV */ {
				w = 1023
			}
			if s.scenarioData.UnitMask[unit.Type]&4 != 0 {
				if true /* ! CiV */ {
					w += weather * 128
				} else /* CiV */ {
					w += weather * 256
				}
			}
			w *= 8
			if true /* !DitD && !CiV */ {
				temp = w / (v + 1)
			} else /* DitD || CiV */ {
				temp = w / v
			}
			if temp > v57 && Rand(temp) > v57 {
				break
			}
			v57 -= temp
			s.hideUnit(unit)
			if s.options.IsPlayerControlled(unit.Side) ||
				s.options.Intelligence == Full || unit.State&65 > 0 {
				if !s.sync.SendUpdate(UnitMove{unit, unit.X / 2, unit.Y, sx / 2, sy}) {
					quit = true
					return
				}
			}
			unit.X = sx
			unit.Y = sy
			unit.Terrain = s.terrainAt(unit.X, unit.Y)
			s.function29_showUnit(unit)
			if Function15_distanceToObjective(unit) == 0 {
				unit.ObjectiveX = 0
				unit.TargetFormation = s.function10(unit.Order, 1)
				if (unit.Order == data.Defend || unit.Order == data.Move) &&
					unit.State&32 == 0 {
					message = WeHaveReachedOurObjective{unit}
				}
			}
			unit.Fatigue = Clamp(unit.Fatigue+s.scenarioData.Data173, 0, 255)
			if city, captured := s.function16(unit); captured {
				message = WeHaveCaptured{unit, city}
				break
			}
			if v57 > 0 {
				if s.countNeighbourUnits(unit.X, unit.Y, 1-unit.Side) > 0 {
					unit.State |= 17
				} else {
					unit.State &= 254
				}
				s.function29_showUnit(unit)
				continue
			}
			break
		}
		// l2:
		unit.SupplyLevel = Clamp(unit.SupplyLevel-2, 0, 255)
		wasInContactWithEnemy := unit.State&1 != 0
		if Rand(s.scenarioData.Data252[unit.Side]) > 0 {
			unit.State &= 232
		} else {
			unit.State &= 168
		}
		if false /* CiV */ && Rand(s.scenarioData.Data175)/8 > 0 {
			unit.State |= 64
		}
		for i := 0; i < 6; i++ {
			if unit2, ok := s.FindUnit(unit.X+s.generic.Dx[i], unit.Y+s.generic.Dy[i]); ok && unit2.Side == 1-unit.Side {
				s.showUnit(unit2)
				unit2.State = unit2.State | 65
				s.units[unit2.Side][unit2.Index] = unit2
				if s.scenarioData.UnitScores[unit2.Type] > 8 {
					if s.options.IsPlayerControlled(unit.Side) {
						sx = unit2.X
						sy = unit2.Y
						unit.Order = data.Attack
						arg1 = 7
						// arg2 = i
					}
					if s.scenarioData.UnitMask[unit2.Type]&128 == 0 {
						unit.State = unit.State | 16
					}
					if s.scenarioData.UnitCanMove[unit2.Type] {
						unit.State = unit.State | 65
						if !wasInContactWithEnemy {
							message = WeAreInContactWithEnemy{unit}
						}
					}
				}
			}
		}
		s.function29_showUnit(unit)
		//	l11:
		if unit.ObjectiveX == 0 || unit.Order != data.Attack || arg1 < 7 {
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
			if false /* CiV */ {
				unit.State |= 65
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
		if true /* !CiV */ {
			unit.State |= 65
		}
		unit2, ok := s.FindUnitOfSide(sx, sy, 1-unit.Side)
		if !ok {
			panic("")
		}
		arg1 = s.terrainType(unit.Terrain)
		message = WeAreAttacking{unit, unit2, arg1 /* placeholder value */, s.scenarioData.Formations}
		v := s.scenarioData.TerrainMenAttack[arg1] * s.scenarioData.FormationMenAttack[unit.Formation] * unit.MenCount / 32
		if unit.FormationTopBit {
			v = 0
		}
		v2 := s.scenarioData.TerrainTankAttack[arg1] * s.scenarioData.FormationTankAttack[unit.Formation] * s.scenarioData.Data16High[unit.Type] / 2 * unit.EquipCount / 64
		// in CiV the second term is s.scenarioData.Data32[unit.Type]&32 > 0
		if unit.FormationTopBit && s.scenarioData.Data32[unit.Type]&8 > 0 {
			if weather > 3 {
				goto end
			}
			v2 = v2 * (4 - weather) / 4
		}
		v = (v + v2) * unit.Morale / 256 * (255 - unit.Fatigue) / 128
		v = v * s.generals[unit.Side][unit.GeneralIndex].Data1Low / 16
		v = v * s.magicCoeff(s.hexes.Arr144[:], unit.X, unit.Y, unit.Side) / 8
		v++
		tt2 := s.terrainType(unit2.Terrain)
		if s.scenarioData.UnitScores[unit2.Type]&248 > 0 {
			unit.State |= 4
		}
		menCoeff := s.scenarioData.TerrainMenDefence[tt2] * s.scenarioData.FormationMenDefence[unit2.Formation] * unit2.MenCount / 32
		equipCoeff := s.scenarioData.TerrainTankAttack[tt2] * s.scenarioData.FormationTankDefence[unit2.Formation] * s.scenarioData.Data16Low[unit2.Type] / 2 * unit2.EquipCount / 64
		w := (menCoeff + equipCoeff) * unit2.Morale / 256 * (240 - unit2.Fatigue/2) / 128 * s.generals[1-unit.Side][unit2.GeneralIndex].Data2Low / 16
		if unit2.SupplyLevel == 0 {
			w = w * s.scenarioData.Data167 / 8
		}
		w *= s.magicCoeff(s.hexes.Arr144[:], unit2.X, unit2.Y, 1-unit.Side) / 8
		w++
		d := w * 16 / v
		if s.scenarioData.UnitMask[unit.Type]&4 == 0 {
			d += weather
		}
		arg1 = Clamp(d, 0, 63)
		if !unit.FormationTopBit || s.scenarioData.Data32[unit.Type]&128 == 0 {
			menLost := Clamp((Rand(unit.MenCount*arg1)+255)/512, 0, unit.MenCount)
			s.menLost[unit.Side] += menLost
			unit.MenCount -= menLost
			tanksLost := Clamp((Rand(unit.EquipCount*arg1)+255)/512, 0, unit.EquipCount)
			s.tanksLost[unit.Side] += tanksLost
			unit.EquipCount -= tanksLost
			if arg1 < 24 {
				unit.Morale = Clamp(unit.Morale+1, 0, 250)
			}
			unit2.State |= 2
			if arg1 > 32 {
				unit.Order = data.Defend // ? ^48
				message = WeHaveMetStrongResistance{unit}
				unit.Morale = Abs(unit.Morale - 2)
			}
		}
		unit.Fatigue = Clamp(unit.Fatigue+arg1, 0, 255)
		unit.SupplyLevel = Clamp(unit.SupplyLevel-s.scenarioData.Data162, 0, 255)
		arg1 = Clamp(v*16/w-weather, 0, 63)
		if false /* DitD || CiV */ {
			arg1 = Clamp(v*16/w-weather, 0, 128)
		}
		// function13(sx, sy)
		// function4(arg1) - some delay?
		menLost2 := Clamp((Rand(unit2.MenCount*arg1)+500)/512, 0, unit2.MenCount)
		s.menLost[1-unit.Side] += menLost2
		tanksLost2 := Clamp((Rand(unit2.EquipCount*arg1)+255)/512, 0, unit2.EquipCount)
		s.tanksLost[1-unit.Side] += tanksLost2
		unit2.SupplyLevel = Clamp(unit2.SupplyLevel-s.scenarioData.Data163, 0, 255)
		// in CiV instead of the second term it's s.scenarioData.UnitMask[unit2.Type]&2 == 0
		if s.scenarioData.UnitCanMove[unit2.Type] && !unit.FormationTopBit &&
			arg1-s.scenarioData.Data0Low[unit2.Type]*2+unit2.Fatigue/4 > 36 {
			unit2.Morale = Abs(unit2.Morale - 1)
			oldX, oldY := unit2.X, unit2.Y
			s.hideUnit(unit2)
			if unit2.Fatigue > 128 {
				unit2SupplyUnit := s.units[unit2.Side][unit2.SupplyUnit]
				if unit2SupplyUnit.State&128 > 0 {
					unit2.Morale = Abs(unit2.Morale - s.countNeighbourUnits(unit2.X, unit2.Y, unit.Side)*4)
					unit2.X = unit2SupplyUnit.X
					unit2.Y = unit2SupplyUnit.Y
					unit2.State = 0
					unit2.HalfDaysUntilAppear = 6
					unit2.InvAppearProbability = 6
					if false /* DitD || CiV */ {
						unit2.HalfDaysUntilAppear = 4
						unit2.InvAppearProbability = 4
						unit2.Fatigue = 130
						if false /* CiV */ {
							unit2.Fatigue = 120
						}
					}
					message = WeHaveBeenOverrun{unit2}
				}
			}
			v = -128
			bestX, bestY := unit2.X, unit2.Y
			for i := 0; i < 6; i++ {
				nx := unit2.X + s.generic.Dx[i]
				ny := unit2.Y + s.generic.Dy[i]
				tt := s.terrainTypeAt(nx, ny)
				r := s.scenarioData.TerrainMenDefence[tt]
				if s.scenarioData.MoveCostPerTerrainTypesAndUnit[tt][unit2.Type] > 0 {
					if !s.ContainsUnit(nx, ny) && !s.ContainsCity(nx, ny) {
						r += s.magicCoeff(s.hexes.Arr96[:], nx, ny, 1-unit.Side) * 4
						if r > 11 && r >= v {
							v = r
							bestX, bestY = nx, ny
						}
					}
				}
			}
			unit2.X, unit2.Y = bestX, bestY // moved this up comparing to the original code
			unit2.Terrain = s.terrainAt(unit2.X, unit2.Y)
			if _, ok := message.(WeHaveBeenOverrun); !ok {
				if true /* !CiV */ || s.options.IsPlayerControlled(1-unit.Side) || s.options.Intelligence == Full {
					s.showUnit(unit2)
				}
				if true /* !CiV */ {
					unit.ObjectiveX = unit2.X
					unit.ObjectiveY = unit2.Y
				} else /* CiV */ {
					unit2.State &= 190 // retreating? unit no longer visible
				}
			}
			if bestX != oldX || bestY != oldY {
				// unit2 is retreating, unit one is chasing (and maybe capturing a city)
				if _, ok := message.(WeHaveBeenOverrun); !ok {
					message = WeAreRetreating{unit2}
				}
				if arg1 > 60 && (true /* !CiV */ || !unit.FormationTopBit) &&
					s.magicCoeff(s.hexes.Arr96[:], oldX, oldY, unit.Side) > -4 &&
					s.scenarioData.MoveCostPerTerrainTypesAndUnit[s.terrainTypeAt(oldX, oldY)][unit.Type] > 0 {
					s.hideUnit(unit)
					unit.X = oldX
					unit.Y = oldY
					unit.Terrain = s.terrainAt(unit.X, unit.Y)
					s.showUnit(unit)
					if city, captured := s.function16(unit); captured {
						message = WeHaveCaptured{unit, city}
					}
				}
			} else {
				message = nil
			}
			unit2.Formation = s.scenarioData.Data176[1][0]
			unit2.Order = data.OrderType(s.scenarioData.Data176[1][0] + 1)
			unit2.State |= 32
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
		dif := Sign((unit.Formation & 7) - unit.TargetFormation)
		temp := s.scenarioData.Data216[4+dif*4+unit.Formation]
		if temp > Rand(15) {
			unit.FormationTopBit = false
			unit.Formation -= dif
		}
		if temp&16 == 0 {
			break
		}
	}
	{
		recovery := s.scenarioData.RecoveryRate[unit.Type]
		if unit.State&9 == 0 { // has supply line and is not in contact with enemy
			recovery *= 2
		}
		unit.Fatigue = Clamp(unit.Fatigue-recovery, 0, 255)
	}
	s.units[unit.Side][unit.Index] = unit
	return
}

// Has unit captured a city
func (s *GameState) function16(unit data.Unit) (data.City, bool) {
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
	return data.City{}, false
}
func (s *GameState) coordsToMapIndex(x, y int) int {
	return y*s.terrainMap.Width + x/2 - y/2
}

// TODO: deduplicate this function and FindBestMoveFromTowards()
func (s *GameState) function6(offsetIx, add, x, y, unitType int) (int, int) {
	ni := s.generic.DirectionToNeighbourIndex[offsetIx]

	neigh1 := s.generic.Neighbours[add][ni]
	offset1 := s.generic.MapOffsets[neigh1]
	ix1 := s.coordsToMapIndex(x, y) + offset1
	if !s.terrainMap.IsIndexValid(ix1) {
		return 0, 0
	}
	tt1 := s.terrainTypeAtIndex(ix1)
	mc1 := s.scenarioData.MoveCostPerTerrainTypesAndUnit[tt1][unitType]

	neigh2 := s.generic.Neighbours[add+1][ni]
	offset2 := s.generic.MapOffsets[neigh2]
	ix2 := s.coordsToMapIndex(x, y) + offset2
	if !s.terrainMap.IsIndexValid(ix2) {
		return 0, 0
	}
	tt2 := s.terrainTypeAtIndex(ix2)
	mc2 := s.scenarioData.MoveCostPerTerrainTypesAndUnit[tt2][unitType]

	if mc2 > mc1-Rand(2) {
		return neigh2, mc2
	}
	return neigh1, mc1
}

func (s *GameState) function29_showUnit(unit data.Unit) {
	if unit.State&65 != 0 || s.options.IsPlayerControlled(unit.Side) || s.options.Intelligence == Full {
		s.showUnit(unit)
	}
}

// arr is one of 48 element arrays in Hexes
func (s *GameState) magicCoeff(arr []int, x, y, side int) int {
	var bitmaps [5]byte
	for i := 5; i >= 0; i-- {
		bitmaps[0] <<= 2
		bitmaps[1] <<= 2
		bitmaps[4] <<= 2
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

	bitmaps[3] = bitmaps[1]
	bitmaps[2] = bitmaps[0]

	bitmaps[1] = rotateRight6Bits(bitmaps[1])
	bitmaps[0] = rotateRight6Bits(bitmaps[0])

	bitmaps[3] |= rotateRight6Bits(bitmaps[1])
	bitmaps[2] |= rotateRight6Bits(bitmaps[0])

	bitmaps[1] |= rotateRight6Bits(bitmaps[4])

	xA70B := [16]int{0, 2, 1, 0, 4, 2, 1, 0, 3, 2, 1, 0, 5, 2, 1, 0}
	var xA705 [6]int

	for i := 0; i < 6; i++ {
		xA6FE := 0
		for Y := 3; Y >= 0; Y-- {
			xA6FE <<= 1
			if bitmaps[Y]&1 != 0 {
				xA6FE++
			}
			bitmaps[Y] >>= 1
		}

		A := xA70B[xA6FE]
		xA705[A]++
	}
	xA704 := 0
	for i := 5; i >= 0; i-- {
		xA704 += arr[8*i+xA705[i]]
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

func function8(dx, dy int) int {
	return Sign(dy)*5 + Sign(dx-dy)*3 + Sign(dx+dy)
}
func (s *GameState) function10(order data.OrderType, offset int) int {
	if offset < 0 || offset >= 4 {
		panic(offset)
	}
	return s.scenarioData.Data176[int(order)][offset]
}

func Function15_distanceToObjective(unit data.Unit) int {
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

func (s *GameState) reinitSmallMapsAndSuch(currentSide int) {
	s.resetMaps()
	//	v13 := 0
	//	v15 := 0
	//	v16 := 0
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if unit.State&128 == 0 ||
				s.scenarioData.UnitMask[unit.Type]&16 != 0 {
				continue
			}
			sx, sy := unit.X/8, unit.Y/4
			if !InRange(sx, 0, 16) || !InRange(sy, 0, 16) {
				continue
			}
			if unit.Side == currentSide {
				//				v15 += unit.MenCount + unit.EquipCount
				//				v13 += 1
			} else {
				//				v16 += unit.MenCount + unit.EquipCount
				// Possibly a bug in the original code and there should be /4 instead of /16
				// Otherwise it's a way too complicated method to check for limited intelligence.
				if s.options.Intelligence == Limited && //(unit.Side+1)&(s.options.Num()/16) > 0 &&
					unit.State&64 == 0 {
					continue
				}
			}
			v30 := unit.MenCount + unit.EquipCount
			tmp := Clamp(s.scenarioData.FormationMenDefence[unit.Formation], 8, 99) * v30 / 8
			v29 := s.scenarioData.TerrainMenDefence[s.terrainTypeAt(unit.X, unit.Y)] * tmp / 8
			if s.scenarioData.UnitScores[unit.Type] > 7 {
				// special units - supply, air wings
				v29 = 4
				v30 = 4
			}
			s.map0[unit.Side][sx][sy] += (v30 + 4) / 8
			s.map3[unit.Side][sx][sy] = Clamp(s.map3[unit.Side][sx][sy]+(v29+4)/8, 0, 255)
			if s.scenarioData.AvgDailySupplyUse < unit.SupplyLevel-1 {
				v29 = s.scenarioData.UnitScores[unit.Type] / 4
				if v29 > 0 {
					for v30 = -1; v30 <= v29; v30++ {
						for v6 := 0; v6 <= (Abs(v30)-Sign(Abs(v30)))*4; v6++ {
							dx, dy := s.generic.SmallMapOffsets(v6)
							x, y := sx+dx, sy+dy
							if !InRange(x, 0, 16) || !InRange(y, 0, 16) {
								continue
							}
							s.map1[unit.Side][x][y] += 2
							if unit.State&2 != 0 {
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
		if unit.State&128 == 0 {
			continue
		}
		if Abs(unit.X-x)+Abs(2*(unit.Y-y)) < 4 { // TODO: double check it
			num++
		}
	}
	return num
}

func (s *GameState) everyHour() bool {
	if s.hour == 12 {
		if !s.every12Hours() {
			return false
		}
	}
	sunriseOffset := Abs(6-s.month) / 2
	s.isNight = s.hour < 5+sunriseOffset || s.hour > 20-sunriseOffset

	if s.scenarioData.AvgDailySupplyUse > Rand(24) {
		for _, sideUnits := range s.units {
			for i, unit := range sideUnits {
				if unit.State&128 == 0 ||
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
	s.hideAllUnits()
	if s.isNight { // in CiV - opposite
		s.sync.SendUpdate(SupplyDistributionStart{})
	}
	for _, sideUnits := range s.units {
		for i, unit := range sideUnits {
			if unit.State&128 != 0 {
				if s.isNight { // if it's midnight (in CiV - opposite)
					unit = s.resupplyUnit(unit)
				}
			} else {
				if unit.HalfDaysUntilAppear == 0 {
					continue
				}
				unit.HalfDaysUntilAppear--
				if unit.HalfDaysUntilAppear == 0 {
					shouldSpawnUnit := !s.ContainsUnit(unit.X, unit.Y) &&
						Rand(unit.InvAppearProbability) == 0
					if city, ok := s.FindCity(unit.X, unit.Y); ok && city.Owner != unit.Side {
						shouldSpawnUnit = false
					}
					if shouldSpawnUnit {
						unit.State |= 128
						unit.Terrain = s.terrainAt(unit.X, unit.Y)
						// Unit will be shown if needed inside showAllUnits at the end of the function.
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
			if unit.State&8 == 0 { // (has supply line)
				if unit.MenCount <= s.scenarioData.MenCountLimit[unit.Type] {
					unit.MenCount += Rand(s.scenarioData.MenReplacementRate[unit.Side]+32) / 32
				}
				if unit.EquipCount <= s.scenarioData.EquipCountLimit[unit.Type] {
					unit.EquipCount += Rand(s.scenarioData.EquipReplacementRate[unit.Side]+32) / 32
				}
			}
			sideUnits[i] = unit
		}
	}
	s.showAllVisibleUnits()
	if s.isNight { // in CiV - opposite
		s.sync.SendUpdate(SupplyDistributionEnd{})
	}
	if reinforcements[0] || reinforcements[1] {
		if !s.sync.SendUpdate(Reinforcements{Sides: reinforcements}) {
			return false
		}
	}
	return true
}

func (s *GameState) resupplyUnit(unit data.Unit) data.Unit {
	unit.OrderBit4 = false
	if !s.scenarioData.UnitUsesSupplies[unit.Type] ||
		!s.scenarioData.UnitCanMove[unit.Type] {
		return unit
	}
	unit.State |= 8
	minSupplyType := s.scenarioData.MinSupplyType & 15
	if unit.Type >= minSupplyType {
		// headquarters can only gain supply from supply depots,
		//  not other headquarters
		minSupplyType++
	}
	if s.options.IsPlayerControlled(unit.Side) || s.options.Intelligence == Full {
		s.showUnit(unit)
	}
	// keep the last friendly unit so that we can use it outside of the loop
	var supplyUnit data.Unit
outerLoop:
	for j := 0; j < len(s.units[unit.Side]); j++ {
		supplyUnit = s.units[unit.Side][j]
		if supplyUnit.Type < minSupplyType ||
			supplyUnit.State&128 == 0 || supplyUnit.SupplyLevel == 0 {
			continue
		}
		supplyX, supplyY := supplyUnit.X, supplyUnit.Y
		if s.options.IsPlayerControlled(unit.Side) || s.options.Intelligence == Full {
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
					unit.State &= 247
				} else {
					// not sure if it's needed...
					s.supplyLevels[unit.Side] = 0
				}
				s.hideUnit(supplyUnit)
				break outerLoop
			} else {
				var x, y, cost int
				for variant := 0; variant < 1; variant++ {
					x, y, cost = s.FindBestMoveFromTowards(supplyX, supplyY, unit.X, unit.Y, s.scenarioData.MinSupplyType, variant)
					if cost != 0 {
						break
					}
				}
				if s.options.IsPlayerControlled(unit.Side) || s.options.Intelligence == Full {
					s.sync.SendUpdate(SupplyTruckMove{supplyX / 2, supplyY, x / 2, y})
					//  function13(x, y) (show truck icon at x, y)
				}
				//dx, dy := moveToXY(move)
				supplyX, supplyY = x, y
				if s.ContainsUnitOfSide(supplyX, supplyY, 1-unit.Side) {
					break
				}
				supplyTransportBudget -= 256 / (cost + 1)
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
	for _, unit := range s.units[side] {
		if (unit.State&128) != 0 && unit.X == x && unit.Y == y {
			return true
		}
	}
	return false
}
func (s *GameState) ContainsCity(x, y int) bool {
	for _, city := range s.terrain.Cities {
		if city.X == x && city.Y == y {
			return true
		}
	}
	return false
}

func (s *GameState) FindUnit(x, y int) (data.Unit, bool) {
	return s.FindUnitAtMapCoords(x/2, y)
}
func (s *GameState) FindUnitAtMapCoords(x, y int) (data.Unit, bool) {
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if unit.State&128 != 0 && unit.X/2 == x && unit.Y == y {
				return unit, true
			}
		}
	}
	return data.Unit{}, false
}
func (s *GameState) FindUnitOfSide(x, y, side int) (data.Unit, bool) {
	for _, unit := range s.units[side] {
		if unit.State&128 != 0 && unit.X == x && unit.Y == y {
			return unit, true
		}
	}
	return data.Unit{}, false
}
func (s *GameState) FindCity(x, y int) (data.City, bool) {
	for _, city := range s.terrain.Cities {
		if city.X == x && city.Y == y {
			return city, true
		}
	}
	return data.City{}, false
}
func (s *GameState) SaveCity(newCity data.City) {
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

func (s *GameState) showUnit(unit data.Unit) {
	ix := s.coordsToMapIndex(unit.X, unit.Y)
	s.terrainMap.SetTileAtIndex(ix, byte(unit.Type+unit.ColorPalette*16))
}
func (s *GameState) hideUnit(unit data.Unit) {
	ix := s.coordsToMapIndex(unit.X, unit.Y)
	s.terrainMap.SetTileAtIndex(ix, unit.Terrain)
}
func (s *GameState) hideAllUnits() {
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if unit.State&128 != 0 {
				s.hideUnit(unit)
			}
		}
	}
}
func (s *GameState) showAllVisibleUnits() {
	for _, sideUnits := range s.units {
		for i, unit := range sideUnits {
			if unit.State&128 == 0 {
				continue
			}
			sideUnits[i].Terrain = s.terrainAt(unit.X, unit.Y)
			if unit.State&65 != 0 || s.options.IsPlayerControlled(unit.Side) || s.options.Intelligence == Full {
				s.showUnit(unit)
			}
		}
	}
}

func (s *GameState) FindBestMoveFromTowards(supplyX, supplyY, unitX, unitY, unitType, variant int) (int, int, int) {
	dx, dy := unitX-supplyX, unitY-supplyY
	neighbour1 := s.generic.DxDyToNeighbour(dx, dy, 2*variant)
	supplyX1 := supplyX + s.generic.Dx[neighbour1]
	supplyY1 := supplyY + s.generic.Dy[neighbour1]
	terrainType1 := s.terrainTypeAt(supplyX1, supplyY1)
	cost1 := s.scenarioData.MoveCostPerTerrainTypesAndUnit[terrainType1][unitType]

	neighbour2 := s.generic.DxDyToNeighbour(dx, dy, 2*variant+1)
	supplyX2 := supplyX + s.generic.Dx[neighbour2]
	supplyY2 := supplyY + s.generic.Dy[neighbour2]
	terrainType2 := s.terrainTypeAt(supplyX2, supplyY2)
	cost2 := s.scenarioData.MoveCostPerTerrainTypesAndUnit[terrainType2][unitType]
	if cost2 > cost1-Rand(2) {
		return supplyX2, supplyY2, cost2
	}
	return supplyX1, supplyY1, cost1
}

func (s *GameState) everyDay() bool {
	s.daysElapsed++
	var flashback []data.FlashbackUnit
	numActiveUnits := 0
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if unit.State&136 != 0 { // active or no supply line
				numActiveUnits++
			}
			if unit.State&128 != 0 {
				flashback = append(flashback, data.FlashbackUnit{
					X: unit.X, Y: unit.Y, ColorPalette: unit.ColorPalette, Type: unit.Type,
				})
			}
		}
	}
	s.numUnitsToUpdatePerTimeIncrement = (numActiveUnits*s.scenarioData.UnitUpdatesPerTimeIncrement)/128 + 1

	s.flashback = append(s.flashback, flashback)
	// todo: save today's map for flashback
	rnd := Rand(256)
	if rnd < 140 {
		s.weather = int(s.scenarioData.PossibleWeather[4*(s.month/3)+rnd/35])
	}
	s.sync.SendUpdate(WeatherForecast{s.weather})
	if !s.every12Hours() {
		return false
	}
	s.sync.SendUpdate(DailyUpdate{
		DaysRemaining: s.variant.LengthInDays - s.daysElapsed + 1,
		SupplyLevel:   Clamp(s.supplyLevels[s.playerSide]/256, 0, 2)})
	for _, update := range s.scenarioData.DataUpdates {
		if update.Day == s.daysElapsed {
			s.scenarioData.UpdateData(update.Offset, update.Value)
		}
	}
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
func (s *GameState) winningSideAndAdvantage() (winningSide int, advantage int) {
	side0Score := (1+s.menLost[1]+s.tanksLost[1])*s.variant.Data3/8 + s.citiesHeld[0]*3
	side1Score := 1 + s.menLost[0] + s.tanksLost[0] + s.citiesHeld[1]*3
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
func (s *GameState) statusReport() string {
	var strs []string
	strs = append(strs, fmt.Sprintf("STATUS REPORT\t%s\t%s", s.scenarioData.Sides[0], s.scenarioData.Sides[1]))
	menMultiplier, tanksMultiplier := s.scenarioData.MenMultiplier, s.scenarioData.TanksMultiplier
	strs = append(strs, fmt.Sprintf("TROOPS LOST\t%d\t%d", s.menLost[0]*menMultiplier, s.menLost[1]*menMultiplier))
	strs = append(strs, fmt.Sprintf("TANKS LOST\t%d\t%d", s.tanksLost[0]*tanksMultiplier, s.tanksLost[1]*tanksMultiplier))
	strs = append(strs, fmt.Sprintf("CITIES HELD\t%d\t%d", s.citiesHeld[0], s.citiesHeld[1]))
	winningSide, advantage := s.winningSideAndAdvantage()
	advantageStrs := []string{"SLIGHT", "MARGINAL", "TACTICAL", "DECISIVE", "TOTAL"}
	winningSideStr := s.scenarioData.Sides[winningSide]
	strs = append(strs, fmt.Sprintf("   %s %s ADVANTAGE.", advantageStrs[advantage], winningSideStr))
	return strings.Join(strs, "\n")
}

func (s *GameState) finalResults() string {
	winningSide, advantage := s.winningSideAndAdvantage()
	absoluteAdvantage := 6
	if winningSide == 0 {
		absoluteAdvantage -= advantage + 1
	} else {
		absoluteAdvantage += advantage
	}
	v73 := s.playerSide
	if s.options.Num()%4 == 0 { // if a demonstration game
		if absoluteAdvantage < 6 {
			v73 = 1
		} else {
			v73 = 0
		}
	}
	v74 := absoluteAdvantage + v73*(11-absoluteAdvantage*2)
	criticalLocationBalance := s.criticalLocationsCaptured[0] - s.criticalLocationsCaptured[1]
	if criticalLocationBalance >= s.variant.CriticalLocations[0] {
		v74 = 1 + 9*(1-v73)
	}
	if -criticalLocationBalance >= s.variant.CriticalLocations[1] {
		v74 = 1 + 9*v73
	}
	var strs []string
	finalResults := []string{"TOTAL DEFEAT", "DECISIVE DEFEAT", "TACTICAL DEFEAT", "MARGINAL DEFEAT", "DEFEAT", "DISADVANTAGE", "ADVANTAGE", "MARGINAL VICTORY", "TACTICAL VICTORY", "DECISIVE VICTORY", "TOTAL VICTORY"}
	strs = append(strs, fmt.Sprintf("FINAL RESULT: %s", finalResults[v74-1]))
	balance := s.options.GameBalance + v73*(4-2*s.options.GameBalance)
	balances := []string{"VERY DIFFICULT", "DIFFICULT", "FAIR", "EASY", "VERY EASY"}
	strs = append(strs, fmt.Sprintf("PLAY BALANCE: %s", balances[balance-1]))
	rank := Min(v74-2*balance+4, 12)
	ranks := []string{"PRIVATE", "SERGEANT", "LIEUTENANT", "CAPTAIN", "MAJOR", "LIEUTENANT-COLONEL", "COLONEL", "BRIGADIER-GENERAL", "MAJOR-GENERAL", "LIEUTENANT-GENERAL", "FIELD MARSHAL", "SUPREME COMMANDER"}
	strs = append(strs, fmt.Sprintf("YOUR RANK: %s", ranks[rank-1]))
	return strings.Join(strs, "\n")
}
func (s *GameState) isGameOver() bool {
	if s.daysElapsed >= s.variant.LengthInDays {
		return true
	}
	criticalLocationBalance := s.criticalLocationsCaptured[0] - s.criticalLocationsCaptured[1]
	if criticalLocationBalance >= s.variant.CriticalLocations[0] {
		return true
	}
	if -criticalLocationBalance >= s.variant.CriticalLocations[1] {
		return true
	}
	return false
}

func (s *GameState) dateTimeString() string {
	meridianString := "AM"
	if s.hour >= 12 {
		meridianString = "PM"
	}
	hour := Abs(s.hour - 12*((s.hour+11)/12-1))
	return fmt.Sprintf("%02d:%02d %s %s, %d %d  %s", hour, s.minute, meridianString, s.scenarioData.Months[s.month], s.day+1, s.year, s.scenarioData.Weather[s.weather])
}
