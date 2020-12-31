package tools

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/tiler"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type IFileFinder interface {
	GetLasFilesToProcess(opts *tiler.TilerOptions) []string
}

type FileFinder struct {}

func NewFileFinder() IFileFinder {
	return &FileFinder{}
}

func (fileFinder *FileFinder) GetLasFilesToProcess(opts *tiler.TilerOptions) []string {
	// If folder processing is not enabled then las file is given by -input flag, otherwise look for las in -input folder
	// eventually excluding nested folders if Recursive flag is disabled
	if !opts.FolderProcessing {
		return []string{opts.Input}
	}

	return fileFinder.getLasFilesFromInputFolder(opts)
}

func (fileFinder *FileFinder) getLasFilesFromInputFolder(opts *tiler.TilerOptions) []string {
	var lasFiles = make([]string, 0)

	baseInfo, _ := os.Stat(opts.Input)
	err := filepath.Walk(
		opts.Input,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() && !opts.Recursive && !os.SameFile(info, baseInfo) {
				return filepath.SkipDir
			} else {
				if strings.ToLower(filepath.Ext(info.Name())) == ".las" {
					lasFiles = append(lasFiles, path)
				}
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	return lasFiles
}

