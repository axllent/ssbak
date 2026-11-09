// Package utils provides utility functions for file and directory operations.
package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axllent/ssbak/app"
)

// IsFile returns if a path is a file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}

	return true
}

// IsDir returns if a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	return true
}

// MkDirIfNotExists will create a directory if it doesn't exist
func MkDirIfNotExists(path string) error {
	if !IsDir(path) {
		app.Log(fmt.Sprintf("Creating temporary directory '%s'", path))
		return os.MkdirAll(path, os.ModePerm)
	}

	return nil
}

// CalcSize returns the size of a directory or file. Symbolic links (including
// path itself) are followed, matching how assets are archived. Broken symbolic
// links within a directory are ignored.
func CalcSize(path string) (int64, error) {
	return calcSize(path, map[string]bool{})
}

// calcSize does the work for CalcSize. visited contains the resolved paths of
// the directories currently being walked, to prevent symlink loops.
func calcSize(path string, visited map[string]bool) (int64, error) {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return 0, err
	}

	info, err := os.Stat(realPath)
	if err != nil {
		return 0, err
	}

	if !info.IsDir() {
		return info.Size(), nil
	}

	if visited[realPath] {
		return 0, nil
	}
	visited[realPath] = true
	defer delete(visited, realPath)

	entries, err := os.ReadDir(realPath)
	if err != nil {
		return 0, err
	}

	var size int64
	for _, entry := range entries {
		child := filepath.Join(realPath, entry.Name())

		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			s, err := calcSize(child, visited)
			if err != nil {
				if os.IsNotExist(err) {
					continue // broken symlink
				}
				return size, err
			}
			size += s
			continue
		}

		fi, err := entry.Info()
		if err != nil {
			return size, err
		}
		size += fi.Size()
	}

	return size, nil
}

// ByteToHr returns a human readable size as a string
func ByteToHr(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
