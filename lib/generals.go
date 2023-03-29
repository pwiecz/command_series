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
	Data0    byte
	Data0_26 int // from bits 2 and 6
	Data0_15 int // from bits 1 and 5
	Data0_37 int // from bits 3 and 7
	Data0_04 int // from bits 0 and 4
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

func coefficientFromTwoBits(data byte, pos0, pos1 int) int {
	bit0 := data&(1<<pos0) != 0
	bit1 := data&(1<<pos1) != 0
	if bit0 && bit1 {
		panic("Both bits are set")
	} else if bit0 {
		return 4
	} else if bit1 {
		return 1
	} else {
		return 2
	}
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
		general.Data0_26 = coefficientFromTwoBits(general.Data0, 2, 6)
		general.Data0_15 = coefficientFromTwoBits(general.Data0, 1, 5)
		general.Data0_37 = coefficientFromTwoBits(general.Data0, 3, 7)
		general.Data0_04 = coefficientFromTwoBits(general.Data0, 0, 4)
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
