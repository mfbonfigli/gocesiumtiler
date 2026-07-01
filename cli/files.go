package cli

import (
	"os"
	"path/filepath"
	"strings"
)

func findPointCloudFilesInFolder(folder string) ([]string, error) {
	files, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext == ".las" || ext == ".laz" {
			out = append(out, filepath.Join(folder, f.Name()))
		}
	}
	return out, nil
}
