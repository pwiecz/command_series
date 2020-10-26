package main

import "fmt"
import "image"
import "image/color"
import "math"
import "math/rand"

import "github.com/hajimehoshi/ebiten"
import "github.com/hajimehoshi/ebiten/ebitenutil"
import "github.com/pwiecz/command_series/data"

type Options struct {
	AlliedCommander int
	GermanCommander int
	Intelligence    int
}

func (o Options) IsPlayerControlled(side int) bool {
	if side == 0 {
		return o.AlliedCommander > 0
	}
	return o.GermanCommander > 0
}
func (o Options) Num() int {
	n := o.AlliedCommander + 2*o.GermanCommander
	if o.Intelligence == 1 {
		n += 56 - 4*(o.AlliedCommander*o.GermanCommander+o.AlliedCommander)
	}
	return n
}

type ShowMap struct {
	mainGame        *Game
	mapImage        *ebiten.Image
	options         Options
	dx, dy          uint8
	minute          int
	hour            int
	day             int /* 0-based */
	month           int /* 0-based */
	year            int
	supplyLevels    [2]int
	currentSpeed    int
	idleTicksLeft   int
	unitsUpdated    int
	weather         int
	isNight         bool
	lastUpdatedUnit int
	citiesHeld      [2]int // 29927 + 15 + side*2
	menLost         [2]int // 29927 + side*2
	tanksLost       [2]int // 29927 + 4 + side*2
	score13         [2]int // 29927 + 13 + side*2 victory points held?
	score21         [2]int // 29927 + 21 + side*2 sth with capturing cities?
	flashback       [][]data.FlashbackUnit
	map0            [2][16][16]int // 0
	map1            [2][16][16]int // 0x200
	map2_0, map2_1  [2][4][4]int   // 0x400 - two byte values
	map2_2, map2_3  [2][16]int
	map3            [2][16][16]int // 0x600
	mode            data.OrderType
	update          int
}

func NewShowMap(g *Game) *ShowMap {
	scenario := g.scenarios[g.selectedScenario]
	variant := g.variants[g.selectedVariant]
	s := &ShowMap{
		mainGame:        g,
		dx:              scenario.MinX,
		dy:              scenario.MinY,
		minute:          scenario.StartMinute,
		hour:            scenario.StartHour,
		day:             scenario.StartDay,
		month:           scenario.StartMonth,
		year:            scenario.StartYear,
		weather:         scenario.StartWeather,
		supplyLevels:    scenario.StartSupplyLevels,
		currentSpeed:    1,
		idleTicksLeft:   60,
		lastUpdatedUnit: 127,
		citiesHeld:      variant.CitiesHeld,
	}
	s.init()
	s.everyHour()
	return s
}

func (s *ShowMap) createMapImage() {
	var mapImage image.Image
	var err error
	if !s.isNight {
		mapImage, err = s.mainGame.terrainMap.GetImage(s.mainGame.sprites.TerrainTiles[:],
			s.mainGame.scenarioData.DaytimePalette)
	} else {
		mapImage, err = s.mainGame.terrainMap.GetImage(s.mainGame.sprites.TerrainTiles[:],
			s.mainGame.scenarioData.NightPalette)
	}
	if err != nil {
		panic(err)
	}
	mapEImage, err := ebiten.NewImageFromImage(mapImage, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	s.mapImage = mapEImage
}

func (s *ShowMap) Update(screen *ebiten.Image) error {
	if s.idleTicksLeft > 0 {
		s.idleTicksLeft--
		return nil
	}
	s.unitsUpdated++
	if s.unitsUpdated <= s.mainGame.scenarioData.UnitUpdatesPerTimeIncrement/2 {
		s.updateUnit()
		return s.Update(screen)
	}
	s.unitsUpdated = 0
	s.minute += s.mainGame.scenarioData.MinutesPerTick
	if s.minute >= 60 {
		s.minute = 0
		s.hour++
		s.everyHour()
	}
	if s.hour >= 24 {
		s.hour = 0
		s.day++
		s.everyDay()
	}
	if s.day >= monthLength(s.month+1, s.year+1900) {
		s.day = 0
		s.month++
	}
	if s.month >= 12 {
		s.month = 0
		s.year++
	}
	return nil
}
func (s *ShowMap) init() {
	for _, sideUnits := range s.mainGame.units {
		for i, unit := range sideUnits {
			if unit.VariantBitmap&(1<<s.mainGame.selectedVariant) != 0 {
				unit.State = 0
				unit.HalfDaysUntilAppear = 0
			}
			unit.VariantBitmap = 0 // not really needed
			sideUnits[i] = unit
		}
	}
	for i, city := range s.mainGame.terrain.Cities {
		if city.VariantBitmap&(1<<s.mainGame.selectedVariant) != 0 {
			city.VictoryPoints = 0
			s.mainGame.terrain.Cities[i] = city
		}
	}
}

func (s *ShowMap) resetMaps() {
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

func (s *ShowMap) updateUnit() {
	unitState := 0
	weather := s.weather
	if s.isNight {
		weather += 8
	}
nextUnit:
	s.lastUpdatedUnit = (s.lastUpdatedUnit + 1) % 128
	unit := s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2]
	var v9 int
	if unit.State&128 == 0 {
		goto nextUnit
	}
	if unit.MenCount+unit.EquipCount < 7 {
		unitState = 3 // surrender
	}
	if unit.Fatigue == 255 {
		unitState = 3 // surrender
	}
	if unitState != 0 {
		unit.State = 0
		unit.HalfDaysUntilAppear = 0
		s.citiesHeld[1-unit.Side] += s.mainGame.scenarioData.UnitScores[unit.Type]
		s.menLost[unit.Side] += unit.MenCount
		s.tanksLost[unit.Side] += unit.EquipCount
		goto end
	}
	if !s.mainGame.scenarioData.UnitCanMove[unit.Type] {
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
			s.mode = unit.Order // ? plus top two bits
			unit.State |= 32
			goto l24
		}
	} else {
		if unit.OrderLower4Bits&8 != 0 {
			s.mode = unit.Order // ? plus top two bits
			goto l24
		}
	}
	if s.update != unit.Side {
		s.reinitSmallMapsAndSuch()
	}
	{
		// v57 := sign(sign_extend([29927 + 10 + unit.side])/16)*4
		sx, sy := unit.X/8, unit.Y/4
		v30 := 0
		for i := 0; i < 9; i++ {
			dx, dy := s.mainGame.generic.SmallMapOffsets(i)
			if InRange(sx+dx, 0, 16) && InRange(dy+sy, 0, 16) {
				v30 += s.map0[1-unit.Side][sx+dx][sy+dy]
			}
		}
		if v30 == 0 {
			if s.mainGame.scenarioData.UnitScores[unit.Type]+int(unit.State&8) == 0 {
				tx, ty := unit.X/32, unit.Y/16
				//unit.X /= 4
				//unit.Y /= 4
				arg1 := -17536 // 0xBB80
				bestI := 0
				bestX, bestY := 0, 0
				for i := 0; i < 9; i++ {
					//t := s.mainGame.generic.Data44[i]
					//if !InRange(SignInt(int(int8((t&6)*32)))*8+unit.X+1, 1, 33) {
					//	panic("")
					//}
					//if !InRange(SignInt((int(int8(t))+2)/8)*4+unit.Y+1, 1, 17) {
					//	panic("")
					//}
					dx, dy := s.mainGame.generic.TinyMapOffsets(i)
					x, y := tx+dx, ty+dy
					if !InRange(x, 0, 4) || !InRange(y, 0, 4) {
						continue
					}
					val := (s.map2_1[unit.Side][x][y] + s.map2_1[1-unit.Side][x][y]) * 16 / ClampInt(s.map2_0[unit.Side][x][y]-s.map2_0[1-unit.Side][x][y], 10, 9999)
					tmp := s.function26(unit.X/4, unit.Y/4, val, i)
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
				//unit = s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2]
				if bestI > 0 {
					unit.OrderLower4Bits = 0
					unit.Order = 0
					v30 = (unit.MenCount + unit.EquipCount + 8) / 16
					s.map2_0[unit.Side][tx][ty] = AbsInt(s.map2_0[unit.Side][bestX][bestY] - v30)
					s.map2_0[unit.Side][bestX][bestY] += v30
					unit.ObjectiveX = bestX*32 + 16 // ((v20&6)*16)|16
					unit.ObjectiveY = bestY*16 + 8  // ((v20&24)*2)| 8
					goto l21
				}
			}
		}
		{
			v58 := s.mainGame.hexes.Arr3[unit.Side][unit.GeneralIndex][0]
			arg1 := -17536 // 0xBB80
			//var bestI int
			var bestDx, bestDy int
			var v63 int
			temp2 := (unit.MenCount + unit.EquipCount + 4) / 8
			v61 := temp2 * ClampInt(s.mainGame.scenarioData.Data144[unit.Formation&7], 8, 99) / 8 * s.mainGame.scenarioData.Data112[s.terrainTypeAt(unit.X, unit.Y)] / 8
			if s.mainGame.scenarioData.UnitScores[unit.Type] > 7 {
				temp2 = 1
				v61 = 1
			}
			s.map0[unit.Side][sx][sy] = ClampInt(s.map0[unit.Side][sx][sy]-temp2, 0, 255)
			s.map3[unit.Side][sx][sy] = ClampInt(s.map3[unit.Side][sx][sy]-v61, 0, 255)
			// save a copy of the unit, as we're going to modify it.
			unitCopy := unit
			for i := 1; i <= 9; i++ {
				dx, dy := s.mainGame.generic.SmallMapOffsets(i - 1)
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
					ddx, ddy := s.mainGame.generic.SmallMapOffsets(i + 1)
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
						v48 = ClampInt((unit.MenCount+1)*8/(v52+1)-7, 0, 16)
					} else {
						v48 = -ClampInt((v52+1)*8/(unit.MenCount+1)-8, 0, 16)
					}
					v48 += int(int8((s.mainGame.hexes.Arr3[unit.Side][unit.GeneralIndex][1]&240))>>4) + int(int8(s.mainGame.scenarioData.Data[unit.Type]))/16
					var v55 int
					if unit.EquipCount > v16 {
						v55 = ClampInt((unit.EquipCount+1)*8/(v16+1)-7, 0, 16)
					} else {
						v55 = -ClampInt((v16+1)*8/(unit.EquipCount+1)-8, 0, 16)
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
							if v58*32 > 0 {
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
						if v55+(int(int8(s.mainGame.hexes.Arr3[unit.Side][unit.GeneralIndex][2]&240))>>4)+(int(int8((s.mainGame.scenarioData.Data[unit.Type]&15)<<4))>>4) < -9 {
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
								v9 = 2 // have met strong resistenance (?)
							}
						}
					}
				}
				v := s.mainGame.scenarioData.UnitScores[unit.Type] & 248
				if unit.State&1 > 0 {
					v += (unit.Fatigue/16 + unit.Fatigue/32)
				}
				if v > 7 {
					t = unit.EquipCount - v52*2
					v9 = -128
					temp = data.Reserve
					unit.Fatigue &= 255
				}
				t = s.function26(unit.X, unit.Y, t, i)
				if i == 1 {
					v63 = t
					s.mode = temp
				}
				if t > arg1 {
					arg1 = t
					bestDx, bestDy = dx, dy
					//bestI = i
				}
				if i+1 > SignInt(int(s.mode))+v9 {
					continue
				}
				break
			}
			// function18
			unit = unitCopy // revert modified unit
			unit.OrderLower4Bits |= 8
			supplyUse := s.mainGame.scenarioData.ProbabilityOfUnitsUsingSupplies
			if unit.State&8 > 0 {
				supplyUse *= 2
			}
			if unit.SupplyLevel < supplyUse {
				unit2 := s.mainGame.units[unit.Side][unit.FormationHigher4Bits]
				if unit2.State&128 == 0 {
					unit2 = s.mainGame.units[unit.Side][unit2.FormationHigher4Bits]
				}
				unit.ObjectiveX = unit2.X
				unit.ObjectiveY = unit2.Y
				t := data.Move
				if v9 > 0 {
					t = data.Defend
				}
				unit.Order = t
				unit.OrderLower4Bits = 0
				goto l21
			}
			if unit.Fatigue/4 > arg1-v63 {
				bestDx, bestDy = 0, 0
			}
			if bestDx == 0 && bestDy == 0 {
				if unit.Fatigue > 64 {
					s.mode = data.Defend
				}
				if s.mode == data.Reserve {
					s.mode = data.Defend
				}
				s.map0[unit.Side][sx][sy] += temp2
				s.map3[unit.Side][sx][sy] += v61
				// update = 13
			} else {
				if s.map0[unit.Side][sx+bestDx][sy+bestDy] > 0 {
					s.map0[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				}
				s.map3[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				unit.ObjectiveY = (((sy+bestDy)&240)/4 + rand.Intn(2)/256 + 1) & 63
				unit.ObjectiveX = ((((sy+bestDy)&15)*4+rand.Intn(2)/256+1)*2 + (unit.ObjectiveY & 1)) & 127
				s.mode = data.Move
				if v9 != 0 {
					unit.Order = data.Defend
					goto l24
				}
			}
			unit.Order = s.mode
		}
	}
l24:
	unit.OrderLower4Bits = s.function10(unit.Order, 1)
	if s.mode == data.Attack {
		arg1 := 16000
		terrainType := s.terrainTypeAt(unit.X, unit.Y)
		menCoeff := s.mainGame.scenarioData.Data96[terrainType] * unit.MenCount
		equipCoeff := s.mainGame.scenarioData.Data104[terrainType] * unit.EquipCount * (s.mainGame.scenarioData.UnitScores[unit.Type] / 16) / 4
		coeff := (menCoeff + equipCoeff) / 8 * (255 - unit.Fatigue) / 255 * (unit.Morale + int(int8(s.mainGame.scenarioData.Data0[unit.Type]&240))) / 128
		temp2 := coeff * s.magicCoeff(s.mainGame.hexes.Arr144[:], unit.X, unit.Y, unit.Side) / 8
		v := 0
		if v9 > 0 {
			if s.mainGame.scenarioData.UnitResupplyPerType[unit.Type]&7 < 3 {
				v = 12
			}
		}
		for i := v; i <= 18; i++ {
			arg2 := 16001
			nx := unit.X + s.mainGame.generic.Dx152[i]
			ny := unit.Y + s.mainGame.generic.Dy153[i]
			if s.ContainsUnitOfSide(nx, ny, 1-unit.Side) {
				unit2, ok := s.FindUnit(nx, ny)
				if !ok {
					panic("")
				}
				terrainType := s.terrainTypeAt(unit2.X, unit2.Y)
				menCoeff := s.mainGame.scenarioData.Data112[terrainType] * unit2.MenCount
				equipCoeff := s.mainGame.scenarioData.Data120[terrainType] * unit2.EquipCount * (s.mainGame.scenarioData.Data16[unit.Type&15] & 15) / 4
				t := (menCoeff + equipCoeff) * s.mainGame.scenarioData.Data144[unit.Formation&7] / 8
				w := 2
				if s.mainGame.scenarioData.UnitMask[unit.Type]&4 != 0 {
					w /= 2
				}
				d := s.mainGame.scenarioData.UnitScores[unit.Type] + int((unit.State&6)*2) + 14 - w
				n := t / ClampInt(d, 1, 32)
				arg2 = n * s.magicCoeff(s.mainGame.hexes.Arr144[:], unit2.X, unit2.Y, unit2.Side) / 8 * (255 - unit2.Fatigue) / 256 * unit2.Morale / 128
			} else {
				t := s.terrainAt(nx, ny)
				if i == 18 {
					t = s.mapTerrainAt(unit.X, unit.Y)
				}
				tt := s.terrainType(t)
				var v int
				if unit.MenCount > unit.EquipCount {
					v = s.mainGame.scenarioData.Data96[tt]
				} else {
					v = s.mainGame.scenarioData.Data104[tt]
				}
				if tt < 7 {
					// temperarily hide the unit while we compute sth
					s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2].State = unit.State & 127
					arg2 = temp2 - s.magicCoeff(s.mainGame.hexes.Arr48[:], nx, ny, unit.Side)*2 + v
					// unhide the unit
					s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2].State = unit.State
				}
			}
			if i < 12 {
				arg2 *= 2
			}
			if city, ok := s.FindCity(nx, ny); ok {
				if city.Owner != unit.Side {
					if city.VictoryPoints > 0 {
						v := -city.VictoryPoints
						if s.ContainsUnitOfSide(nx, ny, 1-unit.Side) {
							v += arg2
						}
						arg2 = v
					}
				}
			}
			if arg2-1 < arg1 {
				arg1 = arg2
				unit.ObjectiveX = nx
				unit.ObjectiveY = ny
			}
		}
	}
	if s.mode == data.Reserve {
		unit.ObjectiveX = 0
	}
	if s.mode == data.Defend {
		if unit.ObjectiveX > 0 {
			unit.ObjectiveX = unit.X
			unit.ObjectiveY = unit.Y
		}
		// temperarily hide the unit while we compute sth
		s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2].State &= 127
		arg1 := -17536 // 48000
		var bestI int
		var arg1_6 int
		for i := 0; i <= 6; i++ {
			ix := s.CoordsToMapIndex(unit.X, unit.Y) + s.mainGame.generic.MapOffsets[i]
			if !InRange(ix, 0, len(s.mainGame.terrainMap.Terrain)) {
				continue
			}
			var v int
			tt := s.terrainType(s.mainGame.terrainMap.Terrain[ix]) // terrainTypeAtIndex()?
			if tt == 7 {
				v = -128
			} else {
				r := s.mainGame.scenarioData.Data112[tt]
				nx := unit.X + s.mainGame.generic.Dx[i]
				ny := unit.Y + s.mainGame.generic.Dy[i]
				v = r + s.magicCoeff(s.mainGame.hexes.Arr0[:], nx, ny, unit.Side)
				if city, ok := s.FindCity(nx, ny); ok {
					if s.ContainsUnitOfSide(nx, ny, unit.Side) {
						v += city.VictoryPoints
					}
				}
				if (s.mainGame.scenarioData.UnitScores[unit.Type]&248)+SignInt(unit.Fatigue-96+int(int8(s.mainGame.hexes.Arr3[unit.Side][unit.GeneralIndex][2]&240))/4) > 0 {
					v = r + s.magicCoeff(s.mainGame.hexes.Arr96[:], nx, ny, unit.Side)
				}
			}
			if v+1 > arg1 {
				arg1 = v
				bestI = i
			}
			if i == 6 {
				arg1_6 = v
			}
		}
		s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2].State |= 128
		v := s.mainGame.scenarioData.Data144[unit.Formation&7] - 8
		if false { // if full intelligence?? (unit.Side+1)&auto >>>4 > 0
			v *= 2
		}
		if v > arg1-arg1_6 {
			bestI = 6
		}
		if bestI < 6 {
			unit.ObjectiveX = unit.X + s.mainGame.generic.Dx[bestI]
			unit.ObjectiveY = unit.Y + s.mainGame.generic.Dy[bestI]
		} else {
			unit.OrderLower4Bits = s.function10(unit.Order, 1)
		}
	}
	{
		t := s.mainGame.scenarioData.Data32[unit.Type]
		temp2 := (t & 31) * 2
		if temp2 > 0 {
			if (t&8)+weather < 10 {
				if 32-unit.Fatigue/4 > 0 {
					for i := 0; i <= 32-unit.Fatigue/4; i++ {
						unit2 := s.mainGame.units[1-unit.Side][rand.Intn(64)]
						if unit2.State&6 > 0 {
							if AbsInt(unit.X-unit2.X)/2+AbsInt(unit.Y-unit2.Y) <= temp2 {
								unit.ObjectiveX = unit2.X
								unit.ObjectiveY = unit2.Y
								unit.Order = unit.Order | 2
								unit.Formation = s.mainGame.scenarioData.Data178
							}
						}
					}
				}
			}
		}
	}
l21:
	s.update = unit.Side
	v9 = 0
	if unit.SupplyLevel == 0 {
		v9 = 7 // have exhausted our supplies
	}
	{
		v57 := 25
		var distance int
		var v49, sx, sy, arg1 int
		for {
			mvAdd := 0
			if unit.ObjectiveX == 0 {
				break
			}
			distance = s.function15(unit)
			i := s.mainGame.scenarioData.Data32[unit.Type]
			if distance > 0 {
				if distance < (i&31)*2+1 {
					if unit.Order == data.Attack {
						v49 = 0
						sx = unit.ObjectiveX
						sy = unit.ObjectiveY
						unit.FormationHigher4Bits |= 8
						arg1 = 7
					}
				}
			}
		l5:
			temp := function8(unit.ObjectiveX-unit.X, unit.ObjectiveY-unit.Y)
			if temp == 0 {
				unit.ObjectiveX = 0
				unit.OrderLower4Bits = s.function10(unit.Order, 1)
				break
			}
			unit.OrderLower4Bits = s.function10(unit.Order, 0)
			if /* sth with controlling player || */ unit.State&32 > 0 {
				if distance == 1 && unit.Order == data.Defend {
					if unit.State&1 > 0 {
						unit.OrderLower4Bits = s.function10(unit.Order, 1)
					}
				}
			}
			ix, offset, moveCost := s.function6(temp, mvAdd, unit.X, unit.Y, unit.Type)
			if offset < 0 || offset >= len(s.mainGame.generic.Dx) {
				fmt.Println(ix, offset, moveCost)
			}
			sx = unit.X + s.mainGame.generic.Dx[offset]
			sy = unit.Y + s.mainGame.generic.Dy[offset]
			if i&64 > 0 {
				sx = unit.ObjectiveX
				sy = unit.ObjectiveY
				ix = s.CoordsToMapIndex(sx, sy)
				tt := s.terrainTypeAtIndex(ix)
				moveCost = s.mainGame.scenarioData.MoveCostPerTerrainTypesAndUnit[tt][unit.Type]
				mvAdd = 1
			}
			if s.ContainsUnitOfSide(sx, sy, unit.Side) {
				moveCost = 0
			}
			if s.ContainsUnitOfSide(sx, sy, 1-unit.Side) {
				moveCost = -1
			}
			if moveCost < 1 {
				if unit.Order != data.Attack || moveCost != -1 {
					if AbsInt(unit.ObjectiveX-unit.X)+AbsInt(unit.ObjectiveY-unit.Y) > 2 {
						if mvAdd == 0 {
							mvAdd = 2
							goto l5
						}
					}
				}
			}
			if moveCost < 1 {
				break
			}
			v := s.mainGame.scenarioData.Data192[unit.Formation] * moveCost / 8
			if unit.State&16 != 0 {
				v *= s.mainGame.scenarioData.UnitResupplyPerType[unit.Type] & 7
				v /= 8
			}
			v *= (512 - unit.Fatigue) / 32
			v *= s.mainGame.hexes.Arr3[unit.Side][unit.GeneralIndex][3] & 15
			v /= 16
			if unit.SupplyLevel == 0 {
				v /= 2
			}
			w := 1024
			if s.mainGame.scenarioData.UnitMask[unit.Type]&4 != 0 {
				w += weather * 128
			}
			w /= 8
			temp = w / (v + 1)
			if temp > v57 {
				if rand.Intn(temp) > v57 {
					break
				}
			}
			v57 -= temp
			if true /* todo: player controlled or sth */ {
				//function28
			} else if unit.State&65 > 0 {
				//function28
			}
			unit.X = sx
			unit.Y = sy
			// function29: add unit tile to map
			dist := s.function15(unit)
			if dist == 0 {
				unit.ObjectiveX = 0
				unit.OrderLower4Bits = s.function10(unit.Order, 1)
				if unit.Order == data.Defend || unit.Order == data.Move {
					if unit.State&32 == 0 {
						v9 = 6 // reached our objective, awaiting further orders
					}
				}
			}
			unit.Fatigue = ClampInt(unit.Fatigue+s.mainGame.scenarioData.Data173, 0, 255)
			if s.function16(unit) {
				v9 = 5 // have captured ....
				break
			}
			if v57 > 0 {
				if s.countNeighbourUnits(unit.X, unit.Y, 1-unit.Side) > 0 {
					unit.State |= 17
				} else {
					unit.State &= 254
				}
				// function29 (add own tile to map - no need here?)
				// todo: check conditions when unit should be visible
				continue
			}
			break
		}
		// l2:
		if false {
			fmt.Println(v57, v49, arg1)
		}
		unit.SupplyLevel = ClampInt(unit.SupplyLevel-2, 0, 255)
		t := unit.State & 1
		if rand.Intn(s.mainGame.scenarioData.Data252[unit.Side]) > 0 {
			unit.State &= 232
		} else {
			unit.State &= 168
		}
		for i := 0; i < 6; i++ {
			if unit2, ok := s.FindUnit(unit.X+s.mainGame.generic.Dx[i], unit.Y+s.mainGame.generic.Dy[i]); ok && unit2.Side == 1-unit.Side {
				// show unit2 on map
				unit2.State = unit2.State | 65
				// todo: save unit2
				s.mainGame.units[unit2.Side][unit2.Index] = unit2
				if s.mainGame.scenarioData.UnitScores[unit2.Type] > 8 {
					if true /* controlled by X */ {
						sx = unit2.X
						sy = unit2.Y
						unit.Order = data.Attack
						arg1 = 7
						//						arg2 = i
					}
					unit2Mask := s.mainGame.scenarioData.UnitMask[unit2.Type]
					if unit2Mask&128 == 0 {
						unit.State = unit.State | 16
					}
					if unit2Mask&64 == 0 { // unit cannot move ???
						unit.State = unit.State | 65
						if t == 0 {
							v9 = 4 // in contact with enemy forces
						}
					}
				}
			}
		}
		// function29 - add unit tile to map
		//	l11:
		if unit.ObjectiveX == 0 {
			goto end
		}
		if unit.Order != data.Attack {
			goto end
		}
		if arg1 < 7 {
			goto end
		}
		if distance == 1 {
			if s.ContainsUnitOfSide(sx, sy, unit.Side) {
				unit.ObjectiveX = 0
				goto end
			}
		}
		unit.OrderLower4Bits = s.function10(unit.Order, 2)
		if unit.Fatigue > 64 {
			goto end
		}
		if unit.SupplyLevel == 0 {
			goto end
		}
		if !s.ContainsUnitOfSide(sx, sy, 1-unit.Side) {
			goto end
		}
		if unit.Formation&7 != s.mainGame.scenarioData.Data178 {
			goto end
		}
		if unit.FormationHigher4Bits&8 == 0 {
			// * function28
			// * show unit on map
			// * function14
		} else if s.mainGame.scenarioData.Data32[unit.Type]&8 > 0 {
			if weather > 3 {
				// sth
				goto end
			}
			// function27
		}
		v9 = 1
		// [53767] = 0
		unit.State |= 65
		unit2, ok := s.FindUnit(sx, sy)
		if !ok || unit2.Side != 1-unit.Side {
			panic("")
		}
		arg1 = s.terrainTypeAt(unit.X, unit.Y)
		v := s.mainGame.scenarioData.Data96[arg1] * s.mainGame.scenarioData.Data128[unit.Formation&7] * unit.MenCount / 32
		if unit.FormationHigher4Bits&8 > 0 {
			v = 0
		}
		v2 := s.mainGame.scenarioData.Data104[arg1] * s.mainGame.scenarioData.Data136[unit.Formation&7] * (s.mainGame.scenarioData.Data16[unit.Type] / 16) / 2 * unit.EquipCount / 64
		if unit.FormationHigher4Bits&8 > 0 {
			if s.mainGame.scenarioData.Data32[unit.Type]&8 > 0 {
				w := 4 - weather
				if w < 1 {
					goto end
				}
				v2 = v2 * w / 4
			}
		}
		v = (v + v2) * unit.Morale / 255 * (255 - unit.Fatigue) / 128
		v = v * (s.mainGame.hexes.Arr3[unit.Side][unit.GeneralIndex][1] & 15) / 16
		v = v * s.magicCoeff(s.mainGame.hexes.Arr144[:], unit.X, unit.Y, unit.Side) / 8
		v++
		tt2 := s.terrainTypeAt(unit2.X, unit2.Y)
		if s.mainGame.scenarioData.UnitScores[unit2.Type]&258 > 0 {
			unit.State |= 4
		}
		menCoeff := s.mainGame.scenarioData.Data112[tt2] * s.mainGame.scenarioData.Data144[unit2.Formation&7] * unit2.MenCount / 32
		equipCoeff := s.mainGame.scenarioData.Data104[tt2] * s.mainGame.scenarioData.Data152[unit2.Formation&7] * (s.mainGame.scenarioData.Data16[unit2.Type] & 15) / 2 * unit2.EquipCount / 64
		w := (menCoeff + equipCoeff) * unit2.Morale / 256 * (240 - unit2.Fatigue/2) / 128 * (s.mainGame.hexes.Arr3[1-unit.Side][unit2.GeneralIndex][2] & 15) / 16
		if unit2.SupplyLevel == 0 {
			w = w * s.mainGame.scenarioData.Data167 / 8
		}
		w *= s.magicCoeff(s.mainGame.hexes.Arr144[:], unit2.X, unit2.Y, 1-unit.Side) / 8
		w++
		d := w / 16 / v
		if s.mainGame.scenarioData.UnitMask[unit.Type]&4 == 0 {
			d += weather
		}
		arg1 = ClampInt(w, 0, 63)
	}
end:
	s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2] = unit
}

// Has unit captured a city
func (s *ShowMap) function16(unit data.Unit) bool {
	if city, ok := s.FindCity(unit.X, unit.Y); ok {
		if city.Owner != unit.Side {
			// msg = 5
			city.Owner = unit.Side
			s.score13[unit.Side] += city.VictoryPoints
			s.score13[1-unit.Side] -= city.VictoryPoints
			s.score21[unit.Side] += city.VictoryPoints & 1
			return true
		}
	}
	return false
}
func (s *ShowMap) CoordsToMapIndex(x, y int) int {
	return y*s.mainGame.terrainMap.Width + x/2 - y/2
}

// TODO: deduplicate this function and FindBestMoveFromTowards()
func (s *ShowMap) function6(offsetIx, add, x, y, unitType int) (int, int, int) {
	ni := s.mainGame.generic.DirectionToNeighbourIndex[offsetIx]
	neigh1 := s.mainGame.generic.Neighbours[add][ni]
	offset1 := s.mainGame.generic.MapOffsets[neigh1]
	ix1 := s.CoordsToMapIndex(x, y) + offset1
	if !InRange(ix1, 0, len(s.mainGame.terrainMap.Terrain)) {
		return 0, 0, 0
	}
	tt1 := s.terrainTypeAtIndex(ix1)
	mc1 := s.mainGame.scenarioData.MoveCostPerTerrainTypesAndUnit[tt1][unitType]

	neigh2 := s.mainGame.generic.Neighbours[add+1][ni]
	offset2 := s.mainGame.generic.MapOffsets[neigh2]
	ix2 := s.CoordsToMapIndex(x, y) + offset2
	if !InRange(ix2, 0, len(s.mainGame.terrainMap.Terrain)) {
		return 0, 0, 0
	}
	tt2 := s.terrainTypeAtIndex(ix2)
	mc2 := s.mainGame.scenarioData.MoveCostPerTerrainTypesAndUnit[tt2][unitType]

	if mc2 > mc1-rand.Intn(2) {
		return ix2, neigh2, mc2
	}
	return ix1, neigh1, mc1
}

// arr is one of 48 element arrays in Hexes
func (s *ShowMap) magicCoeff(arr []int, x, y, side int) int {
	var bitmaps [5]byte
	for i := 5; i >= 0; i-- {
		bitmaps[0] <<= 2
		bitmaps[1] <<= 2
		bitmaps[4] <<= 2
		nx := x + s.mainGame.generic.Dx[i]
		ny := y + s.mainGame.generic.Dy[i]
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
	return SignInt(dy)*5 + SignInt(dx-dy)*3 + SignInt(dx+dy)
}
func (s *ShowMap) function10(order data.OrderType, offset int) byte {
	if offset < 0 || offset >= 4 {
		panic(offset)
	}
	return byte(s.mainGame.scenarioData.Data176[int(order)*4+offset])
}

// distance to objective
func (s *ShowMap) function15(unit data.Unit) int {
	dx := unit.ObjectiveX - unit.X
	dy := unit.ObjectiveY - unit.Y
	if AbsInt(dy) > AbsInt(dx)/2 {
		return AbsInt(dy)
	} else {
		return (AbsInt(dx) + AbsInt(dy) + 1) / 2
	}
}
func (s *ShowMap) function26(x, y int, val int, index int) int {
	v := s.mainGame.generic.Data214[((x/4)&1)*2+((y/2)&1)*18+index]
	if ((((x/2)&3)+1)&2)+((((y)&3)+1)&2) == 4 {
		v = ((index + 1) / 2) & 6
	}
	return val * v / 8
}

func (s *ShowMap) reinitSmallMapsAndSuch() {
	s.resetMaps()
	v13 := 0
	v15 := 0
	v16 := 0
	for _, sideUnits := range s.mainGame.units {
		for _, unit := range sideUnits {
			if unit.State&128 == 0 {
				// goto l23
				continue
			}
			if s.mainGame.scenarioData.UnitMask[unit.Type]&16 == 0 {
				sx, sy := unit.X/8, unit.Y/4
				if s.options.IsPlayerControlled(unit.Side) {
					v15 += unit.MenCount + unit.EquipCount
					v13 += 1
				} else {
					v16 += unit.MenCount + unit.EquipCount
					if false { // if full intelligence?? (unit.Side+1)&auto >>>4 > 0
						if unit.State&64 == 0 {
							continue
						}
					}
				}
				v30 := unit.MenCount + unit.EquipCount
				tmp := ClampInt(s.mainGame.scenarioData.Data144[unit.Formation&7], 8, 99) * v30 / 8
				v29 := s.mainGame.scenarioData.Data112[s.terrainTypeAt(unit.X, unit.Y)] * tmp / 8
				if s.mainGame.scenarioData.UnitScores[unit.Type] >= 7 {
					v29 = 4
					v30 = 4
				}
				s.map0[unit.Side][sx][sy] += (v30 + 4) / 8
				s.map3[unit.Side][sx][sy] = ClampInt(s.map3[unit.Side][sx][sy]+(v29+4)/8, 0, 255)
				if s.mainGame.scenarioData.ProbabilityOfUnitsUsingSupplies < unit.SupplyLevel-1 {
					v29 = s.mainGame.scenarioData.UnitScores[unit.Type] / 4
					if v29 > 0 {
						for v30 = -1; v30 <= v29; v30++ {
							for v6 := 0; v6 <= (AbsInt(v30)-SignInt(AbsInt(v30)))*4; v6++ {
								dx, dy := s.mainGame.generic.SmallMapOffsets(v6)
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
	}
	// function18();
	for _, city := range s.mainGame.terrain.Cities {
		if city.Owner != 0 || city.VictoryPoints != 0 {
			sx, sy := city.X/8, city.Y/4
			v29 := city.VictoryPoints / 8
			if v29 > 0 {
				s.map3[city.Owner][sx][sy]++
				for i := 1; i <= v29; i++ {
					for j := 0; j <= (i-1)*4; j++ {
						dx, dy := s.mainGame.generic.SmallMapOffsets(j)
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
	// function18();
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			s.map1[0][x][y] = s.map1[0][x][y] * s.mainGame.terrain.Coeffs[x][y] / 8
			s.map1[1][x][y] = s.map1[1][x][y] * s.mainGame.terrain.Coeffs[x][y] / 8
		}
	}
	// function18();
	for side := 0; side < 2; side++ {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				s.map2_0[side][x/4][y/4] += s.map0[side][x][y]
				s.map2_1[side][x/4][y/4] += s.map1[side][x][y]
			}
		}
	}
	// function18();
}

func (s *ShowMap) countNeighbourUnits(x, y, side int) int {
	num := 0
	for _, unit := range s.mainGame.units[side] {
		if unit.State&128 == 0 {
			continue
		}
		if AbsInt(unit.X-x)+AbsInt(2*(unit.Y-y)) < 4 { // TODO: double check it
			num++
		}
	}
	return num
}

func (s *ShowMap) everyHour() {
	if s.hour == 12 {
		s.every12Hours()
	}
	sunriseOffset := int(math.Abs(6.-float64(s.month)) / 2.)
	s.isNight = s.hour < 5+sunriseOffset || s.hour > 20-sunriseOffset

	if s.mainGame.scenarioData.ProbabilityOfUnitsUsingSupplies > 24*rand.Intn(256)/256 {
		for _, sideUnits := range s.mainGame.units {
			for i, unit := range sideUnits {
				if unit.State&128 == 0 {
					continue
				}
				if s.mainGame.scenarioData.UnitUsesSupplies[unit.Type] &&
					unit.SupplyLevel > 0 {
					unit.SupplyLevel--
					sideUnits[i] = unit
				}
			}
		}

	}

	s.createMapImage()
}
func (s *ShowMap) every12Hours() (reinforcements [2]bool) {
	s.supplyLevels[0] += s.mainGame.scenarioData.ResupplyRate[0]
	s.supplyLevels[1] += s.mainGame.scenarioData.ResupplyRate[1]
	if s.isNight { // if it's midnight
		for _, sideUnits := range s.mainGame.units {
			for i, unit := range sideUnits {
				if unit.State&128 != 0 {
					unit = s.resupplyUnit(unit)
				} else {
					if unit.HalfDaysUntilAppear == 0 {
						continue
					}
					unit.HalfDaysUntilAppear--
					if unit.HalfDaysUntilAppear != 0 {
						continue
					}
					shouldSpawnUnit := !s.ContainsUnit(unit.X, unit.Y) &&
						(unit.InvAppearProbability*rand.Intn(256))/256 > 0
					if city, ok := s.FindCity(unit.X, unit.Y); ok && city.Owner != unit.Side {
						shouldSpawnUnit = false
					}
					if shouldSpawnUnit {
						unit.State |= 128
						reinforcements[unit.Side] = true
						fmt.Println("Reinforcement ", unit.X, unit.Y)
					} else {
						unit.HalfDaysUntilAppear = 1
					}
				}
				sideUnits[i] = unit
			}
		}
	}
	for _, sideUnits := range s.mainGame.units {
		for i, unit := range sideUnits {
			m := unit.State & 136
			if m^136 != 0 { // has supply line
				if unit.MenCount <= s.mainGame.scenarioData.MenCountLimit[unit.Type] {
					unit.MenCount += (s.mainGame.scenarioData.MenReplacementRate[unit.Side] + 32) * rand.Intn(256) / 256 / 32
				}
				if unit.EquipCount <= s.mainGame.scenarioData.EquipCountLimit[unit.Type] {
					unit.EquipCount += (s.mainGame.scenarioData.EquipReplacementRate[unit.Side] + 32) * rand.Intn(256) / 256 / 32
				}
			}
			sideUnits[i] = unit
		}
	}
	return
}

func (s *ShowMap) resupplyUnit(unit data.Unit) data.Unit {
	unit.OrderLower4Bits &= 7
	if !s.mainGame.scenarioData.UnitUsesSupplies[unit.Type] ||
		!s.mainGame.scenarioData.UnitCanMove[unit.Type] {
		return unit
	}
	unit.State |= 32
	minSupplyType := s.mainGame.scenarioData.MinSupplyType & 15
	if unit.Type >= minSupplyType {
		// headquarters can only gain supply from supply depots,
		//  not other headquarters
		minSupplyType++
	}
	// keep the last friendly unit so that we can use it outside of the loop
	var supplyUnit data.Unit
outerLoop:
	for j := 0; j < len(s.mainGame.units[unit.Side]); j++ {
		supplyUnit = s.mainGame.units[unit.Side][j]
		if supplyUnit.Type < minSupplyType ||
			supplyUnit.State&128 == 0 || supplyUnit.SupplyLevel == 0 {
			continue
		}
		supplyX, supplyY := supplyUnit.X, supplyUnit.Y
		supplyTransportBudget := s.mainGame.scenarioData.MaxSupplyTransportCost
		if unit.Type == s.mainGame.scenarioData.MinSupplyType&15 {
			supplyTransportBudget *= 2
		}
		for supplyTransportBudget > 0 {
			dx, dy := unit.X-supplyX, unit.Y-supplyY
			if AbsInt(dx)+AbsInt(dy) < 3 {
				supplyLevel := s.supplyLevels[unit.Side]
				if supplyLevel != 0 {
					unitResupply := s.mainGame.scenarioData.UnitResupplyPerType[unit.Type]
					maxResupply := ClampInt(
						(supplyLevel-unit.SupplyLevel*2)/16,
						0,
						s.mainGame.scenarioData.MaxResupplyAmount)
					unitResupply = ClampInt(unitResupply, 0, maxResupply)
					unit.SupplyLevel += unitResupply
					s.supplyLevels[unit.Side] = supplyLevel - unitResupply
					unit.State &= 247
				}
				break outerLoop
			} else {
				var x, y, cost int
				for variant := 0; variant < 1; variant++ {
					x, y, cost = s.FindBestMoveFromTowards(supplyX, supplyY, unit.X, unit.Y, s.mainGame.scenarioData.MinSupplyType, variant)
					if cost != 0 {
						break
					}
				}
				//dx, dy := moveToXY(move)
				supplyX, supplyY = x, y
				if s.ContainsUnitOfSide(supplyX, supplyY, 1-unit.Side) {
					break
				}
				supplyTransportBudget -= 256 / (cost + 1)
			}
		}
	}
	if unit.SupplyLevel == 0 {
		unit.Fatigue = ClampInt(unit.Fatigue, 0, 255)
		// todo: does it really work? Aren't the last units on the list all zeroes...
		if supplyUnit.X != 0 {
			unit.ObjectiveX = supplyUnit.X
			unit.ObjectiveY = supplyUnit.Y
		}
	}
	return unit
}

func (s *ShowMap) ContainsUnit(x, y int) bool {
	return s.ContainsUnitOfSide(x, y, 0) ||
		s.ContainsUnitOfSide(x, y, 1)
}
func (s *ShowMap) ContainsUnitOfSide(x, y, side int) bool {
	for _, unit := range s.mainGame.units[side] {
		if (unit.State&128) != 0 && unit.X == x && unit.Y == y {
			return true
		}
	}
	return false
}
func (s *ShowMap) FindUnit(x, y int) (data.Unit, bool) {
	for _, sideUnits := range s.mainGame.units {
		for _, unit := range sideUnits {
			if unit.X == x && unit.Y == y {
				return unit, true
			}
		}
	}
	return data.Unit{}, false

}
func (s *ShowMap) FindCity(x, y int) (data.City, bool) {
	for _, city := range s.mainGame.terrain.Cities {
		if city.X == x && city.Y == y {
			return city, true
		}
	}
	return data.City{}, false
}

// function17
func (s *ShowMap) terrainType(terrain byte) int {
	return s.mainGame.generic.TerrainTypes[terrain&63]
}

func (s *ShowMap) terrainTypeAt(x, y int) int {
	return s.terrainType(s.terrainAt(x, y))
}
func (s *ShowMap) terrainTypeAtIndex(ix int) int {
	return s.terrainType(s.terrainAtIndex(ix))
}

func (s *ShowMap) terrainAt(x, y int) byte {
	if unit, ok := s.FindUnit(x, y); ok {
		return byte(unit.Type + unit.ColorPalette*64)
	}
	return s.mapTerrainAt(x, y)
}
func (s *ShowMap) terrainAtIndex(ix int) byte {
	for _, sideUnits := range s.mainGame.units {
		for _, unit := range sideUnits {
			if s.CoordsToMapIndex(unit.X, unit.Y) == ix {
				return byte(unit.Type + unit.ColorPalette*64)
			}
		}
	}
	return s.mainGame.terrainMap.Terrain[ix]
}

func (s *ShowMap) mapTerrainAt(x, y int) byte {
	x /= 2
	if !InRange(y, 0, s.mainGame.terrainMap.Height) ||
		!InRange(x, 0, s.mainGame.terrainMap.Width-y%2) {
		return 0
	}
	return s.mainGame.terrainMap.GetTile(x, y)
}

func (s *ShowMap) FindBestMoveFromTowards(supplyX, supplyY, unitX, unitY, unitType, variant int) (int, int, int) {
	dx, dy := unitX-supplyX, unitY-supplyY
	neighbour1 := s.mainGame.generic.DxDyToNeighbour(dx, dy, 2*variant)
	supplyX1 := supplyX + s.mainGame.generic.Dx[neighbour1]
	supplyY1 := supplyY + s.mainGame.generic.Dy[neighbour1]
	// in the original code the source and target spots in the terrain map are filled
	// with the unit tiles, but it *shouldn't* impact the logic here.
	// also in original code there's map offset used not x,y coords.
	terrainType1 := s.terrainTypeAt(supplyX1, supplyY1)
	cost1 := s.mainGame.scenarioData.MoveCostPerTerrainTypesAndUnit[terrainType1][unitType]
	neighbour2 := s.mainGame.generic.DxDyToNeighbour(dx, dy, 2*variant+1)
	supplyX2 := supplyX + s.mainGame.generic.Dx[neighbour2]
	supplyY2 := supplyY + s.mainGame.generic.Dy[neighbour2]
	terrainType2 := s.terrainTypeAt(supplyX2, supplyY2)
	cost2 := s.mainGame.scenarioData.MoveCostPerTerrainTypesAndUnit[terrainType2][unitType]
	if cost2 < cost1-rand.Intn(1) {
		return supplyX2, supplyY2, cost2
	}
	return supplyX1, supplyY1, cost1
}

func (s *ShowMap) everyDay() {
	var flashback []data.FlashbackUnit
	for _, sideUnits := range s.mainGame.units {
		for _, unit := range sideUnits {
			if unit.State&128 != 0 {
				flashback = append(flashback, data.FlashbackUnit{
					X: unit.X, Y: unit.Y, ColorPalette: unit.ColorPalette, Type: unit.Type,
				})
			}
		}
	}
	s.flashback = append(s.flashback, flashback)
	// todo: save todays map for flashback
	rnd := rand.Intn(256)
	if rnd < 140 {
		s.weather = int(s.mainGame.scenarioData.PossibleWeather[4*(s.month/3)+rnd/35])
	}
	s.every12Hours()
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
func (s *ShowMap) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(s.dx)*(-8), float64(s.dy)*(-8))
	screen.DrawImage(s.mapImage, opts)
	for _, sideUnits := range s.mainGame.units {
		for _, unit := range sideUnits {
			if unit.State&128 == 0 {
				continue
			}
			unitImg := *s.mainGame.sprites.UnitSymbolSprites[unit.Type]
			if !s.isNight {
				unitImg.Palette = data.GetPalette(unit.ColorPalette, s.mainGame.scenarioData.DaytimePalette)
			} else {
				unitImg.Palette = data.GetPalette(unit.ColorPalette, s.mainGame.scenarioData.NightPalette)
			}
			unitEImg, err := ebiten.NewImageFromImage(&unitImg, ebiten.FilterNearest)
			if err != nil {
				panic(err)
			}
			originalGeoM := opts.GeoM
			opts.GeoM.Translate(float64(unit.X)*4, float64(unit.Y)*8)
			screen.DrawImage(unitEImg, opts)
			opts.GeoM = originalGeoM
		}
	}
	hour := s.hour
	meridianString := "AM"
	if hour >= 12 {
		hour -= 12
		meridianString = "PM"
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("%02d:%02d %s %s, %d %d  %s", hour, s.minute, meridianString, s.mainGame.scenarioData.Months[s.month], s.day+1, s.year, s.mainGame.scenarioData.Weather[s.weather]))
}
func (s *ShowMap) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 320, 192
}
