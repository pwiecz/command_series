package atr

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestSectorGeometry(t *testing.T) {
	for _, size := range []int{128, 256} {
		imageSize := 3*128 + 717*size
		image := make([]byte, 16+imageSize)
		image[0], image[1] = ATR_MAGIC1, ATR_MAGIC2
		binary.LittleEndian.PutUint16(image[2:], uint16(imageSize/16))
		binary.LittleEndian.PutUint16(image[4:], uint16(size))
		image[len(image)-1] = 42
		r, err := newAtrSectorReader(bytes.NewReader(image))
		if err != nil {
			t.Fatal(err)
		}
		last, err := r.ReadSector(720)
		if err != nil {
			t.Fatal(err)
		}
		if len(last) != size || last[len(last)-1] != 42 {
			t.Fatal("incorrect final sector")
		}
	}
}

func TestInvalidSectorSize(t *testing.T) {
	header := make([]byte, 16)
	header[0], header[1] = ATR_MAGIC1, ATR_MAGIC2
	if _, err := newAtrSectorReader(bytes.NewReader(header)); err == nil {
		t.Fatal("accepted zero sector size")
	}
}

type testSectors map[int][]byte

func (s testSectors) ReadSector(n int) ([]byte, error) { return s[n], nil }

func TestCyclicFile(t *testing.T) {
	sector := make([]byte, 128)
	sector[126] = 1
	if _, err := readFile(testSectors{1: sector}, &atrFileInfo{start: 1}); err == nil {
		t.Fatal("accepted cyclic chain")
	}
}
