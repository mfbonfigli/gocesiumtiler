package tools

import (
	"log"
	"os"
	"path/filepath"
)

func OpenFileOrFail(filePath string) *os.File {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return file
}

func GetExecutablePath() string {
	//Executable path
	ex, err := os.Executable()
	if err != nil {
		log.Fatal("cannot retrieve executable directory", err)
	}

	return filepath.Dir(ex)
}