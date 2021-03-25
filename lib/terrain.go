package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
)

// Representation of data parsed from a {scenario}.TER file.
type City struct {
	Owner         int
	VictoryPoints int
	XY            UnitCoords
	VariantBitmap byte
	Name          string
}

type Cities []City

type Terrain struct {
	Cities Cities
	// Coefficients for 4x4-tile squares on the map (a 16x16 map of coefficients).
	// n-th (0-based) coefficient, if a coefficient for square with top left corner:
	// (4*(n%16), 4*n/16).
	Coeffs [16][16]int // Bytes [768-1024]
}

func (t Terrain) IsCityAt(xy UnitCoords) bool {
	for _, city := range t.Cities {
		if city.VictoryPoints > 0 && city.XY == xy {
			return true
		}
	}
	return false
}
func (t Terrain) FindCityAt(xy UnitCoords) (*City, bool) {
	for i, city := range t.Cities {
		if city.VictoryPoints > 0 && city.XY == xy {
			return &t.Cities[i], true
		}
	}
	return nil, false
}

func ReadTerrain(fsys fs.FS, filename string, game Game) (*Terrain, error) {
	fileData, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return nil, fmt.Errorf("Cannot open terrain file %s (%v)", filename, err)
	}
	var reader io.Reader
	if game == Conflict {
		decoded, err := UnpackFile(bytes.NewReader(fileData))
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(decoded)
	} else {
		reader = bytes.NewReader(fileData)
	}
	terrain, err := ParseTerrain(reader)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse terrain file %s (%v)", filename, err)
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
	city.XY = UnitCoords{int(cityData[1]), int(cityData[2])}
	city.VariantBitmap = cityData[3]
	name := cityData[4:]
	for len(name) > 0 && (name[len(name)-1] == 0x20 || name[len(name)-1] == 0) {
		name = name[:len(name)-1]
	}
	city.Name = string(name)
	return city, nil
}

func ParseTerrain(data io.Reader) (*Terrain, error) {
	terrain := &Terrain{}
	for i := 0; i < 48; i++ {
		city, err := ParseCity(data)
		if err != nil {
			return nil, err
		}
		if len(city.Name) == 0 {
			continue
		}
		terrain.Cities = append(terrain.Cities, city)
	}
	var coeffData [256]byte
	_, err := io.ReadFull(data, coeffData[:])
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	for i, v := range coeffData {
		terrain.Coeffs[i%16][i/16] = int(v)
	}
	return terrain, nil
}

func (c *Cities) ReadOwnerAndVictoryPoints(data io.Reader) error {
	var buf [1]byte
	if _, err := io.ReadFull(data, buf[:]); err != nil {
		return err
	}
	numCities := int(buf[0])
	if numCities != len(*c) {
		return fmt.Errorf("Mismatched number of cities, %d vs %d", numCities, len(*c))
	}
	for i := 0; i < numCities; i++ {
		if _, err := io.ReadFull(data, buf[:]); err != nil {
			return err
		}
		(*c)[i].Owner = int((buf[0] & 64) >> 6)
		(*c)[i].VictoryPoints = int(buf[0] & 63)
	}
	return nil
}

func (c Cities) WriteOwnerAndVictoryPoints(writer io.Writer) error {
	if len(c) > 255 {
		return fmt.Errorf("Too many cities to encode %d", len(c))
	}
	if _, err := writer.Write([]byte{byte(len(c))}); err != nil {
		return err
	}
	for _, city := range c {
		b := byte(city.Owner<<6) + byte(city.VictoryPoints)
		if _, err := writer.Write([]byte{b}); err != nil {
			return err
		}
	}
	return nil
}
