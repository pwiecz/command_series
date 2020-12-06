package data

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
	var reader io.Reader
	if FileNameToGame(filename) == Conflict {
		decoded, err := UnpackFile(file)
		if err != nil {
			return Terrain{}, err
		}
		reader = bytes.NewReader(decoded)
	} else {
		// Skip first two bytes of the file (they are all zeroes).
		var header [2]byte
		if _, err := io.ReadFull(file, header[:]); err != nil {
			return Terrain{}, err
		}
		reader = file
	}
	terrain, err := ParseTerrain(reader)
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
