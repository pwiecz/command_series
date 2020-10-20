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
nextUnit:
	s.lastUpdatedUnit = (s.lastUpdatedUnit + 1) % 128
	unit := s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2]
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
	if s.countNeighbourUnits(unit.X, unit.Y, 1-unit.Side) == 0 {
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
				unit.X /= 4
				unit.Y /= 4
				arg1 := 48000
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
					//funArg := (s.map2_1[unit.Side][x][y] + s.map2_1[1-unit.Side][x][y]) * 16 / ClampInt(s.map2_0[unit.Side][x][y]-s.map2_0[1-unit.Side][x][y], 10, 9999)
					var tmp int
					// tmp = function26(funArg)
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
				unit = s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2]
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
			v58 := s.mainGame.hexes.Arr3[unit.Side][unit.GeneralIndex]
			if false {
				fmt.Println(v58)
			}
		}
	}
	//...
l21:
l24:
end:
	s.mainGame.units[s.lastUpdatedUnit%2][s.lastUpdatedUnit/2] = unit
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
					if false { // if full intelligence??
						if unit.State&64 == 0 {
							continue
						}
					}
				}
				v30 := unit.MenCount + unit.EquipCount
				tmp := ClampInt(s.mainGame.scenarioData.Data144[unit.Formation&7], 8, 99) * v30 / 8
				v29 := s.mainGame.scenarioData.Data112[s.terrainType(s.terrainAt(unit.X/2, unit.Y)&63)] * tmp / 8
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
					shouldSpawnUnit := !s.ContainsUnitOfSide(unit.X, unit.Y, 0) &&
						!s.ContainsUnitOfSide(unit.X, unit.Y, 1) &&
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

func (s *ShowMap) ContainsUnitOfSide(x, y, side int) bool {
	for _, unit := range s.mainGame.units[side] {
		if (unit.State&128) != 0 && unit.X == x && unit.Y == y {
			return true
		}
	}
	return false
}
func (s *ShowMap) FindCity(x, y int) (data.City, bool) {
	for _, city := range s.mainGame.terrain.Cities {
		if city.X == x && city.Y == y {
			return city, true
		}
	}
	return data.City{}, false
}

func (s *ShowMap) terrainType(terrain byte) int {
	return s.mainGame.generic.TerrainTypes[terrain&63]
}
func (s *ShowMap) terrainAt(x, y int) byte {
	if y >= 0 && y < len(s.mainGame.terrainMap.Terrain) &&
		x >= 0 && x < len(s.mainGame.terrainMap.Terrain[y]) {
		return s.mainGame.terrainMap.Terrain[y][x]
	}
	return 0
}

func (s *ShowMap) FindBestMoveFromTowards(supplyX, supplyY, unitX, unitY, unitType, variant int) (int, int, int) {
	dx, dy := unitX-supplyX, unitY-supplyY
	neighbour1 := s.mainGame.generic.DxDyToNeighbour(dx, dy, 2*variant)
	supplyX1 := supplyX + s.mainGame.generic.Dx[neighbour1]
	supplyY1 := supplyY + s.mainGame.generic.Dy[neighbour1]
	// in the original code the source and target spots in the terrain map are filled
	// with the unit tiles, but it *shouldn't* impact the logic here.
	// also in original code there's map offset used not x,y coords.
	// TODO: check if using just x/2 is ok
	terrainType1 := s.terrainType(s.terrainAt(supplyX1/2, supplyY1) & 63)
	cost1 := s.mainGame.scenarioData.MoveCostPerTerrainTypesAndUnit[terrainType1][unitType]
	neighbour2 := s.mainGame.generic.DxDyToNeighbour(dx, dy, 2*variant+1)
	supplyX2 := supplyX + s.mainGame.generic.Dx[neighbour2]
	supplyY2 := supplyY + s.mainGame.generic.Dy[neighbour2]
	terrainType2 := s.terrainType(s.terrainAt(supplyX2/2, supplyY2) & 63)
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
