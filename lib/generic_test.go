package lib

import "testing"

func TestTinyAndSmallMapOffsetsAreSane(t *testing.T) {
	gameData, _, err := readTestData("crusade.atr", 0)
	if err != nil {
		t.Fatal("Error reading game data,", err)
	}
	generic := gameData.Generic

	tinyOffsetsMap := make(map[int]struct{})
	for i := 0; i < 9; i++ {
		dx, dy := generic.TinyMapOffsets(i)
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
		dx, dy := generic.SmallMapOffsets(i)
		if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
			t.Errorf("Invalid small map offset for %d %d,%d", i, dx, dy)
		}
		num := (dx+2)*5 + dy + 2
		smallOffsetsMap[num] = struct{}{}
	}
	for i := 9; i < 25; i++ {
		dx, dy := generic.SmallMapOffsets(i)
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
