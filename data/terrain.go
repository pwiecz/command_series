package data

import "bufio"
import "bytes"
import "fmt"
import "io"
import "os"

// Representation of data parsed from a {scenario}.TER file.
type City struct {
	Owner         int
	VictoryPoints int
	X, Y          int
	VariantBitmap byte
	Name          string
}

type Terrain struct {
	Cities []City
	// Coefficients for 4x4-tile squares on the map (a 16x16 map of coefficients).
	// n-th (0-based) coefficient, if a coefficient for with top left corner:
	// (4*(n%16), 4*n/16).
	Coeffs [16][16]int // Bytes [768-1024]
}

func ReadTerrain(filename string) (Terrain, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Terrain{}, fmt.Errorf("Cannot open terrain file %s, %v", filename, err)
	}
	defer file.Close()
	terrain, err := ParseTerrain(file)
	if err != nil {
		return Terrain{}, fmt.Errorf("Cannot parse terrain file %s, %v", filename, err)
	}
	return terrain, nil
}

func ParseCity(data io.Reader) (City, error) {
	var cityData [16]byte
	_, err := io.ReadFull(data, cityData[:])
	if err != nil {
		return City{}, err
	}
	if cityData[0]&128 == 0 {
		return City{}, nil
	}
	var city City
	city.Owner = int((cityData[0] & 64) >> 6)
	city.VictoryPoints = int(cityData[0] & 63)
	city.X = int(cityData[1])
	city.Y = int(cityData[2])
	city.VariantBitmap = cityData[3]
	name := cityData[4:]
	for len(name) > 0 && (name[len(name)-1] == 0x20 || name[len(name)-1] == 0) {
		name = name[:len(name)-1]
	}
	city.Name = string(name)
	return city, nil
}

func ParseTerrain(data io.Reader) (Terrain, error) {
	reader := bufio.NewReader(data)
	b, err := reader.Peek(1)
	if err != nil {
		return Terrain{}, err
	}
	if b[0] == 0xff {
		return parseTerrainConflict(reader)
	} else {
		return parseTerrainCrusade(reader)
	}
}

func parseTerrainCrusade(data io.Reader) (Terrain, error) {
	var terrain Terrain
	for i := 0; i < 48; i++ {
		city, err := ParseCity(data)
		if err != nil {
			return Terrain{}, err
		}
		if len(city.Name) == 0 {
			continue
		}
		terrain.Cities = append(terrain.Cities, city)
	}
	var coeffData [256]byte
	_, err := io.ReadFull(data, coeffData[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return Terrain{}, err
	}
	for i, v := range coeffData {
		terrain.Coeffs[i%16][i/16] = int(v)
	}
	return terrain, nil
}

func parseTerrainConflict(data *bufio.Reader) (Terrain, error) {
	var prefix [5]byte
	if _, err := io.ReadFull(data, prefix[:]); err != nil {
		return Terrain{}, err
	}
	var decodedData []byte
	for {
		b, err := data.ReadByte()
		if err != nil {
			if err != io.EOF {
				return Terrain{}, err
			}
			break
		}
		if b != 0xff {
			decodedData = append(decodedData, b)
		} else {
			var bCnt [2]byte
			if _, err := io.ReadFull(data, bCnt[:]); err != nil {
				return Terrain{}, err
			}
			for i := 0; i < int(bCnt[1])+4; i++ {
				decodedData = append(decodedData, bCnt[0])
			}
		}
	}
	return parseTerrainCrusade(bytes.NewReader(decodedData))
}
