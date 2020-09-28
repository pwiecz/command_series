package data

import "fmt"
import "io"
import "os"

// Represenation of data parsed from {scenario}.GEN files.
type General struct {
	Data [4]byte
	Name string
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
		_, err := io.ReadFull(data, general.Data[:])
		if err != nil {
			return generals, err
		}
		generalName := make([]byte, 12)
		_, err = io.ReadFull(data, generalName)
		for len(generalName) > 0 && generalName[len(generalName)-1] == 0 {
			generalName = generalName[0 : len(generalName)-1]
		}
		if len(generalName) == 0 {
			continue
		}
		general.Name = string(generalName)
		generals[i/8] = append(generals[i/8], general)
	}
	return generals, nil
}
