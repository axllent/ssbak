package utils

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/axllent/ssbak/app"
)

// IsFile returns if a path is a file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.Mode().IsRegular() {
		return false
	}

	return true
}

// IsDir returns if a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.IsDir() {
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

// CalcSize returns the size of a directory or file
func CalcSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
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

// GzipFile will compress an existing file with gzip and save it was output
func GzipFile(file, output string) error {
	src, err := os.Open(filepath.Clean(file))
	if err != nil {
		return err
	}

	defer func() {
		if err := src.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	outFile, err := os.Create(path.Clean(output))
	if err != nil {
		return err
	}

	defer func() {
		if err := outFile.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	buf := bufio.NewWriter(outFile)
	defer func() { _ = buf.Flush() }()

	gz := gzip.NewWriter(buf)
	defer func() { _ = gz.Close() }()

	inSize, _ := CalcSize(file)
	app.Log(fmt.Sprintf("Compressing '%s' (%s) to '%s'", file, ByteToHr(inSize), output))

	_, err = io.Copy(gz, src)

	outSize, _ := CalcSize(output)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", output, ByteToHr(outSize)))

	return err
}

// SkipResampled detects whether the assets is a resampled image
func skipResampled(filePath string) bool {
	if !app.IgnoreResampled {
		return false
	}

	for _, r := range app.ResampledRegex {
		// Silverstripe 5 generates thumbnails for CMS previews by default with `__FitMaxWzM1MiwyNjRd`
		if !strings.Contains(filePath, "__FitMaxWzM1MiwyNjRd.") && r.MatchString(filePath) {
			return true
		}
	}

	return false
}
