package data

import "bytes"
import "fmt"
import "io/ioutil"
import "os"
import "path"
import "strconv"
import "strings"

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

func ReadScenarios(dirname string) ([]Scenario, error) {
	var scenarios []Scenario
	dir, err := os.Open(dirname)
	if err != nil {
		return scenarios, fmt.Errorf("Cannot open directory %s, %v\n", dirname, err)
	}
	defer dir.Close()
	dirInfo, err := dir.Stat()
	if err != nil {
		return scenarios, fmt.Errorf("Cannot get info about directory %s, %v\n", dirname, err)
	}
	if !dirInfo.IsDir() {
		return scenarios, fmt.Errorf("%s is not a directory\n", dirname)
	}

	filenames, err := dir.Readdirnames(0)
	if err != nil {
		return scenarios, fmt.Errorf("Cannot list directory %s, %v\n", dirname, err)
	}

	for _, filename := range filenames {
		if strings.HasSuffix(filename, ".SCN") {
			scenarioFilename := path.Join(dirname, filename)
			scenario, err := ReadScenario(scenarioFilename)
			if err != nil {
				return scenarios, err
			}
			scenarios = append(scenarios, scenario)
		}
	}

	if len(scenarios) == 0 {
		return scenarios, fmt.Errorf("No scenarios found in directory %s\n", dirname)
	}
	return scenarios, nil
}

func ReadScenario(filename string) (Scenario, error) {
	var scenario Scenario
	file, err := os.Open(filename)
	if err != nil {
		return scenario, fmt.Errorf("Cannot open scenario file %s, %v\n", filename, err)
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return scenario, fmt.Errorf("Cannot read scenario file %s, %v\n", filename, err)
	}
	scenario, err = ParseScn(data)
	if err != nil {
		return scenario, fmt.Errorf("Cannot parse scenario file %s, %v\n", filename, err)
	}
	return scenario, err
}

func ParseScn(data []byte) (Scenario, error) {
	var result Scenario
	segments := bytes.SplitN(data, []byte{0x9b}, 11)
	if len(segments) != 11 {
		return result, fmt.Errorf("Expected 11 segments, got %d", len(segments))
	}
	result.Name = string(segments[0])
	result.FilePrefix = string(segments[1])
	if !strings.HasPrefix(result.FilePrefix, "D:") {
		return result, fmt.Errorf("Unexpected scenario file prefix: \"%s\"", result.FilePrefix)
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
