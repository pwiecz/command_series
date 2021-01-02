package lib

import "bytes"
import "path"
import "os/user"
import "reflect"
import "testing"

import "github.com/pwiecz/command_series/atr"

func TestParseEncodeParseOwnerAndVictoryPoints(t *testing.T) {
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
	if err := scenarioData.Terrain.Cities.WriteOwnerAndVictoryPoints(&buf); err != nil {
		t.Fatal("Error encoding owners and victory points,", err)
	}
	var cities Cities
	for _, city := range scenarioData.Terrain.Cities {
		city.Owner = 0
		city.VictoryPoints = 0
		cities = append(cities, city)
	}

	if err := cities.ReadOwnerAndVictoryPoints(&buf); err != nil {
		t.Fatal("Error parsing owners and victory points,", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("Unread %d bytes remained in the encoded owners and victory points buffer", buf.Len())
	}
	if !reflect.DeepEqual(scenarioData.Terrain.Cities, cities) {
		if len(scenarioData.Terrain.Cities) != len(cities) {
			t.Fatalf("Different number of cities, %d vs %d", len(scenarioData.Terrain.Cities), len(cities))
		}
		for i, city := range cities {
			if !reflect.DeepEqual(scenarioData.Terrain.Cities[i], city) {
				t.Fatalf("City %d different, \n%v\nvs\n%v", i, scenarioData.Terrain.Cities[i], city)
			}
		}
	}
}
