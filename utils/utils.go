package utils

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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

// DirSize returns the size of a directory
func DirSize(path string) (string, error) {
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
	return ByteToHr(size), err
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
	src, err := os.Open(file)
	if err != nil {
		return err
	}
	defer src.Close()

	outFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outFile.Close()

	buf := bufio.NewWriter(outFile)
	defer buf.Flush()

	gz := gzip.NewWriter(buf)
	defer gz.Close()

	inSize, _ := DirSize(file)
	app.Log(fmt.Sprintf("Compressing '%s' (%s) to '%s'", file, inSize, output))

	_, err = io.Copy(gz, src)

	outSize, _ := DirSize(output)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", output, outSize))

	return err
}

// Which locates a binary in the current $PATH.
// It will append ".exe" to the filename if the platform is Windows.
func Which(binName string) (string, error) {
	if runtime.GOOS == "windows" {
		// append ".exe" to binary name if Windows
		binName += ".exe"
	}

	return exec.LookPath(binName)
}

// SkipResampled detects whether the assets is a resampled image
func skipResampled(filePath string) bool {
	if !app.IgnoreResampled {
		return false
	}

	for _, r := range app.ResampledRegex {
		if r.MatchString(filePath) {
			return true
		}
	}

	return false
}
