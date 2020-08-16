package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// GetTempDir will create & return a temporary directory if one has not been specified
func GetTempDir() string {
	if TempDir == "" {
		randBytes := make([]byte, 6)
		rand.Read(randBytes)
		TempDir = filepath.Join(os.TempDir(), "ssbak-"+hex.EncodeToString(randBytes))
		AddTempFile(TempDir)
	}
	if err := mkDirIfNotExists(TempDir); err != nil {
		// need a better way to exit
		fmt.Printf("Error: %v", err)
		os.Exit(2)
	}

	return TempDir
}

// AddTempFile adds a file to the temporary files to clean up
func AddTempFile(file string) {
	TempFiles = append(TempFiles, file)
}

// Cleanup removes temporary files & directories on exit
func Cleanup() {
	for _, file := range TempFiles {
		if isFile(file) {
			os.Remove(file)
		} else if isDir(file) {
			os.RemoveAll(file)
		}
	}
}

// Log will print out data in verbose output
func Log(msg string) {
	if Verbose {
		log.Println(msg)
	}
}

// MkDirIfNotExists will create a directory if it doesn't exist
func mkDirIfNotExists(path string) error {
	if !isDir(path) {
		Log(fmt.Sprintf("Creating temporary directory '%s'", path))
		return os.MkdirAll(path, os.ModePerm)
	}

	return nil
}

// IsFile returns if a path is a file
func isFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.Mode().IsRegular() {
		return false
	}

	return true
}

// IsDir returns if a path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.IsDir() {
		return false
	}

	return true
}
