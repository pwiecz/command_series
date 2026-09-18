package lib

import (
	"bytes"
	"testing"
)

func TestUnpackInvalidSize(t *testing.T) {
	if _, err := UnpackFile(bytes.NewReader([]byte{0, 0, 255, 0, 0})); err == nil {
		t.Fatal("accepted negative decoded size")
	}
}
