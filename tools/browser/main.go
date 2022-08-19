package main

import (
	"fmt"
	"io/fs"
	"os"
	"runtime"

	"github.com/pwiecz/command_series/atr"
	"github.com/pwiecz/command_series/lib"
	"github.com/pwiecz/go-fltk"
)

type MainWindow struct {
	*fltk.Window
	mapWindow    *MapWindow
	infoTable    *InfoTable
	gameData     *lib.GameData
	scenarioData *lib.ScenarioData
}

func NewMainWindow() *MainWindow {
	w := &MainWindow{}
	w.Window = fltk.NewWindow(1600, 900)
	w.Begin()

	mainPack := fltk.NewPack(0, 0, 1600, 900)
	mainPack.SetType(fltk.VERTICAL)

	menuBar := fltk.NewMenuBar(0, 0, 1600, 30)
	menuBar.AddEx("&File/&Load", fltk.CTRL+int('o'), w.onLoadPressed, 0)

	pack := fltk.NewPack(0, 0, 1600, 870)
	pack.SetType(fltk.HORIZONTAL)

	w.mapWindow = NewMapWindow(0, 0, 900, 780)

	rightPack := fltk.NewPack(0, 0, 700, 870)
	rightPack.SetType(fltk.VERTICAL)

	infoTablePack := fltk.NewPack(0, 0, 700, 590)
	infoTablePack.SetType(fltk.HORIZONTAL)
	w.infoTable = NewInfoTable(0, 0, 700, 590)
	dummyGroup := fltk.NewGroup(0, 0, 0, 590)
	infoTablePack.End()
	infoTablePack.Resizable(dummyGroup)

	rightPack.End()
	rightPack.Resizable(infoTablePack)

	pack.End()
	pack.Resizable(w.mapWindow)

	mainPack.End()
	mainPack.Resizable(pack)

	w.End()
	w.Resizable(mainPack)
	w.SetCallback(func() {
		if fltk.EventType() == fltk.SHORTCUT && fltk.EventKey() == fltk.ESCAPE {
			// Don't close the main window when user just presses Escape.
			return
		}
		w.Hide()
	})
	return w
}

func (w *MainWindow) onLoadPressed() {
	fileChooser := fltk.NewFileChooser("", "Atari images (*.atr)", fltk.FileChooser_SINGLE, "Select image file")
	fileChooser.SetPreview(false)
	defer fileChooser.Destroy()
	fileChooser.Popup()
	selectedFilenames := fileChooser.Selection()
	if len(selectedFilenames) != 1 {
		return
	}
	filename := selectedFilenames[0]
	gameData, err := loadGameFromFileOrDir(filename)
	if err != nil {
		fltk.MessageBox("Error loading", err.Error())
		w.gameData = nil
		return
	}
	if len(gameData.Scenarios) == 0 {
		fltk.MessageBox("No scenarios in the game file", err.Error())
		w.gameData = nil
		return
	}
	w.gameData = gameData

	selectedScenario := 0

	scenarioChoiceDialog := fltk.NewWindow(300, 300)
	scenarioChoiceDialog.SetLabel("Choose scenario")
	scenarioChoiceDialog.SetModal()
	mainPack := fltk.NewPack(0, 0, 300, 300)
	mainPack.SetType(fltk.VERTICAL)
	scenarios := fltk.NewChoice(0, 0, 250, 30, "Scenarios:")
	for _, scenario := range gameData.Scenarios {
		scenarios.Add(scenario.Name, func() {})
	}
	scenarios.SetValue(0)
	buttonPack := fltk.NewPack(0, 0, 250, 30)
	buttonPack.SetType(fltk.HORIZONTAL)
	buttonPack.SetSpacing(5)
	ok := fltk.NewButton(0, 0, 100, 30, "Ok")
	ok.SetCallback(func() {
		selectedScenario = scenarios.Value()
		scenarioChoiceDialog.Destroy()
	})
	scenarioChoiceDialog.SetCallback(func() {})
	scenarioChoiceDialog.Show()
	for scenarioChoiceDialog.IsShown() {
		fltk.Wait()
	}

	scenarioData, err := loadScenarioFromFileOrDir(filename, gameData.Scenarios[selectedScenario].FilePrefix)
	if err != nil {
		fltk.MessageBox("Error loading scenario", err.Error())
		return
	}
	w.scenarioData = scenarioData

	w.mapWindow.SetGameData(gameData, scenarioData)
}

func loadGameFromFileOrDir(filename string) (*lib.GameData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("couldn't open %s, %v", filename, err)
	}
	fileStat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("couldn't stat %s, %v", filename, err)
	}
	var fsys fs.FS
	if fileStat.IsDir() {
		fsys = os.DirFS(filename)
	} else {
		var err error
		fsys, err = atr.NewAtrFS(file)
		if err != nil {
			return nil, fmt.Errorf("couldn't load atr image file %s, %v", filename, err)
		}
	}
	gameData, err := lib.LoadGameData(fsys)
	if err != nil {
		return nil, fmt.Errorf("couldn't load game data from %s, %v", filename, err)
	}
	return gameData, nil
}

func loadScenarioFromFileOrDir(filename, filePrefix string) (*lib.ScenarioData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("couldn't open %s, %v", filename, err)
	}
	fileStat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("couldn't stat %s, %v", filename, err)
	}
	var fsys fs.FS
	if fileStat.IsDir() {
		fsys = os.DirFS(filename)
	} else {
		var err error
		fsys, err = atr.NewAtrFS(file)
		if err != nil {
			return nil, fmt.Errorf("couldn't load atr image file %s, %v", filename, err)
		}
	}
	scenarioData, err := lib.LoadScenarioData(fsys, filePrefix)
	if err != nil {
		return nil, fmt.Errorf("couldn't load scenario data from %s, %v", filename, err)
	}
	return scenarioData, nil
}

func main() {
	runtime.LockOSThread()

	for i := 0; i < fltk.ScreenCount(); i++ {
		fltk.SetScreenScale(i, 1.0)
	}
	fltk.SetKeyboardScreenScaling(false)
	w := NewMainWindow()
	fltk.Lock()
	w.Show()

	fltk.Run()

	w.mapWindow.Destroy()
	w.Destroy()
}
