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

func GetRootFolder() string {
	assetsFromEnv := os.Getenv("GOCESIUMTILER_WORKDIR")
	if assetsFromEnv != "" {
		return assetsFromEnv
	} else if strings.HasSuffix(os.Args[0], ".test") || strings.HasSuffix(os.Args[0], ".test.exe") {
		_, b, _, _ := runtime.Caller(0)
		return filepath.Dir(filepath.Dir(b))
	} else {
		ex, err := os.Executable()
		if err != nil {
			log.Fatal("cannot retrieve executable directory", err)
		}
		return filepath.Dir(ex)
	}
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