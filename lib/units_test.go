package lib

import "bytes"
import "reflect"
import "testing"

func TestParseEncodeParseUnits(t *testing.T) {
	_, scenarioData, err := readTestData("crusade.atr", 0)
	if err != nil {
		t.Fatal("Error reading game data,", err)
	}

	var buf bytes.Buffer
	if err := scenarioData.Units.Write(&buf); err != nil {
		t.Fatal("Error encoding units,", err)
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
