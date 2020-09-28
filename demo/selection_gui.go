package main

import "log"
import "os"

import "github.com/hajimehoshi/ebiten"

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <game_dir>\n", os.Args[0])
	}


	ebiten.SetWindowSize(640, 384)
	ebiten.SetWindowTitle("Command Series Engine")
	if err := ebiten.RunGame(NewGame(os.Args[1])); err != nil {
		log.Fatal(err)
	}
}
