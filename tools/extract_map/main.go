package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"log"
	"os"

	"image"

	"github.com/pwiecz/command_series/atr"
	"github.com/pwiecz/command_series/lib"
	"github.com/pwiecz/command_series/tools/lib/assets"
	"golang.org/x/image/draw"
)

func gameToName(game assets.Game) string {
	if game == assets.Crusade {
		return "crusade"
	} else if game == assets.Decision {
		return "decision"
	} else if game == assets.Conflict {
		return "conflict"
	}
	panic(fmt.Errorf("unknown game %d", game))
}

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

	gameData, err := lib.LoadGameData(fsys)
	if err != nil {
		log.Fatalf("Cannot read game data (%v)", err)
	}
	game := assets.Game(gameData.Game)
	gameName := gameToName(game)

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
	colors := assets.NewColorSchemes(&daytime, &night)

	sprites, err := assets.ReadSprites(fsys)
	if err != nil {
		log.Fatalf("Cannot read sprites (%v)", err)
	}

	for night := 0; night <= 1; night++ {
		var terrainTiles [4][48]image.Image
		isNight := night == 1
		daytimeName := "day"
		if isNight {
			daytimeName = "night"
		}
		for variant := 0; variant < 4; variant++ {
			for tileIx := 0; tileIx < 48; tileIx++ {
				baseImage := sprites.TerrainTiles[tileIx]
				baseImage.Palette = colors.GetBackgroundForegroundColors(byte(variant), isNight)
				coloredTile := image.NewNRGBA(baseImage.Bounds())
				draw.Copy(coloredTile, coloredTile.Bounds().Min, baseImage, baseImage.Bounds(), draw.Over, nil)
				terrainTiles[variant][tileIx] = coloredTile
			}
		}
		mergedImage := CreateMergedImage(terrainTiles[:])
		imagefilename := fmt.Sprintf("%s_terrain_%s.png", gameName, daytimeName)
		if err := SaveImageToFile(mergedImage, imagefilename); err != nil {
			log.Fatal(err)
		}
	}

	terrainMap, err := assets.ReadMap(fsys, game)
	if err != nil {
		log.Fatalf("Cannot read map (%v)", err)
	}

	var terVarMap [48][]byte
	for y := 0; y < terrainMap.Height; y++ {
		for x := 0; x < terrainMap.Width; x++ {
			if !terrainMap.AreCoordsValid(assets.MapCoords{X: x, Y: y}) {
				continue
			}
			tile := terrainMap.GetTile(assets.MapCoords{X: x, Y: y})
			variant := tile / 64
			if terVarMap[tile%64] != nil {
				found := false
				for _, existingVariant := range terVarMap[tile%64] {
					if existingVariant == variant {
						found = true
						break
					}
				}
				if !found {
					terVarMap[tile%64] = append(terVarMap[tile%64], variant)
				}
			} else {
				terVarMap[tile%64] = []byte{variant}
			}
		}
	}
	variants := make(map[int]int)
	for i, vars := range terVarMap {
		for _, var_ := range vars {
			tile := i + int(var_)*64
			newIndex := len(variants)
			variants[tile] = newIndex
		}
	}
	mapArray := make([]int, 0, terrainMap.Width*terrainMap.Height)
	for y := 0; y < terrainMap.Height; y++ {
		for x := 0; x < terrainMap.Width; x++ {
			if !terrainMap.AreCoordsValid(assets.MapCoords{X: x, Y: y}) {
				mapArray = append(mapArray, 0)
				continue
			}
			tile := terrainMap.GetTile(assets.MapCoords{X: x, Y: y})
			tileNr := (tile / 64) + (tile%64)*4
			mapArray = append(mapArray, int(tileNr+1))
		}
	}
	for tile, variants := range terVarMap {
		if len(variants) == 0 {
			continue
		}
		fmt.Print(tile, ": ")
		if len(variants) > 1 {
			fmt.Print(" different variants for tile ", tile, ": ")
			for _, variant := range variants {
				fmt.Print(variant, ", ")
			}
			fmt.Println("")
		} else {
			fmt.Println(variants[0])
		}
	}

	var tiledMap TiledMap
	tiledMap.Height = terrainMap.Height
	tiledMap.Width = terrainMap.Width
	tiledMap.Orientation = Hexagonal
	tiledMap.RenderOrder = RightDown
	tiledMap.StaggerAxis = Y
	tiledMap.StaggerIndex = Odd
	tiledMap.TileWidth = 8
	tiledMap.TileHeight = 8
	tiledMap.HexSideLength = 8
	tiledMap.NextLayerID = 2
	tiledMap.NextObjectID = 1
	tiledMap.Layers = make([]Layer, 1)
	tiledMap.Layers[0].Height = 64
	tiledMap.Layers[0].Width = 64
	tiledMap.Layers[0].ID = 1
	tiledMap.Layers[0].Name = "Map"
	tiledMap.Layers[0].Opacity = 1
	tiledMap.Layers[0].Type = TileLayer
	tiledMap.Layers[0].Visible = true
	tiledMap.Layers[0].Data = mapArray
	tiledMap.TileSets = make([]TileSet, 2)
	tiledMap.TileSets[0].Columns = 4
	tiledMap.TileSets[0].FirstGID = 1
	tiledMap.TileSets[0].ImageHeight = 384
	tiledMap.TileSets[0].ImageWidth = 32
	tiledMap.TileSets[0].Image = fmt.Sprintf("%s_terrain_day.png", gameName)
	tiledMap.TileSets[0].Name = "day"
	tiledMap.TileSets[0].TileCount = 192
	tiledMap.TileSets[0].TileHeight = 8
	tiledMap.TileSets[0].TileWidth = 8
	tiledMap.TileSets[1].Columns = 4
	tiledMap.TileSets[1].FirstGID = 193
	tiledMap.TileSets[1].ImageHeight = 384
	tiledMap.TileSets[1].ImageWidth = 32
	tiledMap.TileSets[1].Image = fmt.Sprintf("%s_terrain_night.png", gameName)
	tiledMap.TileSets[1].Name = "day"
	tiledMap.TileSets[1].TileCount = 192
	tiledMap.TileSets[1].TileHeight = 8
	tiledMap.TileSets[1].TileWidth = 8
	//b, err := json.Marshal(tiledMap)
	b, err := json.MarshalIndent(tiledMap, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	mapFilename := fmt.Sprintf("%s_terrain.json", gameName)
	f, err := os.Create(mapFilename)
	if err != nil {
		log.Fatalf("cannot create \"%s\" file (%v)", mapFilename, err)
	}
	n, err := f.Write(b)
	if n != len(b) {
		log.Fatalf("could not write all the map to file, (%v)", err)
	}
	if err != nil {
		f.Close()
		log.Fatalf("cannot write to file %s (%v)", mapFilename, err)
	}
	f.Close()
}

func CreateMergedImage(images [][48]image.Image) image.Image {
	if len(images) == 0 {
		return image.NewNRGBA(image.Rect(0, 0, 0, 0))
	}
	width := images[0][0].Bounds().Dx()
	height := images[0][0].Bounds().Dy()
	mergedImage := image.NewNRGBA(image.Rect(0, 0, width*len(images), height*48))
	for i, imgs := range images {
		for j, img := range imgs {
			draw.Draw(mergedImage, image.Rect(i*width, j*height, (i+1)*width, (j+1)*height),
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
