package fileutil

import (
	"fmt"
	"os"
)

// and not Dir
func FileExists(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filepath)
		}
		// Return other potential errors (e.g., permission denied)
		return fmt.Errorf("failed to check file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filepath)
	}
	return nil
}

func FileReadable(filepath string) error {
	file, err := os.OpenFile(filepath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("cannot read file: %s (permission denied)", filepath)
	}
	file.Close()
	return nil
}
