package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"math"
	"sort"
)

// Representation of data parsed from GENERIC.DTA file.
type Generic struct {
	// Terrain colors on the overview map.
	Data60       [4]int
	TerrainTypes [64]int // Bytes [64:128]
	// First two indices are positions on a 2x2 square, the third one is one of 9 neighbouring
	// squares on 3x3 square tiling.
	Data214 [2][2][9]int
}

// directionIndex assigns number 0..11 to a direction (dx, dy).
// Numbers are assigned consecutively around the origin.
// Odd numbers are assigned to directions exactly diagonal or horizontal.
// Even numbers are assigned to ranges of directions between the odd-numbered directions.
func directionIndex(dx, dy int) int {
	if dy < 0 {
		if dx < dy {
			return 0
		} else if dx == dy {
			return 1
		} else if dx < -dy {
			return 2
		} else if dx == -dy {
			return 3
		} else { // dx > -dy
			return 4
		}
	} else if dy > 0 {
		if dx < -dy {
			return 10
		} else if dx == -dy {
			return 9
		} else if dx < dy {
			return 8
		} else if dx == dy {
			return 7
		} else { // dx > dy
			return 6
		}
	} else { // dy == 0
		if dx > 0 {
			return 5
		} else if dx < 0 {
			return 11
		} else { // dx == 0
			return 0
		}
	}
}

// First neighbouring tile met when going from x0,y0 towards x1,y1.
// If variant is 0 or 1, pick one of the most direct directions.
// If variant is 2 or 3, pick one of the less direct directions.
func FirstNeighbourFromTowards(xy0, xy1 UnitCoords, variant int) UnitCoords {
	dx, dy := xy1.X-xy0.X, xy1.Y-xy0.Y
	direction := directionIndex(dx, dy)
	var neighbourInDirection int
	if variant < 2 {
		neighbourInDirection = ((direction + 3 + variant) % 12) / 2
	} else if variant == 2 {
		neighbourInDirection = ((direction + 1) % 12) / 2
	} else { // variant == 3
		neighbourInDirection = ((direction + 6) % 12) / 2
	}
	return IthNeighbour(xy0, neighbourInDirection)
}

func IthNeighbour(xy UnitCoords, i int) UnitCoords {
	dx, dy := hexNeighbourOffset(i)
	return UnitCoords{xy.X + dx, xy.Y + dy}
}

type offset struct {
	dx, dy int
}

// Offsets on square tiled map.
var squareTilingOffsets = generateSquareTilingOffsets()

func generateSquareTilingOffsets() []offset {
	offsets := make([]offset, 0, 25)
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			offsets = append(offsets, offset{dx, dy})
		}
	}
	// Sort offsets first by distance from origin, than dy, than dx.
	compareOffsets := func(i, j int) bool {
		distI := offsets[i].dx*offsets[i].dx + offsets[i].dy*offsets[i].dy
		distJ := offsets[j].dx*offsets[j].dx + offsets[j].dy*offsets[j].dy
		if distI != distJ {
			return distI < distJ
		}
		if offsets[i].dy != offsets[j].dy {
			return offsets[i].dy < offsets[j].dy
		}
		return offsets[i].dx < offsets[j].dx
	}
	sort.Slice(offsets, compareOffsets)
	return offsets
}

func squareTilingNeighbour(i int) (int, int) {
	offset := squareTilingOffsets[i]
	return offset.dx, offset.dy
}
func SmallMapOffsets(i int) (int, int) {
	return squareTilingNeighbour(i)
}
func TinyMapOffsets(i int) (int, int) {
	if i < 9 {
		return squareTilingNeighbour(i)
	}
	panic(fmt.Errorf("Invalid tiny map offset index %d", i))
}

type offsetWithAngle struct {
	dx, dy int
	angle  float64
}

func HalfTileOffsetDistance(dx, dy int) int {
	absDx, absDy := Abs(dx), Abs(dy)
	if absDy > absDx/2 {
		return absDy
	}
	return (absDx + absDy + 1) / 2
}
func generateHalfOffsetSquareTilingOffsets() []offsetWithAngle {
	offsets := make([]offsetWithAngle, 0, 19)
	for dx := -4; dx <= 4; dx++ {
		for dy := -2; dy <= 4; dy++ {
			// Pick only a correct offset on half tile offset tiling.
			if Abs(dy)%2 != Abs(dx)%2 {
				continue
			}
			distance := HalfTileOffsetDistance(dx, dy)
			if distance > 2 {
				continue
			}
			var angle float64
			// Slightly different order of hexes further away and those closer
			// to keep it faithful to the original ordering.
			if distance == 2 {
				angle = math.Atan2(-float64(dx), float64(dy))
			} else if distance == 1 {
				angle = math.Atan2(float64(dx), -float64(dy))
			}
			offsets = append(offsets, offsetWithAngle{dx, dy, angle})
		}
	}
	// First put offsets further away from the origin, sort them according to the angle.
	compareOffsets := func(i, j int) bool {
		distI := HalfTileOffsetDistance(offsets[i].dx, offsets[i].dy)
		distJ := HalfTileOffsetDistance(offsets[j].dx, offsets[j].dy)
		if distI != distJ {
			return distI > distJ
		}
		return offsets[i].angle < offsets[j].angle
	}
	sort.Slice(offsets, compareOffsets)
	return offsets
}

var halfTileOffsets = generateHalfOffsetSquareTilingOffsets()

func halfTileOffsetNeighbour(i int) (int, int) {
	offset := halfTileOffsets[i]
	return offset.dx, offset.dy
}
func LongRangeHexNeighbourOffset(i int) (int, int) {
	return halfTileOffsetNeighbour(i)
}
func hexNeighbourOffset(i int) (int, int) {
	if i < 7 {
		return halfTileOffsetNeighbour(i + 12)
	}
	panic(fmt.Errorf("Invalid hex neighbour index %d", i))
}

func ReadGeneric(fsys fs.FS) (*Generic, error) {
	fileData, err := fs.ReadFile(fsys, "GENERIC.DTA")
	if err != nil {
		return nil, fmt.Errorf("Cannot read GENERIC.DTA file (%v)", err)
	}
	return ParseGeneric(bytes.NewReader(fileData))
}

func ParseGeneric(reader io.Reader) (*Generic, error) {
	var data [250]byte
	_, err := io.ReadFull(reader, data[:])
	if err != nil {
		return nil, err
	}

	generic := &Generic{}

	for i, value := range data[60:64] {
		generic.Data60[i] = int(value)
	}

	for i, terrain := range data[64:128] {
		generic.TerrainTypes[i] = int(terrain)
	}

	for i, v := range data[214:250] {
		generic.Data214[i/18][(i/9)%2][i%9] = int(v)
	}
	return generic, nil
}
