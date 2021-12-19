package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/pwiecz/command_series/atr"
	"github.com/pwiecz/command_series/lib"
	"github.com/pwiecz/command_series/ui"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <game_disk_image>\n", os.Args[0])
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
	sprites, err := lib.ReadSprites(fsys)
	if err != nil {
		log.Fatalf("Cannot read sprites from the image file (%v)", err)
	}

	gameData, err := lib.LoadGameData(fsys)
	if err != nil {
		log.Fatalf("Cannot read game data (%v)", err)
	}
	var daytime, night [8]byte
	for i, scenario := range gameData.Scenarios {
		scenarioData, err := lib.LoadScenarioData(fsys, scenario.FilePrefix)
		if err != nil {
			log.Fatalf("Cannot read scenario %s (%v)", scenario.FilePrefix, err)
		}
		if i > 0 {
			if scenarioData.Data.DaytimePalette != daytime {
				log.Fatal("DAYTIME PALETTE DIFFERS")
			}
			if scenarioData.Data.NightPalette != night {
				log.Fatal("NIGHT PALETTE DIFFERS")
			}
		}
		daytime = scenarioData.Data.DaytimePalette
		night = scenarioData.Data.NightPalette
	}
	colors := ui.NewColorSchemes(&daytime, &night)

	basename := filepath.Base(filename)
	prefix := basename[0 : len(basename)-len(filepath.Ext(basename))]
	//	if len(sprites.IntroSprites) > 0 {
	/*		mergedDayIntroSprites, mergedNightIntroSprites := CreateMergedImage(sprites.IntroSprites, colors)
			if err := SaveImageToFile(mergedDayIntroSprites, prefix+"_day_intro.png"); err != nil {
				log.Fatal(err)
			}
			if err := SaveImageToFile(mergedNightIntroSprites, prefix+"_night_intro.png"); err != nil {
				log.Fatal(err)
			}*/
	//	}
	mergedTerrainSprites := CreateMergedImage(sprites.TerrainTiles[:], colors)
	if err := SaveImageToFile(mergedTerrainSprites, prefix+"_terrain.png"); err != nil {
		log.Fatal(err)
	}
	mergedDaylightTerrainSprites := CreateMergedDaytimeImage(sprites.TerrainTiles[:], colors, false)
	if err := SaveImageToFile(mergedDaylightTerrainSprites, prefix+"_terrain_day.png"); err != nil {
		log.Fatal(err)
	}
	mergedNightTerrainSprites := CreateMergedDaytimeImage(sprites.TerrainTiles[:], colors, true)
	if err := SaveImageToFile(mergedNightTerrainSprites, prefix+"_terrain_night.png"); err != nil {
		log.Fatal(err)
	}
	mergedSymbolSprites := CreateMergedImage(sprites.UnitSymbolSprites[:], colors)
	if err := SaveImageToFile(mergedSymbolSprites, prefix+"_symbol.png"); err != nil {
		log.Fatal(err)
	}
	mergedIconSprites := CreateMergedImage(sprites.UnitIconSprites[:], colors)
	if err := SaveImageToFile(mergedIconSprites, prefix+"_icon.png"); err != nil {
		log.Fatal(err)
	}
}

func GetTileNumberMapping(terrainMap *lib.Map) map[byte]byte {
	allTileMap := make(map[byte]struct{})
	for y := 0; y < terrainMap.Height; y++ {
		for x := 0; x < terrainMap.Width; x++ {
			coords := lib.MapCoords{X: x, Y: y}
			if !terrainMap.AreCoordsValid(coords) {
				continue
			}
			tileNr := terrainMap.GetTile(coords)
			allTileMap[tileNr] = struct{}{}
		}
	}
	allTileArr := make([]int, 0, len(allTileMap))
	for tileNr := range allTileMap {
		allTileArr = append(allTileArr, int(tileNr))
	}
	sort.Ints(allTileArr)
	tileNumberMapping := make(map[byte]byte)
	for i, tileNr := range allTileArr {
		tileNumberMapping[byte(tileNr)] = byte(i)
	}
	return tileNumberMapping
}

func CreateMergedImage(images []*image.Paletted, colors *ui.ColorSchemes) image.Image {
	width := images[0].Bounds().Dx()
	height := images[0].Bounds().Dy()
	mergedImage := image.NewNRGBA(image.Rect(0, 0, 8*width, height*len(images)))
	for i, img := range images {
		for j := 0; j < 4; j++ {
			img.Palette = colors.GetBackgroundForegroundColors(byte(j), false)
			draw.Draw(mergedImage, image.Rect(2*j*width, i*height, (2*j+1)*width, (i+1)*height),
				img, image.Pt(0, 0), draw.Over)
			img.Palette = colors.GetBackgroundForegroundColors(byte(j), true)
			draw.Draw(mergedImage, image.Rect((2*j+1)*width, i*height, (2*j+2)*width, (i+1)*height),
				img, image.Pt(0, 0), draw.Over)
		}
	}
	return mergedImage
}

func CreateMergedDaytimeImage(images []*image.Paletted, colors *ui.ColorSchemes, isNight bool) image.Image {
	width := images[0].Bounds().Dx()
	height := images[0].Bounds().Dy()
	mergedImage := image.NewNRGBA(image.Rect(0, 0, 4*width, height*len(images)))
	for i, img := range images {
		for j := 0; j < 4; j++ {
			img.Palette = colors.GetBackgroundForegroundColors(byte(j), isNight)
			draw.Draw(mergedImage, image.Rect(j*width, i*height, (j+1)*width, (i+1)*height),
				img, image.Pt(0, 0), draw.Over)
		}
	}
	return mergedImage
}

func SaveImageToFile(image image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("cannot create \"%s\" file (%v)", filename, err)
	}
	if err := png.Encode(f, image); err != nil {
		f.Close()
		return fmt.Errorf("error encoding image to \"%s\" (%v)", filename, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing \"%s\" file (%v)", filename, err)
	}
	return nil
}
