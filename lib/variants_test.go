package lib

import "bytes"
import "testing"

func TestParseEncodeParseVariant(t *testing.T) {
	_, scenarioData, err := readTestData("crusade.atr", 0)
	if err != nil {
		t.Fatal("Error reading game data,", err)
	}

	// D-Day: Rommel's Strategy
	variant := scenarioData.Variants[1]
	if variant.Name != "D-DAY: ROMMEL'S STRATEGY" {
		t.Errorf("Invalid variant name")
	}
	if variant.LengthInDays != 6 {
		t.Errorf("Invalid variant length")
	}
	if variant.CriticalLocations[0] != 3 || variant.CriticalLocations[1] != 2 {
		t.Errorf("Invalid number of critical locations")
	}
	if variant.Data3 != 8 {
		t.Errorf("Invalid data3 parameter")
	}
	if variant.CitiesHeld[0] != 0 || variant.CitiesHeld[1] != 120 {
		t.Errorf("Invalid number of cities held")
	}

	var buf bytes.Buffer
	if err := variant.Write(&buf); err != nil {
		t.Fatal("Error encoding variant,", err)
	}
	var variant2 Variant
	if err := variant2.Read(&buf); err != nil {
		t.Fatal("Error reparsing variant,", err)
	}
	if variant != variant2 {
		t.Errorf("Variants differ after reparsing")
	}
}
