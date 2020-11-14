package data

import "fmt"
import "image"
import "io"
import "os"
import "path"

type Icons struct {
	Sprites [24]*image.Paletted
}

func ReadIcons(dirname string) (Icons, error) {
	iconsFilename := path.Join(dirname, "WAR.PIC")
	iconsFile, err := os.Open(iconsFilename)
	if err != nil {
		return Icons{}, fmt.Errorf("Cannot open icon file %s. %v", iconsFilename, err)
	}
	defer iconsFile.Close()
	return ParseIcons(iconsFile)
}

func ParseIcons(iconsData io.Reader) (Icons, error) {
	icons, err := ParseSpriteData(iconsData, 8, 16, 1)
	if err != nil {
		return Icons{}, err
	}
	if len(icons) != 24 {
		return Icons{}, fmt.Errorf("Unexpected number of icons %d, expected 24", len(icons))
	}
	res := Icons{}
	for i, icon := range icons {
		res.Sprites[i] = icon
	}
	return res, nil
}
