package main

import "flag"
import "fmt"
import "log"
import "os"
import "runtime/pprof"

import "github.com/hajimehoshi/ebiten"

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatalf("Usage: %s <game_disk_image>\n", os.Args[0])
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ebiten.SetWindowSize(1008, 720)
	ebiten.SetWindowTitle("Command Series Engine")
	game, err := NewGame(flag.Arg(0))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := ebiten.RunGame(game); err != nil {
		fmt.Println(err.Error())
	}
}
