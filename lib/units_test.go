package lib

import "bytes"
import "path"
import "os/user"
import "reflect"
import "testing"

import "github.com/pwiecz/command_series/atr"

func TestParseEncodeParseUnits(t *testing.T) {
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
	if err := EncodeUnits(scenarioData.Units, &buf); err != nil {
		t.Fatal("Error encoding buffer,", err)
	}
	units, err := ParseUnits(&buf, scenarioData.Data.UnitTypes, scenarioData.Data.UnitNames, scenarioData.Generals)
	if err != nil {
		t.Fatal("Error reparsing units,", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("Unread %d bytes remained in the encoded units buffer", buf.Len())
	}
	if !reflect.DeepEqual(scenarioData.Units, units) {
		for side := 0; side < 2; side++ {
			if len(scenarioData.Units[side]) != len(units[side]) {
				t.Fatalf("Different number of units for side %d, %d vs %d", side, len(scenarioData.Units[side]), len(units[side]))
			}
			for i, unit := range units[side] {
				if !reflect.DeepEqual(scenarioData.Units[side][i], unit) {
					t.Fatalf("Units %d,%d different, \n%v\nvs\n%v", side, i, scenarioData.Units[side][i], unit)
				}
			}
		}
	}
}
