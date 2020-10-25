package data

import "fmt"
import "image"
import "image/color"
import "image/draw"
import "io"
import "os"
import "path"

// A representation of a hex map parsed from CRUSADE.MAP file.
type Map struct {
	Width, Height int
	Terrain       []byte
}

func GetPalette(n int, palette [8]byte) []color.Color {
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

func (m *Map) GetTile(x, y int) byte {
	return m.Terrain[y*m.Width + x - y/2]
}

// GetImage constructs image.Image object from given set of tiles and given palette.
func (m *Map) GetImage(tiles []*image.Paletted, palette [8]byte) (image.Image, error) {
	if len(tiles) < 48 {
		return nil, fmt.Errorf("Too few tiles. Expected 48, got %d", len(tiles))
	}
	tileBounds := tiles[0].Bounds()
	tileWidth := tileBounds.Max.X - tileBounds.Min.X
	tileHeight := tileBounds.Max.Y - tileBounds.Min.Y
	img := image.NewNRGBA(image.Rect(0, 0, tileWidth*m.Width, tileHeight*m.Height))
	for y := 0; y < m.Height; y++ {
		x0 := (y % 2) * 4
		for x := 0; x < m.Width - y%2; x++ {
			tileNum := int(m.GetTile(x, y) % 64)
			if tileNum >= len(tiles) {
				return nil, fmt.Errorf("Too large tile number. Expected at most 48, got %d", tileNum)
			}
			repalettedImg := *tiles[tileNum]
			repalettedImg.Palette = GetPalette(int(m.GetTile(x,y)/64), palette)
			draw.Draw(img,
				image.Rect(x0, y*tileHeight, x0+tileWidth, (y+1)*tileHeight),
				&repalettedImg,
				image.Point{},
				draw.Over)
			x0 += tileWidth
		}
	}
	return img, nil
}

// ParseMap parses CRUSADE.MAP files.
func ParseMap(data io.Reader) (Map, error) {
	var header [2]byte
	_, err := io.ReadFull(data, header[:1])
	if err != nil {
		return Map{}, nil
	}
	if header[0] == 0xFF {
		return parseMapConflict(data)
	}
	_, err = io.ReadFull(data, header[1:])
	if err != nil {
		return Map{}, nil
	}
	return parseMapCrusade(data, int(header[0]), int(header[1]))
}

// parseMapCrusade parses CRUSADE.MAP file from CiE and DitD games.
// Two first bytes are used for determining dimensions of the map,
// although it's hardcoded (at least the width) to be 64 in other places in code.
func parseMapCrusade(data io.Reader, width, height int) (Map, error) {
	terrainMap := Map{
		Width: width, Height: height,
		Terrain: make([]byte, 0, width*height),
	}
	
	for y := 0; y < terrainMap.Height; y++ {
		rowLength := terrainMap.Width - y%2
		row := make([]byte, rowLength)
		_, err := io.ReadFull(data, row)
		if err != nil {
			return Map{}, err
		}
		terrainMap.Terrain = append(terrainMap.Terrain, row...)
	}
	return terrainMap, nil
}

// parseMapConflict parses CRUSADE.MAP file from the CiV game.
// Two map dimensions are hardcoded to be 64x64.
func parseMapConflict(data io.Reader) (Map, error) {
	var header1 [2]byte
	_, err := io.ReadFull(data, header1[:])
	if err != nil {
		return Map{}, nil
	}
	l1 := int(header1[0]) + 256*int(header1[1])
	var header2 [2]byte
	_, err = io.ReadFull(data, header2[:])
	if err != nil {
		return Map{}, nil
	}
	l2 := int(header2[0]) + 256*int(header2[1])
	fmt.Println(l1, l2, l2-l1)
	terrainMap := Map{Width: 64}
	for y := 0; true; y++ {
		rowLength := terrainMap.Width - y%2
		row := make([]byte, rowLength)
		n, err := io.ReadFull(data, row)
		if err != nil {
			fmt.Printf("Read only %d bytes of map\n", n)
			break
		}
		terrainMap.Terrain = append(terrainMap.Terrain, row...)
		terrainMap.Height++
	}
	return terrainMap, nil
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
