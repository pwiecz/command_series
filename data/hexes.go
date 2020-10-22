package data

import "fmt"
import "io"
import "os"
import "path"

// Representation of data parsed from HEXES.DTA file.
type Hexes struct {
	Arr0 [48]int // Data[0:48]
	Arr48 [48]int // Data[48:96]
	Arr96 [48]int // Data[96:144]
	Arr144 [48]int // Data[144:192]
	// 4 bytes per general/side
	Arr3 [2][8][4]int // Data[192:256]
}

func ReadHexes(dirname string) (Hexes, error) {
	filename := path.Join(dirname, "HEXES.DTA")
	file, err := os.Open(filename)
	if err != nil {
		return Hexes{}, fmt.Errorf("Cannot open hexes file %s, %v", filename, err)
	}
	defer file.Close()
	return ParseHexes(file)
}

func ParseHexes(reader io.Reader) (Hexes, error) {
	var data [256]byte
	_, err := io.ReadFull(reader, data[:])
	if err != nil {
		return Hexes{}, err
	}

	var hexes Hexes
	for i, val := range data[0:48] {
		hexes.Arr0[i] = int(val)
	}
	for i, val := range data[48:96] {
		hexes.Arr48[i] = int(val)
	}
	for i, val := range data[96:144] {
		hexes.Arr96[i] = int(val)
	}
	for i, val := range data[144:192] {
		hexes.Arr144[i] = int(val)
	}

	for side := 0; side < 2; side++ {
		for general := 0; general < 8; general++ {
			generalDataStart := 192 + side*32 + general*4
			for i, val := range data[generalDataStart : generalDataStart+4] {
				hexes.Arr3[side][general][i] = int(val)
			}
		}
	}

	return hexes, nil
}
