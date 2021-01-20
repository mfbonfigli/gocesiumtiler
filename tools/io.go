package tools

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func OpenFileOrFail(filePath string) *os.File {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}

	return file
}

func GetExecutablePath() string {
	b, _ := os.Getwd()
	return b
}

func CreateDirectoryIfDoesNotExist(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err := os.MkdirAll(directory, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}