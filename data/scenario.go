package data

import "bytes"
import "fmt"
import "strconv"
import "strings"

import "github.com/pwiecz/command_series/atr"

// Representation of the data parsed from a {scenario}.SCN file.
type Scenario struct {
	Name                   string
	FilePrefix             string
	StartMinute            int
	StartHour              int
	StartDay               int
	StartMonth             int
	StartYear              int
	StartWeather           int
	StartSupplyLevels      [2]int
	MinX, MaxX, MinY, MaxY int
}

func ReadScenarios(diskimage atr.SectorReader) ([]Scenario, error) {
	files, err := atr.GetDirectory(diskimage)
	if err != nil {
		return nil, fmt.Errorf("Cannot list contents of the disk image (%v)", err)
	}

	var scenarios []Scenario
	for _, file := range files {
		if strings.HasSuffix(file.Name, ".SCN") {
			scenario, err := ReadScenario(diskimage, file.Name)
			if err != nil {
				return nil, err
			}
			scenarios = append(scenarios, scenario)
		}
	}

	if len(scenarios) == 0 {
		return nil, fmt.Errorf("No scenarios found in the disk image")
	}
	return scenarios, nil
}

func ReadScenario(diskimage atr.SectorReader, filename string) (Scenario, error) {
	data, err := atr.ReadFile(diskimage, filename)
	if err != nil {
		return Scenario{}, fmt.Errorf("Cannot read scenario file %s (%v)", filename, err)
	}
	scenario, err := ParseScn(data)
	if err != nil {
		return Scenario{}, fmt.Errorf("Cannot parse scenario file %s (%v)\n", filename, err)
	}
	return scenario, err
}

func ParseScn(data []byte) (Scenario, error) {
	segments := bytes.SplitN(data, []byte{0x9b}, 11)
	if len(segments) != 11 {
		return Scenario{}, fmt.Errorf("Expected 11 segments, got %d", len(segments))
	}
	var result Scenario
	result.Name = string(segments[0])
	result.FilePrefix = string(segments[1])
	if !strings.HasPrefix(result.FilePrefix, "D:") {
		return Scenario{}, fmt.Errorf("Unexpected scenario file prefix: \"%s\"", result.FilePrefix)
	}
	result.FilePrefix = result.FilePrefix[2:]
	var err error
	result.StartMinute, err = strconv.Atoi(string(segments[2]))
	if err != nil {
		return result, fmt.Errorf("Cannot parse scenario start minute: \"%s\"", string(segments[2]))
	}
	result.StartHour, err = strconv.Atoi(string(segments[3]))
	if err != nil {
		return result, fmt.Errorf("Cannot parse scenario start hour: \"%s\"", string(segments[3]))
	}
	result.StartDay, err = strconv.Atoi(string(segments[4]))
	if err != nil {
		return result, fmt.Errorf("Cannot parse scenario start day: \"%s\"", string(segments[4]))
	}
	result.StartMonth, err = strconv.Atoi(string(segments[5]))
	if err != nil {
		return result, fmt.Errorf("Cannot parse scenario start month: \"%s\"", string(segments[5]))
	}
	result.StartYear, err = strconv.Atoi(string(segments[6]))
	if err != nil {
		return result, fmt.Errorf("Cannot parse scenario start year: \"%s\"", string(segments[6]))
	}
	// segments[7] - start month string
	// segments[8] - start weather string
	result.StartWeather, err = strconv.Atoi(string(segments[9]))
	if err != nil {
		return result, fmt.Errorf("Cannot parse scenario start weather: \"%s\"", string(segments[9]))
	}
	if len(segments[10]) != 8 {
		return result, fmt.Errorf("Expected length of binary data segment 8, got %d", len(segments[10]))
	}

	result.StartSupplyLevels[0] = int(segments[10][0]) + 256*int(segments[10][1])
	result.StartSupplyLevels[1] = int(segments[10][2]) + 256*int(segments[10][3])
	result.MinX = int(segments[10][4])
	result.MaxX = int(segments[10][5])
	result.MinY = int(segments[10][6])
	result.MaxY = int(segments[10][7])
	return result, nil
}
