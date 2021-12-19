package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
)

// Representation of data parsed from HEXES.DTA file.
type Hexes struct {
	Arr0   [6][8]int // Data[0:48]
	Arr48  [6][8]int // Data[48:96]
	Arr96  [6][8]int // Data[96:144]
	Arr144 [6][8]int // Data[144:192]
}

func ReadHexes(fsys fs.FS) (*Hexes, error) {
	fileData, err := fs.ReadFile(fsys, "HEXES.DTA")
	if err != nil {
		return nil, fmt.Errorf("cannot read HEXES.DTA file (%v)", err)
	}
	return ParseHexes(bytes.NewReader(fileData))
}

func ParseHexes(reader io.Reader) (*Hexes, error) {
	var data [256]byte
	_, err := io.ReadFull(reader, data[:])
	if err != nil {
		return nil, err
	}

	hexes := &Hexes{}
	for i, val := range data[0:48] {
		hexes.Arr0[i/8][i%8] = int(int8(val))
	}
	for i, val := range data[48:96] {
		hexes.Arr48[i/8][i%8] = int(int8(val))
	}
	for i, val := range data[96:144] {
		hexes.Arr96[i/8][i%8] = int(int8(val))
	}
	for i, val := range data[144:192] {
		hexes.Arr144[i/8][i%8] = int(int8(val))
	}

	// Last 64 bytes is always zero as it gets overwritten with .GEN data, ignore it.

	return hexes, nil
}
