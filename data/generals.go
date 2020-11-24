package data

import "fmt"
import "io"
import "os"

// Represenation of data parsed from {scenario}.GEN files.
type General struct {
	Data0     int
	Data1Low  int // attack?
	Data1High int
	Data2Low  int // defence?
	Data2High int
	Data3Low  int // movement?
	Name      string
}

func ReadGenerals(filename string) ([2][]General, error) {
	file, err := os.Open(filename)
	if err != nil {
		return [2][]General{}, fmt.Errorf("Cannot open generals file %s, %v", filename, err)
	}
	defer file.Close()
	generals, err := ParseGenerals(file)
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
		general.Data1Low = int(generalData[1] & 15)
		general.Data1High = int(int8(generalData[1]&240)) / 16
		general.Data2Low = int(generalData[2] & 15)
		general.Data2High = int(int8(generalData[2]&240)) / 16
		general.Data3Low = int(generalData[3] & 15)
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
