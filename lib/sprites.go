package lib

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/fs"
)

type Font struct {
	fallback   *image.Paletted
	characters map[rune]*image.Paletted
}

func (f *Font) Size() image.Point {
	return f.fallback.Bounds().Size()
}

func (f *Font) Glyph(r rune) *image.Paletted {
	if c, ok := f.characters[r]; ok {
		return c
	}
	return f.fallback
}

type Sprites struct {
	GameFont          *Font
	IntroFont         *Font
	IntroSprites      []*image.Paletted
	TerrainTiles      [48]*image.Paletted
	UnitSymbolSprites [16]*image.Paletted
	UnitIconSprites   [16]*image.Paletted
}

func ReadSprites(fsys fs.FS) (*Sprites, error) {
	iconSpritesData, err := fs.ReadFile(fsys, "CRUSADEI.FNT")
	if err != nil {
		return nil, fmt.Errorf("cannot read CRUSADEI.FNT file (%v)", err)
	}
	symbolSpritesData, err := fs.ReadFile(fsys, "CRUSADES.FNT")
	if err != nil {
		return nil, fmt.Errorf("cannot read CRUSADES.FNT file (%v)", err)
	}
	introSpritesData, err := fs.ReadFile(fsys, "FLAG.FNT")
	if err != nil {
		return nil, fmt.Errorf("cannot read FLAG.FNT file (%v)", err)
	}
	return ParseSprites(bytes.NewReader(iconSpritesData),
		bytes.NewReader(symbolSpritesData),
		bytes.NewReader(introSpritesData))
}

func ParseSpriteData(data io.Reader, width, height, scaleX, scaleY, bits int) ([]*image.Paletted, error) {
	var sprites []*image.Paletted
	if bits != 1 && bits != 2 && bits != 4 && bits != 8 {
		return sprites, fmt.Errorf("unsupported sprite bit depth %d", bits)
	}
	if scaleX < 1 {
		return sprites, fmt.Errorf("unsupported scaleX %d", scaleX)
	}
	if scaleY < 1 {
		return sprites, fmt.Errorf("unsupported scaleY %d", scaleY)
	}
	palette := make([]color.Color, 1<<bits)
	for i := 0; i < len(palette); i++ {
		palette[len(palette)-1-i] = RGBPalette[i*15/(len(palette)-1)]
	}
	bytesPerSprite := (width*height*bits + 7) / 8
	spriteData := make([]byte, bytesPerSprite)
	for {
		_, err := io.ReadFull(data, spriteData)
		if err == io.EOF {
			return sprites, nil
		}
		if err != nil && err != io.EOF {
			return sprites, err
		}
		sprite := image.NewPaletted(image.Rect(0, 0, width*scaleX, height*scaleY), palette)
		for y := 0; y < height*scaleY; y++ {
			for x := 0; x < width*scaleX; x++ {
				pixelNum := y/scaleY*width + x/scaleX
				pixelByte := spriteData[pixelNum*bits/8]
				byteChunkNum := pixelNum % (8 / bits)
				pixelByte = (pixelByte << byte(byteChunkNum*bits)) >> (byteChunkNum * bits)
				pixelByte >>= 8 - bits - byteChunkNum*bits
				sprite.SetColorIndex(x, y, pixelByte) //color.Gray{255 - (pixelByte*255)/byte(bits)})
			}
		}
		sprites = append(sprites, sprite)
	}
}

func ParseSprites(iconData, symbolData, introData io.Reader) (*Sprites, error) {
	iconSprites, err := ParseSpriteData(iconData, 8, 8, 1, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(iconSprites) != 128 {
		return nil, fmt.Errorf("too few icon sprites read. Expected 128, read %d",
			len(iconSprites))
	}
	symbolSprites, err := ParseSpriteData(symbolData, 8, 8, 1, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(symbolSprites) != 128 {
		return nil, fmt.Errorf("too few symbol sprites read. Expected 128, read %d",
			len(symbolSprites))
	}
	introSprites, err := ParseSpriteData(introData, 4, 8, 2, 1, 2)
	if err != nil {
		return nil, err
	}
	if len(introSprites) != 128 {
		return nil, fmt.Errorf("too few intro sprites read. Expected 128, read %d",
			len(introSprites))
	}
	chars := []rune{
		' ', '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/',
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ':', ';', '<', '=', '>', '?', '@',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P',
		'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
		1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009,
		1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019,
		1020, 1021, 1022, 1023, 1024, 1025, 1026, 1027, 1028, 1029,
		1030, 1031, 1032, 1033, 1034, 1035, 1036, 1037,
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p',
		'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	}
	characters := make(map[rune]*image.Paletted)
	introCharacters := make(map[rune]*image.Paletted)
	for i, char := range chars {
		if char == 0 {
			continue
		}
		if i < 59 {
			characters[char] = iconSprites[i]
		}
		introCharacters[char] = introSprites[i]
	}
	sprites := &Sprites{
		IntroSprites: introSprites,
		IntroFont: &Font{
			characters: introCharacters,
			fallback:   introCharacters['?']},
		GameFont: &Font{
			characters: characters,
			fallback:   characters['?']},
	}
	copy(sprites.TerrainTiles[:], iconSprites[64:])
	copy(sprites.UnitIconSprites[:], iconSprites[112:])
	copy(sprites.UnitSymbolSprites[:], symbolSprites[112:])
	return sprites, nil
}
