package lib

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/fs"
)

type IconType int

const (
	Cursor           IconType = 0
	LightningBolt    IconType = 1
	Circles0         IconType = 2
	Circles1         IconType = 3
	Circles2         IconType = 4
	Circles3         IconType = 5
	Circles4         IconType = 6
	Circles5         IconType = 7
	SupplyTruck      IconType = 8
	FightingUnit     IconType = 9
	UnitOnKnees      IconType = 10
	SurrenderingUnit IconType = 11
	ExclamationMark  IconType = 12
	SmilingFace      IconType = 13
	MovingUnit       IconType = 14
	QuestionMark     IconType = 15
	Arrows0          IconType = 16
	Arrows1          IconType = 17
	Arrows2          IconType = 18
	Arrows3          IconType = 19
	Arrows4          IconType = 20
	Arrows5          IconType = 21
	Arrows6          IconType = 22
	Arrows7          IconType = 23
)

// Sequence of pairs of arrows pointing at the center of the tile.
// To be displayed over the objective of a selected unit.
var ArrowIcons = []IconType{
	Arrows0,
	Arrows1,
	Arrows2,
	Arrows3,
	Arrows4,
	Arrows5,
	Arrows6,
	Arrows7}

// Sequence of concentric circles ending with a light bolt.
// To be displayed over the location of a skirmish.
var CircleIcons = []IconType{
	Circles5,
	Circles4,
	Circles3,
	Circles2,
	Circles1,
	Circles0,
	LightningBolt}

type Icons struct {
	Sprites [24]*image.Paletted
}

func ReadIcons(fsys fs.FS) (*Icons, error) {
	iconsData, err := fs.ReadFile(fsys, "WAR.PIC")
	if err != nil {
		return nil, fmt.Errorf("cannot read WAR.PIC file (%v)", err)
	}
	return ParseIcons(bytes.NewReader(iconsData))
}

func ParseIcons(iconsData io.Reader) (*Icons, error) {
	icons, err := ParseSpriteData(iconsData, 8, 16, 1, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(icons) != 24 {
		return nil, fmt.Errorf("unexpected number of icons %d, expected 24", len(icons))
	}
	res := Icons{}
	for i, icon := range icons {
		icon.Palette = []color.Color{color.RGBA{0, 0, 0, 0}, color.RGBA{255, 255, 255, 255}}
		res.Sprites[i] = icon
	}
	return &res, nil
}
