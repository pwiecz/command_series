package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pwiecz/command_series/atr"
	"github.com/pwiecz/command_series/lib"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <game_disk_image> <game_filename>\n", os.Args[0])
	}
	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Cannot open file \"%s\" (%v)", filename, err)
	}
	fsys, err := atr.NewAtrFS(file)
	if err != nil {
		log.Fatalf("Cannot open atr image file (%v)", err)
	}

	generals := lib.Generals([2][]lib.General{nil, nil})
	units, err := lib.ReadUnits(fsys, os.Args[2], lib.Crusade, nil, [2][]string{nil, nil}, &generals)
	if err != nil {
		log.Fatalf("Cannot read units (%v)", err)
	}
	firstUnit := true
	for _, sideUnits := range units {
		for _, unit := range sideUnits {
			if firstUnit {
				firstUnit = false
			} else {
				fmt.Print("|")
			}
			objX, objY := unit.Objective.X, unit.Objective.Y
			if !unit.IsInGame {
				objX = unit.HalfDaysUntilAppear
				objY = unit.InvAppearProbability
			}
			ob4 := 0
			if unit.OrderBit4 {
				ob4 = 1
			}
			fmt.Printf("%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d",
				unit.StateByte(),
				unit.XY.X,
				unit.XY.Y,
				unit.MenCount,
				unit.TankCount,
				unit.Formation,
				unit.SupplyUnit,
				unit.VariantBitmap,
				unit.ColorPalette,
				unit.Type,
				unit.NameIndex,
				unit.TargetFormation,
				ob4,
				unit.Order,
				unit.GeneralIndex,
				objX,
				objY,
				unit.SupplyLevel,
				unit.Morale)
		}
	}
}
