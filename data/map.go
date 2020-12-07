package data

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
func ParseMap(data io.Reader, width, height int) (Map, error) {
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

func ReadMap(dirname string, game Game) (Map, error) {
	filename := path.Join(dirname, "CRUSADE.MAP")
	file, err := os.Open(filename)
	if err != nil {
		return Map{}, fmt.Errorf("Cannot open map file %s. %v", filename, err)
	}
	defer file.Close()
	var reader io.Reader
	if game == Conflict {
		decoded, err := UnpackFile(file)
		if err != nil {
			return Map{}, err
		}
		reader = bytes.NewReader(decoded)
	} else {
		// Skip first two bytes of the file (they are all zeroes).
		var header [2]byte
		if _, err := io.ReadFull(file, header[:]); err != nil {
			return Map{}, err
		}
		reader = file
	}
	terrainMap, err := ParseMap(reader, 64, 64)
	if err != nil {
		return Map{}, fmt.Errorf("Cannot parse map file %s, %v", filename, err)
	}
	return terrainMap, nil
}
