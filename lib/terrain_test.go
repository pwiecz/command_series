package lib

import (
	"bytes"
	"reflect"
	"testing"
)

func TestParseEncodeParseOwnerAndVictoryPoints(t *testing.T) {
	_, scenarioData, err := readTestData("crusade.atr", 0)
	if err != nil {
		t.Fatal("Error reading game data,", err)
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
