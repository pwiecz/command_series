package lib

import (
	"fmt"
	"math/rand"
)

type AI struct {
	rand *rand.Rand

	game Game

	commanderFlags *CommanderFlags
	scenarioData   *Data
	terrain        *Terrain
	terrainTypes   *TerrainTypeMap
	generic        *Generic
	hexes          *Hexes
	units          *Units
	score          *Score

	// Side of the most recently updated unit. Used for detecting moment when we switch analysing sides.
	update          int
	lastUpdatedUnit int

	map0 [2][16][16]int // Location of troops
	map1 [2][16][16]int // Location of important objects (supply units, air wings, important cities...)
	map3 [2][16][16]int
	// Aggregated versions of map0, map1 to 4 times lower resolution.
	map2_0, map2_1 [2][4][4]int // 0x400 - two byte values
}

func newAI(rand *rand.Rand, commanderFlags *CommanderFlags, gameData *GameData, scenarioData *ScenarioData, score *Score) *AI {
	return &AI{
		update:          3,
		lastUpdatedUnit: 127,
		rand:            rand,
		commanderFlags:  commanderFlags,
		game:            gameData.Game,
		scenarioData:    scenarioData.Data,
		terrain:         scenarioData.Terrain,
		terrainTypes:    gameData.TerrainTypeMap,
		generic:         gameData.Generic,
		hexes:           gameData.Hexes,
		units:           scenarioData.Units,
		score:           score}
}

func (s *AI) UpdateUnit(weather int, isNight bool, sync *MessageSync) (message MessageFromUnit, quit bool) {
	if isNight {
		weather += 8
	}
nextUnit:
	s.lastUpdatedUnit = (s.lastUpdatedUnit + 1) % 128
	unit := s.units[s.lastUpdatedUnit/64][s.lastUpdatedUnit%64]
	if !unit.IsInGame {
		goto nextUnit
	}
	if !s.areUnitCoordsValid(unit.XY) {
		panic(fmt.Errorf("%s@(%v):%v", unit.FullName(), unit.XY, unit))
	}
	var arg1 int
	if unit.MenCount+unit.TankCount < 7 || unit.Fatigue == 255 {
		s.terrainTypes.hideUnit(unit)
		message = WeMustSurrender{unit}
		unit.ClearState()
		unit.HalfDaysUntilAppear = 0
		s.score.CitiesHeld[1-unit.Side] += s.scenarioData.UnitScores[unit.Type]
		s.score.MenLost[unit.Side] += unit.MenCount
		s.score.TanksLost[unit.Side] += unit.TankCount
		goto end
	}
	if !s.scenarioData.UnitCanMove[unit.Type] {
		goto nextUnit
	}
	arg1 = s.updateUnitObjective(&unit, weather)
	if unit.SupplyLevel == 0 {
		message = WeHaveExhaustedSupplies{unit}
	}
	{
		sxy, shouldQuit := s.performUnitMovement(&unit, &message, &arg1, weather, sync)
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
			nxy := IthNeighbour(unit.XY, i)
			if unit2, ok := s.units.FindUnitOfSideAt(nxy, 1-unit.Side); ok {
				unit2.InContactWithEnemy = true
				unit2.SeenByEnemy = true // |= 65
				s.terrainTypes.showUnit(unit2)
				s.units[unit2.Side][unit2.Index] = unit2
				if s.scenarioData.UnitScores[unit2.Type] > 8 {
					if !s.commanderFlags.PlayerControlled[unit.Side] {
						sxy = unit2.XY
						unit.Order = Attack
						arg1 = 7
						// arg2 = i
					}
				}
				// in CiE one of supply units or an air wing.
				// in DitD also minefield or artillery
				// in CiV supply units or bombers (not fighters nor artillery)
				if !s.scenarioData.UnitMask7[unit2.Type] {
					unit.State4 = true // |= 16
				}
				if s.scenarioData.UnitCanMove[unit2.Type] {
					unit.InContactWithEnemy = true
					unit.SeenByEnemy = true // |= 65
					if !wasInContactWithEnemy {
						message = WeAreInContactWithEnemy{unit}
					}
				}
			}
		}
		s.function29_showUnit(unit)
		//	l11:
		if unit.Objective.X == 0 || unit.Order != Attack || arg1 < 7 {
			goto end
		}
		if unit.Function15_distanceToObjective() == 1 && s.units.IsUnitOfSideAt(sxy, unit.Side) {
			unit.Objective.X = 0
			goto end
		}
		unit.TargetFormation = s.scenarioData.function10(unit.Order, 2)
		if unit.Fatigue > 64 || unit.SupplyLevel == 0 ||
			!s.units.IsUnitOfSideAt(sxy, 1-unit.Side) ||
			unit.Formation != s.scenarioData.Data176[0][2] {
			goto end
		}
		// [53767] = 0
		if s.performAttack(&unit, sxy, weather, &message, sync) {
			quit = true
			return
		}
	}
end: // l3
	for unit.Formation != unit.TargetFormation {
		dir := Sign(unit.Formation - unit.TargetFormation)
		speed := s.scenarioData.FormationChangeSpeed[(dir+1)/2][unit.Formation]
		if speed > Rand(15, s.rand) {
			unit.LongRangeAttack = false
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

func (s *AI) updateUnitObjective(unit *Unit, weather int) int {
	res := s.updateUnitObjectiveAux(unit, weather)
	// l21:
	s.update = unit.Side
	return res
}

func (s *AI) updateUnitObjectiveAux(unit *Unit, weather int) int {
	numEnemyNeighbours := s.units.NeighbourUnitCount(unit.XY, 1-unit.Side)
	if numEnemyNeighbours == 0 {
		unit.State4 = false // &= 239
	}

	mode, needsObjective := s.bestOrder(unit, &numEnemyNeighbours)
	if !needsObjective {
		return 0 // goto l21
	}
	// l24:
	var arg1 int
	unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
	if mode == Attack {
		if objXY, score := s.bestAttackObjective(*unit, weather, numEnemyNeighbours); objXY.X > 0 {
			unit.Objective = objXY
			arg1 = score
		}
	}
	if mode == Reserve {
		unit.Objective.X = 0
	}
	if mode == Defend {
		// Reset current objective.
		if unit.Objective.X > 0 {
			unit.Objective = unit.XY
		}
		objXY, score := s.bestDefenceObjective(*unit)
		arg1 = score
		if objXY != unit.XY {
			unit.Objective = objXY
		} else {
			unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
		}
	}
	{
		// long range attack
		attackRange := s.scenarioData.AttackRange[unit.Type] * 2
		susceptibleToWeather := s.scenarioData.Data32_8[unit.Type]
		if s.game == Conflict {
			susceptibleToWeather = s.scenarioData.Data32_32[unit.Type]
		}
		if attackRange > 0 && (!susceptibleToWeather || weather < 2) && unit.Fatigue/4 < 32 {
			for i := 0; i <= 32-unit.Fatigue/4; i++ {
				unit2 := s.units[1-unit.Side][Rand(64, s.rand)]
				if Abs(unit.XY.X-unit2.XY.X)/2+Abs(unit.XY.Y-unit2.XY.Y) <= attackRange &&
					((s.game != Conflict && (unit2.IsUnderAttack || unit2.State2)) ||
						(s.game == Conflict && unit2.SeenByEnemy)) {
					unit.Objective = unit2.XY
					unit.Order = Attack
					unit.Formation = s.scenarioData.Data176[0][2]
				}
			}
		}
	}
	return arg1
}

func (s *AI) resetMaps() {
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

// If the side is controlled by the computer create the strategy level maps aggregating locations and numbers of units, important locations and such.
func (s *AI) reinitSmallMapsAndSuch(currentSide int) {
	s.resetMaps()
	// Those variables in the original code do not seem to play any role
	//v13 := 0
	//v15 := 0
	//v16 := 0
	for _, sideUnits := range s.units {
		for _, unit := range sideUnits {
			if !unit.IsInGame || s.scenarioData.UnitMask4[unit.Type] {
				continue // goto l23
			}
			sx, sy := unit.XY.X/8, unit.XY.Y/4
			if !InRange(sx, 0, 16) || !InRange(sy, 0, 16) {
				continue
			}
			if unit.Side == currentSide {
				//v15 += unit.MenCount + unit.TankCount
				//v13 += 1
			} else {
				//v16 += unit.MenCount + unit.TankCount
				if !s.commanderFlags.PlayerHasIntelligence[currentSide] && !unit.SeenByEnemy {
					continue // goto l23
				}
			}
			v30 := unit.MenCount + unit.TankCount
			v29 := v30 * Max(s.scenarioData.FormationMenDefence[unit.Formation], 8) / 8
			tt := s.terrainTypes.terrainTypeAt(unit.XY)
			v29 = v29 * s.scenarioData.TerrainMenDefence[tt] / 8
			if s.scenarioData.UnitScores[unit.Type] > 7 {
				// special units - supply, air wings
				v29 = 4
				v30 = 4
			}
			s.map0[unit.Side][sx][sy] += (v30 + 4) / 8
			s.map3[unit.Side][sx][sy] = Clamp(s.map3[unit.Side][sx][sy]+(v29+4)/8, 0, 255)
			if unit.SupplyLevel-1 <= s.scenarioData.AvgDailySupplyUse {
				continue
			}
			// An "influence" of the unit on the surrounding squares on the "small" map.
			influence := s.scenarioData.UnitScores[unit.Type] / 4
			if influence <= 0 {
				continue
			}
			// Mark the "influence" of the unit on concentric circles around the unit position.
			// The influence gets smaller, further away we get.
			for radius := -1; radius <= influence; radius++ {
				// Last index on a "circle" with given radius.
				lastNeighbour := (Abs(radius) - Sign(Abs(radius))) * 4
				for i := 0; i <= lastNeighbour; i++ {
					dx, dy := SmallMapOffsets(i)
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
			// l23:
		}
	}
	// function18()
	for _, city := range s.terrain.Cities {
		if city.VictoryPoints == 0 {
			continue
		}
		sx, sy := city.XY.X/8, city.XY.Y/4
		influence := city.VictoryPoints / 8
		if influence <= 0 {
			continue
		}
		// Mark the "influence" of the city on concentric circles around the city position.
		// The influence gets smaller, further away we get.
		s.map3[city.Owner][sx][sy]++
		for i := 1; i <= influence; i++ {
			for j := 0; j <= (i-1)*4; j++ {
				dx, dy := SmallMapOffsets(j)
				x, y := sx+dx, sy+dy
				if !InRange(x, 0, 16) || !InRange(y, 0, 16) {
					continue
				}
				s.map1[city.Owner][x][y] += 2
			}
		}
	}
	// function18()
	for side := 0; side < 2; side++ {
		for x := 0; x < 16; x++ {
			for y := 0; y < 16; y++ {
				s.map1[side][x][y] = s.map1[side][x][y] * s.terrain.Coeffs[x][y] / 8
				s.map2_0[side][x/4][y/4] += s.map0[side][x][y]
				s.map2_1[side][x/4][y/4] += s.map1[side][x][y]
			}
		}
	}
	// function18()
}

// Multiplier related to closeness of unit to a unit in a neighbouring square.
// neighbourIndex from [0, 8]
func (s *AI) function26(xy UnitCoords, neighbourIndex int) int {
	nx, ny := SmallMapOffsets(neighbourIndex)
	// If not on the edge of a 4x4 square
	if InRange((xy.X/2)%4, 1, 3) && InRange(xy.Y%4, 1, 3) {
		return 9 - 2*(Abs(nx)+Abs(ny))
	}
	if nx == 0 && ny == 0 {
		return 9
	}
	// Which quadrant of a 4x4 square does it occupy [0,0],[0,1],[1,0],[1,1].
	sx, sy := (xy.X/4)&1, (xy.Y/2)&1
	// As the direction points around 4x4 square make the nx,ny point outside the square.
	if nx > 0 {
		nx++
	}
	if ny > 0 {
		ny++
	}
	dx, dy := Abs(nx-sx), Abs(ny-sy)
	if nx == 0 {
		dx = 0
	}
	if ny == 0 {
		dy = 0
	}
	return 10 - Min(dx, dy) - 2*Max(dx, dy)
}

// Find best order to be performed by the unit.
// If both the objective and the order are already specified return false meaning the unit
// does not need its objective to be recalculated.
func (s *AI) bestOrder(unit *Unit, numEnemyNeighbours *int) (OrderType, bool) {
	var mode OrderType
	if s.commanderFlags.PlayerControlled[unit.Side] {
		s.update = unit.Side
		// If not a local command and either objective is specified, or order is defend or move).
		if !unit.HasLocalCommand && (unit.Order == Defend || unit.Order == Move || unit.Objective.X != 0) {
			return 0, false // goto l21
		} else {
			mode = unit.Order
			unit.HasLocalCommand = true // |= 32
			return mode, true           // goto l24
		}
	} else {
		if unit.OrderBit4 {
			return unit.Order, true // goto l24
		}
	}

	if s.update != unit.Side {
		s.reinitSmallMapsAndSuch(unit.Side)
	}
	{
		// v57 := sign(sign_extend([29927 + 10 + unit.side])/16)*4
		sx, sy := unit.XY.X/8, unit.XY.Y/4
		// Num enemy troops nearby (neighbouring "small" map fields).
		numEnemyTroops := 0
		for neighbourIx := 0; neighbourIx < 9; neighbourIx++ {
			dx, dy := SmallMapOffsets(neighbourIx)
			if InRange(sx+dx, 0, 16) && InRange(dy+sy, 0, 16) {
				numEnemyTroops += s.map0[1-unit.Side][sx+dx][sy+dy]
			}
		}
		// If there are no enemy units in neaby "small" map and there is a supply line to unit and sth (not a special unit?) then look at the "tiny" map.
		if numEnemyTroops == 0 && unit.HasSupplyLine &&
			((s.game != Conflict && s.scenarioData.UnitScores[unit.Type] <= 7) ||
				(s.game == Conflict && !s.scenarioData.UnitMask0[unit.Type])) {
			tx, ty := unit.XY.X/32, unit.XY.Y/16
			bestVal := -17536 // 48000
			bestNeighbourIx := 0
			bestX, bestY := 0, 0
			for neighbourIx := 0; neighbourIx < 9; neighbourIx++ {
				dx, dy := TinyMapOffsets(neighbourIx)
				x, y := tx+dx, ty+dy
				if !InRange(x, 0, 4) || !InRange(y, 0, 4) {
					continue
				}
				// Coords are a good target if there are more high importance objects (supply units, air wings, cities with high vp), and less good target if there are already many friendly units.
				val := (s.map2_1[unit.Side][x][y] + s.map2_1[1-unit.Side][x][y]) * 16 / Clamp(s.map2_0[unit.Side][x][y]-s.map2_0[1-unit.Side][x][y], 10, 9999)
				val = val * s.function26(UnitCoords{unit.XY.X / 4, unit.XY.Y / 4}, neighbourIx) / 8
				if neighbourIx == 0 {
					// Prioritize staying withing the same square.
					val *= 2
				}
				if val > bestVal {
					bestVal = val
					bestNeighbourIx = neighbourIx
					bestX, bestY = x, y
				}
			}
			// Set unit objective to the center of the target square.
			if bestNeighbourIx > 0 {
				unit.TargetFormation = 0
				unit.OrderBit4 = false
				unit.Order = Reserve
				count := (unit.MenCount + unit.TankCount + 8) / 16
				s.map2_0[unit.Side][tx][ty] = Abs(s.map2_0[unit.Side][bestX][bestY] - count)
				s.map2_0[unit.Side][bestX][bestY] += count
				unit.Objective = UnitCoords{bestX*32 + 16, bestY*16 + 8}
				if s.game == Conflict {
					unit.Objective.X += Rand(3, s.rand) * 2
				}
				return 0, false //goto l21
			}
		}
		{
			bestVal := -17536 // 48000
			//var bestI int
			var bestDx, bestDy int
			var v63 int
			temp2 := (unit.MenCount + unit.TankCount + 4) / 8
			v61 := temp2 * Max(s.scenarioData.FormationMenDefence[unit.Formation], 8) / 8
			tt := s.terrainTypes.terrainTypeAt(unit.XY)
			v61 = v61 * s.scenarioData.TerrainMenDefence[tt] / 8
			if s.scenarioData.UnitScores[unit.Type] > 7 {
				// special units - air wings or supply units
				temp2 = 1
				v61 = 1
			}
			// Subtract impact of the unit itself.
			s.map0[unit.Side][sx][sy] = Clamp(s.map0[unit.Side][sx][sy]-temp2, 0, 255)
			s.map3[unit.Side][sx][sy] = Clamp(s.map3[unit.Side][sx][sy]-v61, 0, 255)
			// save a copy of the unit, as we're going to modify it.
			unitCopy := *unit
			for neighbourIx := 0; neighbourIx <= 8; neighbourIx++ {
				dx, dy := SmallMapOffsets(neighbourIx)
				if !InRange(sx+dx, 0, 16) || !InRange(sy+dy, 0, 16) {
					continue
				}
				v54 := 0
				v49 := 0
				v50 := 0
				v53 := 0
				friendlyUnitsInArea := s.map0[unit.Side][sx+dx][sy+dy]
				enemyUnitsInArea := s.map0[1-unit.Side][sx+dx][sy+dy]
				v36 := (friendlyUnitsInArea + s.map3[unit.Side][sx+dx][sy+dy]) / 2
				v52 := (enemyUnitsInArea + s.map3[1-unit.Side][sx+dx][sy+dy]) / 2
				enemyUnitsAround := s.map0[1-unit.Side][sx+dx][sy+dy] / 2
				for j := 1; j <= 8; j++ {
					ddx, ddy := SmallMapOffsets(j)
					if !InRange(sx+dx+ddx, 0, 16) || !InRange(sy+dy+ddy, 0, 16) {
						continue
					}
					v := s.map0[1-unit.Side][sx+dx+ddx][sy+dy+ddy] / 4
					if j >= 5 { // diagonals(?)
						v /= 2
					}
					enemyUnitsAround += v
				}
				temp := Reserve
				if s.map3[1-unit.Side][sx+dx][sy+dy] > 0 {
					temp = Attack
				}
				// Two iterations: one not taking into consideration the current unit, and once taking it into consideration.
				for j := 0; j < 2; j++ {
					var v48 int
					if friendlyUnitsInArea > v52 {
						v48 = Clamp((friendlyUnitsInArea+1)*8/(v52+1)-7, 0, 16)
					} else {
						v48 = -Clamp((v52+1)*8/(friendlyUnitsInArea+1)-8, 0, 16)
					}
					v48 += unit.General.Data1High + s.scenarioData.Data0High[unit.Type]
					var v55 int
					if v36 > enemyUnitsAround {
						v55 = Clamp((v36+1)*8/(enemyUnitsAround+1)-7, 0, 16)
					} else {
						v55 = -Clamp((enemyUnitsAround+1)*8/(v36+1)-8, 0, 16)
					}
					if v48 > 0 {
						v := v48 * s.map1[1-unit.Side][sx+dx][sy+dy]
						if unit.SeenByEnemy {
							v /= 2 /* logical shift not the arithmetic one, actually) */
						}
						if unit.General.Data0_2 {
							v *= 2
						}
						if unit.General.Data0_6 {
							v /= 2
						}
						if j > 0 {
							v += s.map1[unit.Side][sx+dx][sy+dy] * 8 / friendlyUnitsInArea
						}
						v54 += v
					}
					if v55 < 0 {
						temp = Reserve
						if enemyUnitsInArea > 0 {
							v := s.map1[unit.Side][sx+dx][sy+dy] * v55
							if unit.General.Data0_1 {
								v *= 2
							}
							if unit.General.Data0_5 {
								v /= 2
							}
							v53 += v
						}
					}
					if v48 > 0 {
						if *numEnemyNeighbours > 0 {
							temp = Attack
						}
						if enemyUnitsInArea > 0 {
							v := v48
							if unit.General.Data0_3 {
								v *= 2
							}
							if unit.General.Data0_7 {
								v /= 2
							}
							v *= enemyUnitsInArea
							v49 += v
						}
					}
					if v55 < 0 {
						if friendlyUnitsInArea > 0 {
							temp = Defend
							v := friendlyUnitsInArea * v55
							if unit.General.Data0_0 {
								v *= 2
							}
							if unit.General.Data0_4 {
								v /= 2
							}
							v50 += v
						}
						if v55+unit.General.Data2High+s.scenarioData.Data0Low[unit.Type] < -9 {
							if j == neighbourIx+1 {
								unit.Fatigue += 256
							}
						}
					}
					if j == 0 {
						v54 = -v54
						v53 = -v53
						v49 = -v49
						v50 = -v50
						friendlyUnitsInArea += temp2
						v36 += v61
					}
				}
				t := v54 + v53 + v49 + v50
				if neighbourIx == 0 {
					if _, ok := s.terrain.FindCityAt(unit.XY); ok {
						if enemyUnitsInArea > 0 {
							*numEnemyNeighbours = 2
						}
					}
				}
				v := s.scenarioData.UnitScores[unit.Type] & 248
				if unit.InContactWithEnemy {
					v += unit.Fatigue/16 + unit.Fatigue/32
				}
				if v > 7 {
					t = v36 - v52*2
					*numEnemyNeighbours = -128
					temp = Reserve
					unit.Fatigue &= 255
				}
				t = t * s.function26(unit.XY, neighbourIx) / 8
				if neighbourIx == 0 {
					v63 = t
					mode = temp
				}
				if t > bestVal {
					bestVal = t
					bestDx, bestDy = dx, dy
					//bestI = i
				}
				if neighbourIx+2 > Sign(int(mode))+*numEnemyNeighbours {
					continue
				}
				break
			}
			// function18()
			*unit = unitCopy // revert modified unit
			unit.OrderBit4 = true
			supplyUse := s.scenarioData.AvgDailySupplyUse
			if !unit.HasSupplyLine {
				supplyUse *= 2
			}
			if unit.SupplyLevel < supplyUse {
				supplyUnit := s.units[unit.Side][unit.SupplyUnit]
				if !supplyUnit.IsInGame {
					supplyUnit = s.units[unit.Side][supplyUnit.SupplyUnit]
				}
				unit.Objective = supplyUnit.XY
				t := Move
				if *numEnemyNeighbours > 0 {
					t = Defend
				}
				unit.Order = t
				unit.TargetFormation = 0
				unit.OrderBit4 = false
				return 0, false //goto l21
			}
			if s.game == Conflict && s.scenarioData.UnitMask0[unit.Type] {
				bestDx, bestDy = 0, 0
			}
			if unit.Fatigue*4 > bestVal-v63 {
				bestDx, bestDy = 0, 0
			}
			if bestDx == 0 && bestDy == 0 {
				if unit.Fatigue > 64 {
					mode = Defend
				}
				if mode == Reserve {
					mode = Defend
				}
				// Revert previous values, if we're not moving the unit
				s.map0[unit.Side][sx][sy] += temp2
				s.map3[unit.Side][sx][sy] += v61
				// update = 13
			} else {
				if s.map0[unit.Side][sx+bestDx][sy+bestDy] > 0 {
					s.map0[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				}
				s.map3[unit.Side][sx+bestDx][sy+bestDy] += temp2 / 2
				unit.Objective.Y = ((sy+bestDy)*4 + Rand(2, s.rand) + 1) & 63
				unit.Objective.X = (((sx+bestDx)*4+Rand(2, s.rand)+1)*2 + (unit.Objective.Y & 1)) & 127
				mode = Move
				if *numEnemyNeighbours != 0 {
					unit.Order = Defend
					return mode, true // goto l24
				}
			}
			unit.Order = mode
		}
	}
	return mode, true
}

func (s *AI) bestAttackObjective(unit Unit, weather int, numEnemyNeighbours int) (UnitCoords, int) {
	var bestObjective UnitCoords
	bestScore := 16000
	terrainType := s.terrainTypes.terrainTypeAt(unit.XY)
	menCoeff := s.scenarioData.TerrainMenAttack[terrainType] * unit.MenCount
	tankCoeff := s.scenarioData.TerrainTankAttack[terrainType] * unit.TankCount * s.scenarioData.Data16High[unit.Type] / 4
	coeff := (menCoeff + tankCoeff) / 8 * (255 - unit.Fatigue) / 256 * (unit.Morale + s.scenarioData.Data0High[unit.Type]*16) / 128
	temp2 := coeff * s.NeighbourScore(&s.hexes.Arr144, unit.XY, unit.Side) / 8
	v := 0
	if numEnemyNeighbours > 0 && s.scenarioData.Data200Low[unit.Type] < 3 {
		// Only consider direct nighbours of the unit
		v = 12
	}
	for i := v; i <= 18; i++ {
		score := 16001
		dx, dy := LongRangeHexNeighbourOffset(i)
		nxy := UnitCoords{unit.XY.X + dx, unit.XY.Y + dy}
		if !s.areUnitCoordsValid(nxy) {
			continue
		}
		if unit2, ok := s.units.FindUnitOfSideAt(nxy, 1-unit.Side); ok {
			terrainType := s.terrainTypes.terrainTypeAt(unit2.XY)
			menCoeff := s.scenarioData.TerrainMenDefence[terrainType] * unit2.MenCount
			tankCoeff := s.scenarioData.TerrainTankDefence[terrainType] * unit2.TankCount * s.scenarioData.Data16Low[unit2.Type] / 4
			t := (menCoeff + tankCoeff) * s.scenarioData.FormationMenDefence[unit2.Formation] / 8
			w := weather
			if s.game != Conflict && s.scenarioData.UnitMask2[unit.Type] {
				w /= 2
			}
			if s.game == Conflict && !s.scenarioData.UnitMask2[unit.Type] {
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
			score = n * s.NeighbourScore(&s.hexes.Arr144, unit2.XY, unit2.Side) / 8 * (255 - unit2.Fatigue) / 256 * unit2.Morale / 128
		} else if (nxy == unit.XY) || !s.ContainsVisibleUnit(nxy) {
			if tt := s.terrainTypes.terrainTypeAt(nxy); tt < 7 {
				var v int
				if unit.MenCount > unit.TankCount {
					v = s.scenarioData.TerrainMenAttack[tt]
				} else {
					v = s.scenarioData.TerrainTankAttack[tt]
				}
				// temporarily hide the unit while we compute sth
				s.units[unit.Side][unit.Index].IsInGame = false
				score = temp2 - s.NeighbourScore(&s.hexes.Arr48, nxy, unit.Side)*2 + v
				// unhide the unit
				s.units[unit.Side][unit.Index].IsInGame = true
			}
		}
		if i < 12 {
			score *= 2
		}
		if city, ok := s.terrain.FindCityAt(nxy); ok {
			if city.Owner != unit.Side {
				if s.units.IsUnitOfSideAt(nxy, 1-unit.Side) {
					score -= city.VictoryPoints
				} else {
					score = -city.VictoryPoints
				}
			}
		}
		if score <= bestScore {
			bestScore = score
			bestObjective = nxy
		}
	}
	return bestObjective, bestScore
}

func (s *AI) bestDefenceObjective(unit Unit) (UnitCoords, int) {
	// temperarily hide the unit while we compute sth
	s.units[unit.Side][unit.Index].IsInGame = false
	score := -17536 // 48000
	var bestI int
	// Score for i==6 (zero offset - the unit's position).
	var v_6 int
	for i := 0; i <= 6; i++ {
		nxy := IthNeighbour(unit.XY, i)
		if !s.areUnitCoordsValid(nxy) {
			continue
		}
		v := -128
		if (nxy == unit.XY) || !s.ContainsVisibleUnit(nxy) {
			if tt := s.terrainTypes.terrainTypeAt(nxy); tt < 7 {
				r := s.scenarioData.TerrainMenDefence[tt]
				v = r
				if s.game != Conflict {
					v += s.NeighbourScore(&s.hexes.Arr0, nxy, unit.Side) * 2
				}
				if city, ok := s.terrain.FindCityAt(nxy); ok {
					if !s.units.IsUnitOfSideAt(nxy, unit.Side) {
						v += city.VictoryPoints
					}
				}
				if s.scenarioData.UnitScores[unit.Type] > 7 ||
					unit.Fatigue+unit.General.Data2High*4 > 96 {
					v = r + s.NeighbourScore(&s.hexes.Arr96, nxy, unit.Side)*2
				}
			}
		}
		if v >= score {
			score = v
			bestI = i
		}
		if i == 6 {
			v_6 = v
		}
	}
	// unhide unit
	s.units[unit.Side][unit.Index].IsInGame = true
	v := s.scenarioData.FormationMenDefence[unit.Formation] - 8
	if s.commanderFlags.PlayerControlled[unit.Side] {
		v *= 2
	}
	if v+v_6 > score {
		bestI = 6
	}
	return IthNeighbour(unit.XY, bestI), score
}

func (s *AI) areUnitCoordsValid(xy UnitCoords) bool {
	return s.terrainTypes.AreCoordsValid(xy.ToMapCoords())
}

// score of the location based on occupancy and terrain of neighbouring tiles.
// arr is one of arrays in Hexes
func (s *AI) NeighbourScore(arr *[6][8]int, xy UnitCoords, side int) int {
	// Count of neighbour tiles with given type
	var neighbourTypeCount [6]int
	for i := 0; i < 6; i++ {
		nxy := IthNeighbour(xy, i)
		var neighbourType int
		if s.units.IsUnitOfSideAt(nxy, 1-side) {
			neighbourType = 2
		} else if s.units.IsUnitOfSideAt(nxy, side) || !s.areUnitCoordsValid(nxy) || s.terrainTypes.terrainOrUnitTypeAt(nxy) >= 7 {
			neighbourType = 1
		} else {
			// neighbours to the left of nx,ny
			n0xy := IthNeighbour(xy, (i+5)%6)
			// neighbours to the right of nx,ny
			n1xy := IthNeighbour(xy, (i+1)%6)
			neighbourIsEnemy := s.units.IsUnitOfSideAt(n0xy, 1-side) || s.units.IsUnitOfSideAt(n1xy, 1-side)
			neighbourIsFriend := s.units.IsUnitOfSideAt(n0xy, side) || s.units.IsUnitOfSideAt(n1xy, side)
			if neighbourIsEnemy && neighbourIsFriend {
				neighbourType = 5
			} else if neighbourIsEnemy {
				neighbourType = 4
			} else if neighbourIsFriend {
				neighbourType = 3
			}
		}
		neighbourTypeCount[neighbourType]++
	}
	neighbourScore := 0
	for i := 0; i < 6; i++ {
		neighbourScore += arr[i][neighbourTypeCount[i]]
	}
	return neighbourScore
}

func (s *AI) ContainsVisibleUnit(xy UnitCoords) bool {
	if unit, ok := s.units.FindUnitAt(xy); !ok {
		return false
	} else {
		return s.commanderFlags.PlayerCanSeeUnits[unit.Side] || unit.IsVisible()
	}
}

func (s *AI) performUnitMovement(unit *Unit, message *MessageFromUnit, arg1 *int, weather int, sync *MessageSync) (sxy UnitCoords, quit bool) {
	// l22:
	for unitMoveBudget := 25; unitMoveBudget > 0; {
		if unit.Objective.X == 0 {
			return
		}
		distance := unit.Function15_distanceToObjective()
		attackRange := s.scenarioData.AttackRange[unit.Type] * 2
		if distance > 0 && distance <= attackRange && unit.Order == Attack {
			sxy = unit.Objective
			unit.LongRangeAttack = true
			*arg1 = 7
			return // goto l2
		}
		var moveSpeed int
		for mvAdd := 0; mvAdd <= 1; mvAdd++ { // l5:
			if unit.Objective == unit.XY {
				unit.Objective.X = 0
				unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
				return // goto l2
			}
			unit.TargetFormation = s.scenarioData.function10(unit.Order, 0)
			if !s.commanderFlags.PlayerControlled[unit.Side] || unit.HasLocalCommand {
				// If it's next to its objective to defend and it's in contact with enemy
				if distance == 1 && unit.Order == Defend && unit.InContactWithEnemy {
					unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
				}
			}
			sxy, moveSpeed = s.findBestMoveFromTowards(unit.XY, unit.Objective, unit.Type, mvAdd)
			if s.scenarioData.Data32_64[unit.Type] { // in CiV (some scenarios) artillery or mortars
				if s.game != Conflict || unit.Formation == 0 {
					sxy = unit.Objective
					tt := s.terrainTypes.terrainOrUnitTypeAt(sxy)
					moveSpeed = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit.Type]
					*arg1 = tt // shouldn't have any impact
					mvAdd = 1
				} else if s.scenarioData.UnitMask5[unit.Type] {
					// Conflict && unit.Formation != 0
					return // goto l2
				}
			}
			if s.units.IsUnitOfSideAt(sxy, unit.Side) {
				moveSpeed = 0
			}
			if s.units.IsUnitOfSideAt(sxy, 1-unit.Side) {
				moveSpeed = -1
			}
			if moveSpeed >= 1 || (unit.Order == Attack && moveSpeed == -1) ||
				Abs(unit.Objective.X-unit.XY.X)+Abs(unit.Objective.Y-unit.XY.Y) <= 2 {
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
		if !s.scenarioData.UnitMask2[unit.Type] {
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
			if !sync.SendUpdate(UnitMove{*unit, unit.XY.ToMapCoords(), sxy.ToMapCoords()}) {
				quit = true
				return
			}
		}
		unit.XY = sxy
		s.function29_showUnit(*unit)
		if unit.Function15_distanceToObjective() == 0 {
			unit.Objective.X = 0
			unit.TargetFormation = s.scenarioData.function10(unit.Order, 1)
			if (unit.Order == Defend || unit.Order == Move) && !unit.HasLocalCommand {
				*message = WeHaveReachedOurObjective{*unit}
			}
		}
		unit.Fatigue = Clamp(unit.Fatigue+s.scenarioData.Data173, 0, 255)
		if city, captured := s.function16(*unit); captured {
			*message = WeHaveCaptured{*unit, *city}
			return // goto l2
		}
		if unitMoveBudget > 0 {
			if s.units.NeighbourUnitCount(unit.XY, 1-unit.Side) > 0 {
				unit.InContactWithEnemy = true
				unit.State4 = true // |= 17
			} else {
				unit.InContactWithEnemy = false // &= 254
			}
			s.function29_showUnit(*unit)
			// goto l22
		}
	}
	// l2:
	return
}

func (s *AI) performAttack(unit *Unit, sxy UnitCoords, weather int, message *MessageFromUnit, sync *MessageSync) (shouldQuit bool) {
	if !unit.LongRangeAttack {
		s.terrainTypes.hideUnit(*unit)
		if !sync.SendUpdate(UnitMove{*unit, unit.XY.ToMapCoords(), sxy.ToMapCoords()}) {
			shouldQuit = true
			return
		}
		s.terrainTypes.showUnit(*unit)
		if s.game == Conflict {
			unit.InContactWithEnemy = true
			unit.SeenByEnemy = true // |= 65
		}
		// function14
	} else {
		susceptibleToWeather := s.scenarioData.Data32_8[unit.Type]
		if susceptibleToWeather && weather > 3 {
			// [53767] = 0
			return // goto end
		}
		// function27
	}

	if s.game != Conflict {
		unit.InContactWithEnemy = true
		unit.SeenByEnemy = true // |= 65
	}

	unit2, ok := s.units.FindUnitOfSideAt(sxy, 1-unit.Side)
	if !ok {
		panic("")
	}
	*message = WeAreAttacking{*unit, unit2, 0 /* placeholder value */, s.scenarioData.Formations}
	var attackerScore int
	{
		tt := s.terrainTypes.terrainTypeAt(unit.XY)
		var menCoeff int
		if !unit.LongRangeAttack {
			menCoeff = s.scenarioData.TerrainMenAttack[tt] * s.scenarioData.FormationMenAttack[unit.Formation] * unit.MenCount / 32
		}
		tankCoeff := s.scenarioData.TerrainTankAttack[tt] * s.scenarioData.FormationTankAttack[unit.Formation] * s.scenarioData.Data16High[unit.Type] / 2 * unit.TankCount / 64
		susceptibleToWeather := s.scenarioData.Data32_8[unit.Type]
		if s.game == Conflict {
			susceptibleToWeather = s.scenarioData.Data32_32[unit.Type]
		}
		if unit.LongRangeAttack && susceptibleToWeather {
			// long range unit
			if weather > 3 {
				return // goto l3
			}
			tankCoeff = tankCoeff * (4 - weather) / 4
		}
		attackerScore = (menCoeff + tankCoeff) * unit.Morale / 256 * (255 - unit.Fatigue) / 128
		attackerScore = attackerScore * unit.General.Attack / 16
		attackerScore = attackerScore * s.NeighbourScore(&s.hexes.Arr144, unit.XY, unit.Side) / 8
		attackerScore++
	}

	var defenderScore int
	{
		if s.scenarioData.UnitScores[unit2.Type] > 7 {
			unit.State2 = true // |= 4
		}
		tt2 := s.terrainTypes.terrainTypeAt(unit2.XY)
		menCoeff := s.scenarioData.TerrainMenDefence[tt2] * s.scenarioData.FormationMenDefence[unit2.Formation] * unit2.MenCount / 32
		tankCoeff := s.scenarioData.TerrainTankAttack[tt2] * s.scenarioData.FormationTankDefence[unit2.Formation] * s.scenarioData.Data16Low[unit2.Type] / 2 * unit2.TankCount / 64
		defenderScore = (menCoeff + tankCoeff) * unit2.Morale / 256 * (240 - unit2.Fatigue/2) / 128
		defenderScore = defenderScore * unit2.General.Defence / 16
		if unit2.SupplyLevel == 0 {
			defenderScore = defenderScore * s.scenarioData.Data167 / 8
		}
		defenderScore = defenderScore * s.NeighbourScore(&s.hexes.Arr144, unit2.XY, 1-unit.Side) / 8
		defenderScore++
	}

	arg1 := defenderScore * 16 / attackerScore
	if !s.scenarioData.UnitMask2[unit.Type] {
		arg1 += weather
	}
	arg1 = Clamp(arg1, 0, 63)
	if !unit.LongRangeAttack || !s.scenarioData.Data32_128[unit.Type] {
		menLost := Clamp((Rand(unit.MenCount*arg1, s.rand)+255)/512, 0, unit.MenCount)
		s.score.MenLost[unit.Side] += menLost
		unit.MenCount -= menLost
		tanksLost := Clamp((Rand(unit.TankCount*arg1, s.rand)+255)/512, 0, unit.TankCount)
		s.score.TanksLost[unit.Side] += tanksLost
		unit.TankCount -= tanksLost
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
	sync.SendUpdate(UnitAttack{sxy, arg1})

	menLost2 := Clamp((Rand(unit2.MenCount*arg1, s.rand)+500)/512, 0, unit2.MenCount)
	s.score.MenLost[1-unit.Side] += menLost2
	unit2.MenCount -= menLost2
	tanksLost2 := Clamp((Rand(unit2.TankCount*arg1, s.rand)+255)/512, 0, unit2.TankCount)
	s.score.TanksLost[1-unit.Side] += tanksLost2
	unit2.TankCount -= tanksLost2
	unit2.SupplyLevel = Clamp(unit2.SupplyLevel-s.scenarioData.Data163, 0, 255)
	if s.scenarioData.UnitCanMove[unit2.Type] &&
		((s.game != Conflict && !unit.LongRangeAttack) ||
			(s.game == Conflict && !s.scenarioData.UnitMask1[unit2.Type])) &&
		arg1-s.scenarioData.Data0Low[unit2.Type]*2+unit2.Fatigue/4 > 36 {
		unit2.Morale = Abs(unit2.Morale - 1)
		oldXY := unit2.XY
		bestXY := unit2.XY
		s.terrainTypes.hideUnit(unit2)
		if unit2.Fatigue > 128 {
			unit2SupplyUnit := s.units[unit2.Side][unit2.SupplyUnit]
			if unit2SupplyUnit.IsInGame {
				unit2.Morale = Abs(unit2.Morale - s.units.NeighbourUnitCount(unit2.XY, unit.Side)*4)
				unit2.XY = unit2SupplyUnit.XY
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
			nxy := IthNeighbour(unit2.XY, i)
			if !s.areUnitCoordsValid(nxy) || s.units.IsUnitAt(nxy) || s.terrain.IsCityAt(nxy) {
				continue
			}
			tt := s.terrainTypes.terrainOrUnitTypeAt(nxy)
			if s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit2.Type] == 0 {
				continue
			}
			r := s.scenarioData.TerrainMenDefence[tt] +
				s.NeighbourScore(&s.hexes.Arr96, nxy, 1-unit.Side)*4
			if r > 11 && r >= bestDefence {
				bestDefence = r
				bestXY = nxy
			}
		}
		unit2.XY = bestXY // moved this up comparing to the original code
		if _, ok := (*message).(WeHaveBeenOverrun); !ok {
			if s.game != Conflict {
				s.terrainTypes.showUnit(unit2)
				unit.Objective = unit2.XY
			} else {
				if s.commanderFlags.PlayerCanSeeUnits[1-unit.Side] {
					s.terrainTypes.showUnit(unit2)
				}
				unit2.InContactWithEnemy = false
				unit2.SeenByEnemy = false // &= 190
			}
		}
		if bestXY != oldXY {
			// unit2 is retreating, unit is chasing (and maybe capturing a city)
			if _, ok := (*message).(WeHaveBeenOverrun); !ok {
				*message = WeAreRetreating{unit2}
			}
			tt := s.terrainTypes.terrainOrUnitTypeAt(oldXY)
			if arg1 > 60 && (s.game != Conflict || !unit.LongRangeAttack) &&
				s.NeighbourScore(&s.hexes.Arr96, oldXY, unit.Side) > -4 &&
				s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[tt][unit.Type] > 0 {
				s.terrainTypes.hideUnit(*unit)
				unit.XY = oldXY
				s.terrainTypes.showUnit(*unit)
				if city, captured := s.function16(*unit); captured {
					*message = WeHaveCaptured{*unit, *city}
				}
			}
		} else {
			*message = nil
		}
		unit2.Formation = s.scenarioData.Data176[1][0]
		unit2.Order = OrderType((s.scenarioData.Data176[1][0] + 1) % 4)
		unit2.HasLocalCommand = true // |= 32
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
	return
}

func (s *AI) ResupplyUnit(unit Unit, supplyLevels *[2]int, sync *MessageSync) Unit {
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
		supplyXY := supplyUnit.XY
		if unitVisible {
			s.terrainTypes.showUnit(supplyUnit)
		}
		supplyTransportBudget := s.scenarioData.MaxSupplyTransportCost
		if unit.Type == s.scenarioData.MinSupplyType&15 {
			supplyTransportBudget *= 2
		}
		for supplyTransportBudget > 0 {
			dx, dy := unit.XY.X-supplyXY.X, unit.XY.Y-supplyXY.Y
			if Abs(dx)+Abs(dy) < 3 {
				supplyLevel := supplyLevels[unit.Side]
				if supplyLevel > 0 {
					maxResupply := Clamp(
						(supplyLevel-unit.SupplyLevel*2)/16,
						0,
						s.scenarioData.MaxResupplyAmount)
					unitResupply := s.scenarioData.UnitResupplyPerType[unit.Type]
					unitResupply = Clamp(unitResupply, 0, maxResupply)
					unit.SupplyLevel += unitResupply
					supplyLevels[unit.Side] -= unitResupply
					unit.HasSupplyLine = true
				} else {
					// not sure if it's needed...
					supplyLevels[unit.Side] = 0
				}
				s.terrainTypes.hideUnit(supplyUnit)
				break outerLoop
			} else {
				var speed int
				var xy UnitCoords
				// TODO: why changing variant < 2 to variant < 1 has no effect (cost never 0? at least in dday?)
				for variant := 0; variant < 2; variant++ {
					xy, speed = s.findBestMoveFromTowards(supplyXY, unit.XY, s.scenarioData.MinSupplyType, variant)
					if speed != 0 {
						break
					}
				}
				if unitVisible {
					sync.SendUpdate(SupplyTruckMove{supplyXY.ToMapCoords(), xy.ToMapCoords()})
					//  function13(x, y) (show truck icon at x, y)
				}
				supplyXY = xy
				if s.units.IsUnitOfSideAt(supplyXY, 1-unit.Side) {
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
		if supplyUnit.XY.X != 0 {
			unit.Objective = supplyUnit.XY
		}
	}
	s.terrainTypes.hideUnit(unit)
	return unit
}

// function6
// Finds best position to move to if you want to move from unitX0,unitY0 to unitX1, unitY1 with unit
// of type unitType. If variant == 0 consider only neighbour fields directly towards the goal,
// if variant == 1 look at neighbour two fields "more to the side".
// Also return the speed the unit can move to the returned position.
func (s *AI) findBestMoveFromTowards(unitXY0, unitXY1 UnitCoords, unitType, variant int) (UnitCoords, int) {
	candXY1 := FirstNeighbourFromTowards(unitXY0, unitXY1, 2*variant)
	var speed1 int
	if !s.areUnitCoordsValid(candXY1) {
		candXY1 = unitXY0
	} else {
		terrainType1 := s.terrainTypes.terrainOrUnitTypeAt(candXY1)
		speed1 = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[terrainType1][unitType]
	}

	candXY2 := FirstNeighbourFromTowards(unitXY0, unitXY1, 2*variant+1)
	var speed2 int
	if !s.areUnitCoordsValid(candXY2) {
		candXY2 = unitXY0
	} else {
		terrainType2 := s.terrainTypes.terrainOrUnitTypeAt(candXY2)
		speed2 = s.scenarioData.MoveSpeedPerTerrainTypeAndUnit[terrainType2][unitType]
	}

	if speed2 > speed1-Rand(2, s.rand) {
		return candXY2, speed2
	}
	return candXY1, speed1
}

func (s *AI) function29_showUnit(unit Unit) {
	if unit.InContactWithEnemy || unit.SeenByEnemy /* &65 != 0 */ ||
		s.commanderFlags.PlayerCanSeeUnits[unit.Side] {
		s.terrainTypes.showUnit(unit)
	}
}

// Has unit captured a city
func (s *AI) function16(unit Unit) (*City, bool) {
	if city, ok := s.terrain.FindCityAt(unit.XY); ok {
		if city.Owner != unit.Side {
			// msg = 5
			city.Owner = unit.Side
			s.score.CitiesHeld[unit.Side] += city.VictoryPoints
			s.score.CitiesHeld[1-unit.Side] -= city.VictoryPoints
			s.score.CriticalLocationsCaptured[unit.Side] += city.VictoryPoints & 1
			return city, true
		}
	}
	return nil, false
}
