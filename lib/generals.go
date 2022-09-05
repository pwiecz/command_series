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
	Data0   byte
	Data0_0 bool // bit 0
	Data0_1 bool // bit 1
	Data0_2 bool // bit 2
	Data0_3 bool // bit 3
	Data0_4 bool // bit 4
	Data0_5 bool // bit 5
	Data0_6 bool // bit 6
	Data0_7 bool // bit 7
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
		general.Data0 = generalData[0]
		general.Data0_0 = generalData[0]&1 != 0
		general.Data0_1 = generalData[0]&2 != 0
		general.Data0_2 = generalData[0]&4 != 0
		general.Data0_3 = generalData[0]&8 != 0
		general.Data0_4 = generalData[0]&16 != 0
		general.Data0_5 = generalData[0]&32 != 0
		general.Data0_6 = generalData[0]&64 != 0
		general.Data0_7 = generalData[0]&128 != 0
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
