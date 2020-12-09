package data

import "bytes"
import "fmt"
import "io"

import "github.com/pwiecz/command_series/atr"

// Represenation of data parsed from {scenario}.GEN files.
type General struct {
	Data0     int
	Attack    int
	Data1High int
	Defence   int
	Data2High int
	Movement  int
	Name      string
}

func ReadGenerals(diskimage atr.SectorReader, filename string) ([2][]General, error) {
	fileData, err := atr.ReadFile(diskimage, filename)
	if err != nil {
		return [2][]General{}, fmt.Errorf("Cannot read generals file %s, %v", filename, err)
	}
	generals, err := ParseGenerals(bytes.NewReader(fileData))
	if err != nil {
		return [2][]General{}, fmt.Errorf("Cannot parse generals file %s, %v", filename, err)
	}
	return generals, nil
}

func ParseGenerals(data io.Reader) ([2][]General, error) {
	var generals [2][]General
	for i := 0; i < 16; i++ {
		var general General
		var generalData [4]byte
		_, err := io.ReadFull(data, generalData[:])
		if err != nil {
			return generals, err
		}
		general.Data0 = int(generalData[0])
		general.Attack = int(generalData[1] & 15)
		general.Data1High = int(int8(generalData[1]&240)) / 16
		general.Defence = int(generalData[2] & 15)
		general.Data2High = int(int8(generalData[2]&240)) / 16
		general.Movement = int(generalData[3] & 15)
		generalName := make([]byte, 12)
		_, err = io.ReadFull(data, generalName)
		for len(generalName) > 0 && generalName[len(generalName)-1] == 0 {
			generalName = generalName[0 : len(generalName)-1]
		}
		general.Name = string(generalName)
		generals[i/8] = append(generals[i/8], general)
	}
	return generals, nil
}
