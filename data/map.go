package data

import "bufio"
import "bytes"
import "fmt"
import "image"
import "image/color"
import "io"
import "os"
import "path"

// A representation of a hex map parsed from CRUSADE.MAP file.
type Map struct {
	Width, Height int
	terrain       []byte
	mapImage      *image.NRGBA
}

func GetPalette(n int, palette *[8]byte) []color.Color {
	pal := make([]color.Color, 2)
	// just guessing here
	pal[0] = &RGBPalette[palette[2]]
	switch n {
	case 0:
		pal[1] = &RGBPalette[palette[3]] // or 7
	case 1:
		pal[1] = &RGBPalette[palette[6]]
	case 2:
		pal[1] = &RGBPalette[palette[0]]
	case 3:
		pal[1] = &RGBPalette[palette[4]]
	}
	return pal
}
func (m *Map) AreCoordsValid(x, y int) bool {
	if y < 0 || y >= m.Height || x < 0 || x >= m.Width-y%2 {
		return false
	}
	return true
}
func (m *Map) IsIndexValid(ix int) bool {
	return ix >= 0 && ix < len(m.terrain)
}

// x, y in map coords, not unit coords
func (m *Map) GetTile(x, y int) byte {
	return m.GetTileAtIndex(y*m.Width + x - y/2)
}
func (m *Map) SetTile(x, y int, tile byte) {
	m.SetTileAtIndex(y*m.Width+x-y/2, tile)
}
func (m *Map) GetTileAtIndex(ix int) byte {
	if ix < 0 || ix >= len(m.terrain) {
		fmt.Println("invalid index", ix)
		return 0
	}
	return m.terrain[ix]
}
func (m *Map) SetTileAtIndex(ix int, tile byte) {
	if ix < 0 || ix >= len(m.terrain) {
		return
	}
	m.terrain[ix] = tile
}

// ParseMap parses CRUSADE.MAP files.
func ParseMap(data io.Reader) (Map, error) {
	reader := bufio.NewReader(data)
	var header [2]byte
	_, err := io.ReadFull(reader, header[:1])
	if err != nil {
		return Map{}, nil
	}
	if header[0] == 0xFF {
		return parseMapConflict(reader)
	}
	_, err = io.ReadFull(reader, header[1:])
	if err != nil {
		return Map{}, nil
	}
	return parseMapCrusade(reader, int(header[0]), int(header[1]))
}

// parseMapCrusade parses CRUSADE.MAP file from CiE and DitD games.
// Two first bytes are used for determining dimensions of the map,
// although it's hardcoded (at least the width) to be 64 in other places in code.
func parseMapCrusade(data io.Reader, width, height int) (Map, error) {
	terrainMap := Map{
		Width: width, Height: height,
		terrain: make([]byte, 0, width*height),
	}

	for y := 0; y < terrainMap.Height; y++ {
		rowLength := terrainMap.Width - y%2
		row := make([]byte, rowLength)
		_, err := io.ReadFull(data, row)
		if err != nil {
			for i := len(terrainMap.terrain); i < width*height; i++ {
				terrainMap.terrain = append(terrainMap.terrain, 0)
			}
			return terrainMap, nil //Map{}, err
		}
		terrainMap.terrain = append(terrainMap.terrain, row...)
	}
	return terrainMap, nil
}

// parseMapConflict parses CRUSADE.MAP file from the CiV game.
// Two map dimensions are hardcoded to be 64x64.
func parseMapConflict(data *bufio.Reader) (Map, error) {
	var header1 [3]byte
	_, err := io.ReadFull(data, header1[:])
	if err != nil {
		return Map{}, nil
	}
	var decodedData []byte
	for {
		b, err := data.ReadByte()
		if err != nil {
			if err != io.EOF {
				return Map{}, err
			}
			break
		}
		if b != 0xff {
			decodedData = append(decodedData, b)
		} else {
			var bCnt [2]byte
			if _, err := io.ReadFull(data, bCnt[:]); err != nil {
				return Map{}, err
			}
			count := int(bCnt[1])
			if count == 0xff {
				b, err := data.ReadByte()
				if err != nil {
					return Map{}, err
				}
				count += int(b)
			}
			for i := 0; i < count+4; i++ {
				decodedData = append(decodedData, bCnt[0])
			}
		}
	}
	return parseMapCrusade(bytes.NewReader(decodedData), 64, 64)
}

func ReadMap(dirname string) (Map, error) {
	mapFilename := path.Join(dirname, "CRUSADE.MAP")
	mapFile, err := os.Open(mapFilename)
	if err != nil {
		return Map{}, fmt.Errorf("Cannot open map file %s. %v", mapFilename, err)
	}
	defer mapFile.Close()
	return ParseMap(mapFile)
}
