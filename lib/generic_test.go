package lib

import "testing"

func TestTinyAndSmallMapOffsetsAreSane(t *testing.T) {
	tinyOffsetsMap := make(map[int]struct{})
	for i := 0; i < 9; i++ {
		dx, dy := TinyMapOffsets(i)
		if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
			t.Errorf("Invalid tiny map offset for %d %d,%d", i, dx, dy)
		}
		num := (dx+1)*3 + dy + 1
		tinyOffsetsMap[num] = struct{}{}
	}
	if len(tinyOffsetsMap) != 9 {
		t.Error("Duplicate tiny offsets")
	}

	smallOffsetsMap := make(map[int]struct{})
	for i := 0; i < 9; i++ {
		dx, dy := SmallMapOffsets(i)
		if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
			t.Errorf("Invalid small map offset for %d %d,%d", i, dx, dy)
		}
		num := (dx+2)*5 + dy + 2
		smallOffsetsMap[num] = struct{}{}
	}
	for i := 9; i < 25; i++ {
		dx, dy := SmallMapOffsets(i)
		if dx < -2 || dx > 2 || dy < -2 || dy > 2 {
			t.Errorf("Invalid small map offset for %d %d,%d", i, dx, dy)
		}
		num := (dx+2)*5 + dy + 2
		smallOffsetsMap[num] = struct{}{}
	}
	if len(smallOffsetsMap) != 25 {
		t.Error("Duplicate small offsets")
	}
}

func TestDirectionTowardsNeightbour(t *testing.T) {
	initialCoords := UnitCoords{2, 4} // "random" initial coords
	for i := 0; i < 6; i++ {
		neighbourCoords := IthNeighbour(initialCoords, i)
		// variants 0 and 1 should go directly towards the neighbour
		neighbourToGoTo := FirstNeighbourFromTowards(initialCoords, neighbourCoords, 0)
		if neighbourCoords.X != neighbourToGoTo.X || neighbourCoords.Y != neighbourToGoTo.Y {
			t.Errorf("Expecting neighbour %v got %v for index %d, variant 0", neighbourCoords, neighbourToGoTo, i)
		}
		neighbourToGoTo = FirstNeighbourFromTowards(initialCoords, neighbourCoords, 1)
		if neighbourCoords.X != neighbourToGoTo.X || neighbourCoords.Y != neighbourToGoTo.Y {
			t.Errorf("Expecting neighbour %v got %v for index %d, variant 1", neighbourCoords, neighbourToGoTo, i)
		}
		// variants 1, 2 should go to a adjacent hex to the neighbour
		neighbourToGoTo = FirstNeighbourFromTowards(initialCoords, neighbourCoords, 2)
		dx, dy := neighbourToGoTo.X-initialCoords.X, neighbourToGoTo.Y-initialCoords.Y
		if HalfTileOffsetDistance(dx, dy) != 1 {
			t.Errorf("Expecting neighbour at distance 1, for index %d variant 2", i)
		}
		neighbourToGoTo = FirstNeighbourFromTowards(initialCoords, neighbourCoords, 3)
		dx, dy = neighbourToGoTo.X-initialCoords.X, neighbourToGoTo.Y-initialCoords.Y
		if HalfTileOffsetDistance(dx, dy) != 1 {
			t.Errorf("Expecting neighbour at distance 1, for index %d variant 3", i)
		}
	}
}
