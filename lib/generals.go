package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
)

// Represenation of data parsed from {scenario}.GEN files.
type General struct {
	// A bitmap of various flags of the general
	Data0 int
	// Attack bonus for the units commanded by the general from 0 to 15
	Attack    int
	Data1High int
	// Defence bonus for the units commanded by the general from 0 to 15
	Defence   int
	Data2High int
	// Movement bonus for the units commanded by the general from 0 to 15
	Movement int
	Name     string
}

type Generals [2][]General

func ReadGenerals(fsys fs.FS, filename string) (*Generals, error) {
	fileData, err := fs.ReadFile(fsys, filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read generals file %s (%v)", filename, err)
	}
	generals, err := ParseGenerals(bytes.NewReader(fileData))
	if err != nil {
		return nil, fmt.Errorf("cannot parse generals file %s (%v)", filename, err)
	}
	return generals, nil
}

func ParseGenerals(data io.Reader) (*Generals, error) {
	var generals Generals
	for i := 0; i < 16; i++ {
		var general General
		var generalData [4]byte
		_, err := io.ReadFull(data, generalData[:])
		if err != nil && err != io.EOF {
			return nil, err
		}
		general.Data0 = int(generalData[0])
		general.Attack = int(generalData[1] & 15)
		general.Data1High = int(int8(generalData[1]&240)) / 16
		general.Defence = int(generalData[2] & 15)
		general.Data2High = int(int8(generalData[2]&240)) / 16
		general.Movement = int(generalData[3] & 15)
		generalName := make([]byte, 12)
		io.ReadFull(data, generalName)
		for len(generalName) > 0 && generalName[len(generalName)-1] == 0 {
			generalName = generalName[0 : len(generalName)-1]
		}
		general.Name = string(generalName)
		generals[i/8] = append(generals[i/8], general)
	}
	return &generals, nil
}
