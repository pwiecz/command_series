package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/pwiecz/command_series/atr"
	"github.com/pwiecz/command_series/lib"
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

	basename := filepath.Base(filename)
	prefix := basename[0 : len(basename)-len(filepath.Ext(basename))]
	if len(sprites.IntroSprites) > 0 {
		mergedIntroSprites := CreateMergedImage(sprites.IntroSprites)
		if err := SaveImageToFile(mergedIntroSprites, prefix+"_intro.png"); err != nil {
			log.Fatal(err)
		}
	}
	mergedTerrainSprites := CreateMergedImage(sprites.TerrainTiles[:])
	if err := SaveImageToFile(mergedTerrainSprites, prefix+"_terrain.png"); err != nil {
		log.Fatal(err)
	}
	mergedSymbolSprites := CreateMergedImage(sprites.UnitSymbolSprites[:])
	if err := SaveImageToFile(mergedSymbolSprites, prefix+"_symbol.png"); err != nil {
		log.Fatal(err)
	}
	mergedIconSprites := CreateMergedImage(sprites.UnitIconSprites[:])
	if err := SaveImageToFile(mergedIconSprites, prefix+"_icon.png"); err != nil {
		log.Fatal(err)
	}
}

func CreateMergedImage(images []*image.Paletted) image.Image {
	width := images[0].Bounds().Dx()
	height := images[0].Bounds().Dy()
	mergedImage := image.NewNRGBA(image.Rect(0, 0, width, height*len(images)))
	for i, img := range images {
		draw.Draw(mergedImage, image.Rect(0, i*height, width, (i+1)*height),
			img, image.Pt(0, 0), draw.Over)
	}
	return mergedImage
}

func SaveImageToFile(image image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Cannot create \"%s\" file (%v)", filename, err)
	}
	if err := png.Encode(f, image); err != nil {
		f.Close()
		return fmt.Errorf("Error encoding image to \"%s\" (%v)", filename, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("Error closing \"%s\" file (%v)", filename, err)
	}
	return nil
}
