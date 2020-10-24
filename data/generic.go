package data

import "fmt"
import "io"
import "os"
import "path"

// Representation of data parsed from GENERIC.DTA file.
type Generic struct {
	DirectionToNeighbourIndex map[int]int
	Neighbours                [4][12]int
	// Offsets on a 2-byte square map 4x4.
	// First 0 offset to the origin field itself,
	// then to its 4 neighbours in cardinal directions,
	// then to its 4 neighbours in diagonal direction.
	tinyMapOffsets [9]int // Bytes [44:52]
	MapOffsets     [6]int
	TerrainTypes   [64]int // Bytes [64:128]
	Dx152          [19]int //
	Dy153          [19]int // Bytes [152:190] (Dx, Dy interleaved, overlaps following arrays)
	Dx             [6]int  //
	Dy             [6]int  // Bytes [176:188] (Dx,Dy interleaved)
	// Offsets on a square map 16x16.
	// First 0 offset to the origin field itself,
	// then to its 4 neighbours in cardinal directions,
	// then to 4 neighbours in diagonal direction,
	// then offsets to fields with distance 2 from the origin.
	smallMapOffsets [25]int // Bytes [189:214]
	Data214         [36]int
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

func (g Generic) DxDyToNeighbour(dx, dy, variant int) int {
	direction := 5*signInt(dy) + 3*signInt(dx-dy) + signInt(dx+dy)
	neighbourIndex, ok := g.DirectionToNeighbourIndex[direction]
	if !ok {
		panic(fmt.Errorf("No neighbour index for direction %d", direction))
	}
	return g.Neighbours[variant][neighbourIndex]
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

func ReadGeneric(dirname string) (Generic, error) {
	filename := path.Join(dirname, "GENERIC.DTA")
	file, err := os.Open(filename)
	if err != nil {
		return Generic{}, fmt.Errorf("Cannot open generic file %s, %v", filename, err)
	}
	defer file.Close()
	return ParseGeneric(file)
}

func ParseGeneric(reader io.Reader) (Generic, error) {
	var data [250]byte
	_, err := io.ReadFull(reader, data[:])
	if err != nil {
		return Generic{}, err
	}

	var generic Generic
	generic.DirectionToNeighbourIndex = make(map[int]int)
	for i, offset := range data[0:19] {
		generic.DirectionToNeighbourIndex[i-9] = int(offset)
	}

	for i, neighbour := range data[20:44] {
		generic.Neighbours[i%2][i/2] = int(neighbour)
	}

	for i, d := range data[44:53] {
		generic.tinyMapOffsets[i] = int(d)
	}

	for i, offset := range data[53:59] {
		generic.MapOffsets[i] = int(int8(offset))
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
	for i, dxdy := range data[176:188] {
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
		generic.Data214[i] = int(v)
	}
	return generic, nil
}
