package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
)

// Representation of data parsed from GENERIC.DTA file.
type Generic struct {
	DirectionToNeighbourIndex map[int]int // Data[0:19]
	Neighbours                [4][12]int  // Data[20:44], Data[128:152]
	// Offsets on a 2-byte square map 4x4.
	// First 0 offset to the origin field itself,
	// then to its 4 neighbours in cardinal directions,
	// then to its 4 neighbours in diagonal direction.
	tinyMapOffsets [9]int // Bytes [44:52]
	MapOffsets     [7]int // Bytes [53:60]
	Data60         [4]int
	TerrainTypes   [64]int // Bytes [64:128]
	Dx152          [19]int //
	Dy153          [19]int // Bytes [152:190] (Dx, Dy interleaved, overlaps following arrays)
	Dx             [7]int  //
	Dy             [7]int  // Bytes [176:190] (Dx,Dy interleaved, overlaps following array)
	// Offsets on a square map 16x16.
	// First 0 offset to the origin field itself,
	// then to its 4 neighbours in cardinal directions,
	// then to 4 neighbours in diagonal direction,
	// then offsets to fields with distance 2 from the origin.
	smallMapOffsets [25]int // Bytes [189:214]
	Data214         [2][2][9]int
}

func CoordsToMapAddress(x, y int) int {
	return y*64 + x/2 - y/2
}

func signInt(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

// First neighbouring tile met when going from x0,y0 towards x1,y1.
// If variant is 0 or 1, pick one of the most direct directions.
// If variant is 2 or 3, pick one of the less direct directions.
func (g Generic) FirstNeighbourFromTowards(xy0, xy1 UnitCoords, variant int) UnitCoords {
	dx, dy := xy1.X-xy0.X, xy1.Y-xy0.Y
	direction := 5*signInt(dy) + 3*signInt(dx-dy) + signInt(dx+dy)
	neighbourIndex, ok := g.DirectionToNeighbourIndex[direction]
	if !ok {
		panic(fmt.Errorf("No neighbour index for direction %d", direction))
	}
	return g.IthNeighbour(xy0, g.Neighbours[variant][neighbourIndex])
}

func (g Generic) IthNeighbour(xy UnitCoords, i int) UnitCoords {
	dx, dy := g.Dx[i], g.Dy[i]
	return UnitCoords{xy.X + dx, xy.Y + dy}
}

func (g Generic) SmallMapOffsets(i int) (dx int, dy int) {
	offsetNum := g.smallMapOffsets[i]
	if offsetNum >= 0 { /* dy >= 0 */
		dy = (offsetNum + 13) / 16
	} else { /* dy <= 0 */
		dy = (offsetNum - 13) / 16
	}
	if (offsetNum+32)%16 < 8 { /* dx >= 0 */
		dx = (offsetNum + 32) % 16
	} else { /* dx <= 0 */
		dx = (offsetNum+32)%16 - 16
	}
	return
}
func (g Generic) TinyMapOffsets(i int) (dx int, dy int) {
	offsetNum := g.tinyMapOffsets[i] / 2
	if offsetNum >= 0 { /* dy >= 0 */
		dy = (offsetNum + 1) / 4
	} else { /* dy <= 0 */
		dy = (offsetNum - 1) / 4
	}
	if (offsetNum+8)%4 < 2 { /* dx >= 0 */
		dx = (offsetNum + 8) % 4
	} else { /* dx <= 0 */
		dx = (offsetNum+8)%4 - 4
	}
	return
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
	generic.DirectionToNeighbourIndex = make(map[int]int)
	for i, offset := range data[0:19] {
		generic.DirectionToNeighbourIndex[i-9] = int(offset)
	}

	for i, neighbour := range data[20:44] {
		generic.Neighbours[i%2][i/2] = int(neighbour)
	}

	for i, d := range data[44:53] {
		generic.tinyMapOffsets[i] = int(int8(d))
	}

	for i, offset := range data[53:60] {
		generic.MapOffsets[i] = int(int8(offset))
	}

	for i, value := range data[60:64] {
		generic.Data60[i] = int(value)
	}

	for i, terrain := range data[64:128] {
		generic.TerrainTypes[i] = int(terrain)
	}

	for i, neighbour := range data[128:152] {
		generic.Neighbours[2+(i%2)][i/2] = int(neighbour)
	}

	for i, dxdy := range data[152:190] {
		if i%2 == 0 {
			generic.Dx152[i/2] = int(int8(dxdy))
		} else {
			generic.Dy153[i/2] = int(int8(dxdy))
		}
	}
	for i, dxdy := range data[176:190] {
		if i%2 == 0 {
			generic.Dx[i/2] = int(int8(dxdy))
		} else {
			generic.Dy[i/2] = int(int8(dxdy))
		}
	}

	for i, v := range data[189:214] {
		generic.smallMapOffsets[i] = int(int8(v))
	}

	for i, v := range data[214:250] {
		generic.Data214[i/18][(i/9)%2][i%9] = int(v)
	}
	return generic, nil
}
