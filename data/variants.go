package data

import "bytes"
import "errors"
import "fmt"
import "io/ioutil"
import "os"

// Representation of data parsed from {scenario}.VAR file.
type Variant struct {
	Name              string
	LengthInDays      int
	CriticalLocations [2]int // per side. Number of critical locations that need to be captured by a side to win.
	Data2             int
	CitiesHeld        [2]int
}

func ReadVariants(filename string) ([]Variant, error) {
	var variants []Variant
	variantsFile, err := os.Open(filename)
	if err != nil {
		return variants, fmt.Errorf("Cannot open variants file %s, %v\n", filename, err)
	}
	defer variantsFile.Close()
	variantsData, err := ioutil.ReadAll(variantsFile)
	if err != nil {
		return variants, fmt.Errorf("Cannot read variants file %s, %v\n", filename, err)
	}
	variants, err = ParseVariants(variantsData)
	if err != nil {
		return variants, err
	}
	return variants, nil
}

func ParseVariants(data []byte) ([]Variant, error) {
	var variants []Variant
	segments := bytes.Split(data, []byte{0x9b})
	for i := 0; i < len(segments)-1; i++ {
		var variant Variant
		variant.Name = string(segments[i])
		if i > 0 {
			variant.Name = string(segments[i][6:])
		}
		if variant.Name == "X" {
			break
		}
		if len(segments[i+1]) < 6 {
			return nil, errors.New("Too short variant segment")
		}
		variantData := segments[i+1][0:6]
		variant.LengthInDays = int(variantData[0])
		variant.CriticalLocations[0] = int(variantData[1])
		variant.CriticalLocations[1] = int(variantData[2])
		variant.Data2 = int(variantData[3])
		variant.CitiesHeld[0] = int(variantData[4]) * 10
		variant.CitiesHeld[1] = int(variantData[5]) * 10
		variants = append(variants, variant)
	}
	return variants, nil
}
