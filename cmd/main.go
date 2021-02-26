package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/pwiecz/command_series/atr"
	"github.com/pwiecz/command_series/ui"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var seed = flag.Int64("seed", 0, "if specified, use given seed to initialize random number generator. Otherwise, a random seed will be used")

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
	source := rand.NewSource(time.Now().UnixNano())
	// Using flag.Visit we can distinguish between flag being set to its default value
	// from flag not being set by the user.
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			source = rand.NewSource(*seed)
		}
	})

	filename := flag.Arg(0)
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Cannot open file or directory %s (%v)", filename, err)
	}
	defer file.Close()
	fileStat, err := file.Stat()
	if err != nil {
		log.Fatalf("Cannot stat file %s (%v)", filename, err)
	}
	var fsys fs.FS
	if fileStat.IsDir() {
		fsys = os.DirFS(filename)
	} else {
		var err error
		fsys, err = atr.NewAtrFS(file)
		if err != nil {
			log.Fatalf("Cannot open atr image file %s (%v)", filename, err)
		}
	}

	ebiten.SetWindowSize(1008, 720)
	ebiten.SetWindowTitle("Command Series Engine")
	game, err := ui.NewGame(fsys, rand.New(source))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := ebiten.RunGame(game); err != nil {
		fmt.Println(err.Error())
	}
}
