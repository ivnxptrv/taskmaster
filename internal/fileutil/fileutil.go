package fileutil

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// FileExists returns nil if path exists and is a regular file (not a directory).
func FileExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("failed to check file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}
	return nil
}

// FileReadable returns nil if the current user can read the file.
func FileReadable(path string) error {
	if err := unix.Access(path, unix.R_OK); err != nil {
		return fmt.Errorf("file not readable: %s: %w", path, err)
	}
	return nil
}

// FileExecutable returns nil if the current user can execute the file.
// Uses access(2) so the check reflects the running user's permissions,
// not just the file's bits.
func FileExecutable(path string) error {
	if err := unix.Access(path, unix.X_OK); err != nil {
		return fmt.Errorf("file not executable: %s: %w", path, err)
	}
	return nil
}

// FileWritable returns nil if the current user can write the file.
func FileWritable(path string) error {
	if err := unix.Access(path, unix.W_OK); err != nil {
		return fmt.Errorf("file not writable: %s: %w", path, err)
	}
	return nil
}

// DirExists returns nil if path exists and is a directory.
func DirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("directory does not exist or cannot be accessed: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}
	return nil
}

// DirExecutable returns nil if the current user can chdir/traverse the dir.
func DirExecutable(path string) error {
	if err := unix.Access(path, unix.X_OK); err != nil {
		return fmt.Errorf("dir not traversable: %s: %w", path, err)
	}
	return nil
}

// OpenOutput opens (or creates) a file for the child's stdout/stderr in append mode.
// Creates parent directories as needed.
func OpenOutput(path string) (*os.File, error) {
	if path == "" || path == "/dev/null" {
		return os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
}
