package data

import "bytes"
import "fmt"
import "image"
import "image/color"
import "io"

import "github.com/pwiecz/command_series/atr"

type IconType int

const (
	Cursor           IconType = 0
	LightningBold    IconType = 1
	SupplyTruck      IconType = 8
	FightingUnit     IconType = 9
	UnitOnKnees      IconType = 10
	SurrenderingUnit IconType = 11
	ExclamationMark  IconType = 12
	SmilingFace      IconType = 13
	MovingUnit       IconType = 14
	QuestionMark     IconType = 15
)

type Icons struct {
	// cursor, lightning bolt, 6 x concentric cicles, supply track, fighting unit,
	// unit on knees, surrendering unit, exclamation mark, smiling face, running unit,
	// question mark, 8 x pairs of arrows
	Sprites [24]*image.Paletted
}

func ReadIcons(diskimage atr.SectorReader) (Icons, error) {
	iconsData, err := atr.ReadFile(diskimage, "WAR.PIC")
	if err != nil {
		return Icons{}, fmt.Errorf("Cannot read WAR.PIC file (%v)", err)
	}
	return ParseIcons(bytes.NewReader(iconsData))
}

func ParseIcons(iconsData io.Reader) (Icons, error) {
	icons, err := ParseSpriteData(iconsData, 8, 16, 1, 1, 1)
	if err != nil {
		return Icons{}, err
	}
	if len(icons) != 24 {
		return Icons{}, fmt.Errorf("Unexpected number of icons %d, expected 24", len(icons))
	}
	res := Icons{}
	for i, icon := range icons {
		icon.Palette = []color.Color{color.RGBA{0, 0, 0, 0}, color.RGBA{255, 255, 255, 255}}
		res.Sprites[i] = icon
	}
	return res, nil
}
