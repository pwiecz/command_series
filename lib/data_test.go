package lib

import "bytes"
import "path"
import "os/user"
import "reflect"
import "testing"

import "github.com/pwiecz/command_series/atr"

func TestParseEncodeParseDataFirst255Bytes(t *testing.T) {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatal("Cannot get current user info", err)
	}
	atrFile := path.Join(currentUser.HomeDir, "command_series", "crusade.atr")
	diskimage, err := atr.NewAtrSectorReader(atrFile)
	if err != nil {
		t.Fatalf("Cannot read diskimage %s, %v", atrFile, err)
	}
	gameData, err := LoadGameData(diskimage)
	if err != nil {
		t.Fatal("Error loading game data,", err)
	}
	scenarioData, err := LoadScenarioData(diskimage, gameData.Scenarios[0].FilePrefix)
	if err != nil {
		t.Fatal("Error loading data for scenario 0", err)
	}

	var buf bytes.Buffer
	if err := scenarioData.Data.WriteFirst255Bytes(&buf); err != nil {
		t.Fatal("Error encoding data,", err)
	}
	// copy of fields of scenarioData.Data past first 255 bytes
	var data Data
	data.MoveSpeedPerTerrainTypeAndUnit = scenarioData.Data.MoveSpeedPerTerrainTypeAndUnit
	data.PossibleWeather = scenarioData.Data.PossibleWeather
	data.DaytimePalette = scenarioData.Data.DaytimePalette
	data.NightPalette = scenarioData.Data.NightPalette
	data.MenCountLimit = scenarioData.Data.MenCountLimit
	data.EquipCountLimit = scenarioData.Data.EquipCountLimit
	data.DataUpdates = scenarioData.Data.DataUpdates
	data.UnitTypes = scenarioData.Data.UnitTypes
	data.Strings1 = scenarioData.Data.Strings1
	data.Formations = scenarioData.Data.Formations
	data.Experience = scenarioData.Data.Experience
	data.Strings4 = scenarioData.Data.Strings4
	data.Equipments = scenarioData.Data.Equipments
	data.UnitNames = scenarioData.Data.UnitNames
	data.Strings7 = scenarioData.Data.Strings7
	data.Strings9 = scenarioData.Data.Strings9
	data.Months = scenarioData.Data.Months
	data.Sides = scenarioData.Data.Sides
	data.Weather = scenarioData.Data.Weather
	data.Colors = scenarioData.Data.Colors
	if err := data.ReadFirst255Bytes(&buf); err != nil {
		t.Fatal("Error parsing data,", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("Unread %d bytes remained in the encoded data buffer", buf.Len())
	}
	if !reflect.DeepEqual(scenarioData.Data, data) {
		t.Errorf("Reparsed data differ")
		v1 := reflect.ValueOf(scenarioData.Data)
		v2 := reflect.ValueOf(data)
		for i := 0; i < v1.NumField(); i++ {
			f1 := v1.Field(i)
			f2 := v2.Field(i)
			if !reflect.DeepEqual(f1.Interface(), f2.Interface()) {
				t.Errorf("%s field differs, %v vs %v", reflect.TypeOf(data).Field(i).Name, f1.Interface(), f2.Interface())
			}
		}
	}
	var buf1 bytes.Buffer
	if err := scenarioData.Data.WriteFirst255Bytes(&buf1); err != nil {
		t.Fatal("Error encoding data,", err)
	}
	var buf2 bytes.Buffer
	if err := data.WriteFirst255Bytes(&buf2); err != nil {
		t.Fatal("Error encoding reparsed data,", err)
	}
	bytes1 := buf1.Bytes()
	bytes2 := buf2.Bytes()
	if len(bytes1) != len(bytes2) {
		t.Fatalf("Two data encodings differ in length, %d vs %d", len(bytes1), len(bytes2))
	}
	for i, b1 := range bytes1 {
		if b1 != bytes2[i] {
			t.Errorf("Two data encodings differ at position %d, %d vs %d", i, b1, bytes2[i])
		}
	}
}
