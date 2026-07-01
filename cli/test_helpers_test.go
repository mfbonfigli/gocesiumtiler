package cli

import "os"

func touchFile(name string) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	return file.Close()
}
