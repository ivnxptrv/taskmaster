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

func FileExecutable(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("could not stat file: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("file is not executable: %s", filepath)
	}

	return nil
}
func FileWritable(filepath string) error {
	file, err := os.OpenFile(filepath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("file is not writable: %w", err)
	}
	file.Close()

	return nil
}

func DirExists(dirpath string) error {
	info, err := os.Stat(dirpath)
	if err != nil {
		return fmt.Errorf("directory does not exist or cannot be accessed: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", dirpath)
	}

	return nil
}

func DirExecutable(dirpath string) error {
	info, err := os.Stat(dirpath)
	if err != nil {
		return fmt.Errorf("could not stat dir: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("dir can not be workdir: %s", dirpath)
	}

	return nil
}
