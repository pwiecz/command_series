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
		log.Fatalf("Usage: %s <game_dir>\n", os.Args[0])
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
	if err := ebiten.RunGame(NewGame(flag.Arg(0))); err != nil {
		fmt.Println(err.Error())
	}
}
