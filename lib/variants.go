package lib

import "bytes"
import "fmt"
import "io"

import "github.com/pwiecz/command_series/atr"

// Representation of data parsed from {scenario}.VAR file.
type Variant struct {
	Name              string
	LengthInDays      int
	CriticalLocations [2]int // per side. Number of critical locations that need to be captured by a side to win.
	Data3             int
	CitiesHeld        [2]int
}

func ReadVariants(diskimage atr.SectorReader, filename string) ([]Variant, error) {
	variantsData, err := atr.ReadFile(diskimage, filename)
	if err != nil {
		return nil, fmt.Errorf("Cannot read variants file %s (%v)", filename, err)
	}
	variants, err := ParseVariants(bytes.NewBuffer(variantsData))
	if err != nil {
		return variants, err
	}
	return variants, nil
}

func ParseVariants(reader io.Reader) ([]Variant, error) {
	var variants []Variant
	for {
		var variant Variant
		err := variant.Read(reader)
		if err == io.EOF || variant.Name == "X" {
			break
		} else if err != nil {
			return nil, err
		}
		variants = append(variants, variant)
	}
	return variants, nil
/*	segments := bytes.Split(data, []byte{0x9b})
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
		variant.Data3 = int(variantData[3])
		variant.CitiesHeld[0] = int(variantData[4]) * 10
		variant.CitiesHeld[1] = int(variantData[5]) * 10
		variants = append(variants, variant)
	}
	return variants, nil*/
}

func (v Variant) Write(writer io.Writer) error {
	if _, err := io.WriteString(writer, v.Name); err != nil {
		return err
	}
	var data [7]byte
	data[0] = 0x9b
	data[1] = byte(v.LengthInDays)
	data[2] = byte(v.CriticalLocations[0])
	data[3] = byte(v.CriticalLocations[1])
	data[4] = byte(v.Data3)
	data[5] = byte(v.CitiesHeld[0] / 10)
	data[6] = byte(v.CitiesHeld[1] / 10)
	if _, err := writer.Write(data[:]); err != nil {
		return err
	}
	return nil
}

func (v *Variant) Read(reader io.Reader) error {
	var name []byte
	var buf [1]byte
	for {
		if _, err := reader.Read(buf[:]); err != nil {
			return err
		}
		if buf[0] == 0x9b {
			break
		}
		name = append(name, buf[0])
	}
	v.Name = string(name)
	var data [6]byte
	if _, err := reader.Read(data[:]); err != nil {
		return err
	}
	v.LengthInDays = int(data[0])
	v.CriticalLocations[0] = int(data[1])
	v.CriticalLocations[1] = int(data[2])
	v.Data3 = int(data[3])
	v.CitiesHeld[0] = int(data[4]) * 10
	v.CitiesHeld[1] = int(data[5]) * 10
	return nil
}
