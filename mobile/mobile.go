package mobile

import (
	"bytes"
	_ "embed"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/pwiecz/command_series/atr"
	"github.com/pwiecz/command_series/ui"
)

//go:embed crusade.atr
var crusadeAtr []byte

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	fsys, err := atr.NewAtrFS(bytes.NewReader(crusadeAtr))
	if err != nil {
		panic(err)
	}
	game, err := ui.NewGame(fsys, rand.New(source))
	if err != nil {
		panic(err)
	}
	mobile.SetGame(game)
}

// Dummy is a dummy exported function.
//
// gomobile doesn't compile a package that doesn't include any exported function.
// Dummy forces gomobile to compile this package.
func Dummy() {}
