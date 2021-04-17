package ui

import (
	"os"
	"path/filepath"
	"strings"
)

func saveDir(scenario string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".command_series", "saves", scenario), nil
}

func listSaveFiles(scenario string) []string {
	saveDir, err := saveDir(scenario)
	if err != nil {
		return nil
	}
	files, err := os.ReadDir(saveDir)
	if err != nil {
		return nil
	}
	var saveFiles []string
	for _, fileInfo := range files {
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".sav") {
			continue
		}
		saveFiles = append(saveFiles, fileInfo.Name())
	}
	return saveFiles
}
