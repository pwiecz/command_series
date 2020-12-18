package lib

import "bytes"
import "fmt"
import "io"

import "github.com/pwiecz/command_series/atr"

// Representation of data parsed from HEXES.DTA file.
type Hexes struct {
	Arr0   [48]int // Data[0:48]
	Arr48  [48]int // Data[48:96]
	Arr96  [48]int // Data[96:144]
	Arr144 [48]int // Data[144:192]
}

func ReadHexes(diskimage atr.SectorReader) (Hexes, error) {
	fileData, err := atr.ReadFile(diskimage, "HEXES.DTA")
	if err != nil {
		return Hexes{}, fmt.Errorf("Cannot read HEXES.DTA file (%v)", err)
	}
	return ParseHexes(bytes.NewReader(fileData))
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
	// Last 64 bytes is always zero as it gets overwritten with .GEN data, ignore it.

	return hexes, nil
}
