package sspak

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/axllent/ssbak/app"
	"github.com/klauspost/compress/zstd"
)

// extractAssets extracts a compressed assets archive (tar.gz or tar.zst) into directory.
// The format is detected by the file extension of filePath.
func extractAssets(filePath, directory string) error {
	var err error
	filePath, err = filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	return extractAssetsFromReader(file, strings.HasSuffix(filePath, ".tar.zst"), directory)
}

// extractAssetsFromReader extracts a compressed assets tar archive from r into directory.
// isZSTD selects zstd decompression; otherwise gzip is assumed.
func extractAssetsFromReader(r io.Reader, isZSTD bool, directory string) (err error) {
	directory = stripTrailingSlash(directory)
	directory, err = filepath.Abs(directory)
	if err != nil {
		return err
	}

	undoDir, err := mkdirAll(directory, 0750)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			undoDir()
		}
	}()

	var tarReader *tar.Reader

	if isZSTD {
		zstdDecoder, err := zstd.NewReader(r)
		if err != nil {
			return err
		}
		defer zstdDecoder.Close()
		tarReader = tar.NewReader(zstdDecoder)
	} else {
		gzipReader, err := gzip.NewReader(bufio.NewReader(r))
		if err != nil {
			return err
		}
		defer func() { _ = gzipReader.Close() }()
		tarReader = tar.NewReader(gzipReader)
	}

	// Post extraction directory permissions & timestamps
	type dirInfo struct {
		path   string
		header *tar.Header
	}
	postExtraction := []dirInfo{}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileInfo := header.FileInfo()

		// Tar slip prevention (CWE-22): resolve the full path first, then
		// verify it sits within the extraction directory. String-matching on
		// header.Name alone misses absolute paths, mixed separators, and other
		// bypass tricks — only the resolved path is trustworthy.
		filename := filepath.Join(directory, filepath.FromSlash(header.Name))
		if !strings.HasPrefix(filename+string(os.PathSeparator), filepath.Clean(directory)+string(os.PathSeparator)) {
			continue
		}
		dir := filepath.Dir(filename)

		if skipResampled(filename) {
			continue
		}

		if fileInfo.IsDir() {
			if err := os.MkdirAll(filename, 0750); err != nil {
				return err
			}
			_ = os.Chown(filename, header.Uid, header.Gid) // #nosec
			postExtraction = append(postExtraction, dirInfo{filename, header})
			continue
		}

		// ensure parent directory exists (may not be present in tar)
		if !IsDir(dir) {
			if err := os.MkdirAll(dir, 0750); err != nil {
				return err
			}
		}

		f, err := os.Create(filename) // #nosec
		if err != nil {
			return err
		}

		w := bufio.NewWriter(f)
		buf := make([]byte, 4096)
		for {
			n, readErr := tarReader.Read(buf)
			if n > 0 {
				if _, err := w.Write(buf[:n]); err != nil {
					_ = f.Close()
					return err
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				_ = f.Close()
				return readErr
			}
		}

		if err := w.Flush(); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}

		_ = os.Chmod(filename, os.FileMode(header.Mode))            // #nosec
		_ = os.Chtimes(filename, header.AccessTime, header.ModTime) // #nosec
		_ = os.Chown(filename, header.Uid, header.Gid)              // #nosec
	}

	if len(postExtraction) > 0 {
		app.Log(fmt.Sprintf("Setting timestamps for %d extracted directories", len(postExtraction)))
		for _, d := range postExtraction {
			_ = os.Chtimes(d.path, d.header.AccessTime, d.header.ModTime) // #nosec
			_ = os.Chmod(d.path, d.header.FileInfo().Mode().Perm())       // #nosec
		}
	}

	return nil
}

// mkdirAll creates all directories and returns an undo function that removes
// the first directory created, allowing cleanup on error.
func mkdirAll(dirPath string, perm os.FileMode) (func(), error) {
	var undoDir string

	for p := dirPath; ; p = path.Dir(p) {
		fInfo, err := os.Stat(p)
		if err == nil {
			if fInfo.IsDir() {
				break
			}
			fInfo, err = os.Lstat(p)
			if err != nil {
				return nil, err
			}
			if fInfo.IsDir() {
				break
			}
			return nil, fmt.Errorf("mkdirAll (%s): %v", p, syscall.ENOTDIR)
		}
		if os.IsNotExist(err) {
			undoDir = p
		} else {
			return nil, err
		}
	}

	if undoDir == "" {
		return func() {}, nil
	}

	if err := os.MkdirAll(dirPath, perm); err != nil {
		return nil, err
	}

	return func() {
		if err := os.RemoveAll(undoDir); err != nil {
			panic(err)
		}
	}, nil
}

func stripTrailingSlash(p string) string {
	if len(p) > 0 && p[len(p)-1] == '/' {
		return p[:len(p)-1]
	}
	return p
}

// Read a directory and write it to the tar writer. Recursive function that writes all sub folders.
func tarAddDirectory(directory string, tarWriter *tar.Writer, subPath string) error {
	base, err := os.Stat(directory)
	if err != nil {
		return err
	}

	if base.IsDir() {
		// Add the directory header to tar so we can restore permissions etc
		// calculate the relative path for tar
		evalPath, err := filepath.EvalSymlinks(directory)
		if err != nil {
			return err
		}
		subPath, err := filepath.EvalSymlinks(subPath)
		if err != nil {
			return err
		}

		// relative path
		relativeDirName := evalPath[len(subPath):]

		// inherit directory permissions
		header, err := tar.FileInfoHeader(base, base.Name())
		if err != nil {
			return err
		}

		// set relative directory path
		header.Name = relativeDirName

		// write directory header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
	}

	files, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, file := range files {
		currentPath := filepath.Join(directory, file.Name())
		if file.IsDir() {
			// process contents of directory
			if err := tarAddDirectory(currentPath, tarWriter, subPath); err != nil {
				return err
			}
		} else {
			fi, err := file.Info()
			if err != nil {
				return err
			}
			err = tarAddFile(currentPath, tarWriter, fi, subPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Write path without the prefix in subPath to tar writer.
func tarAddFile(path string, tarWriter *tar.Writer, fileInfo os.FileInfo, subPath string) error {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	evalPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}

	subPath, err = filepath.EvalSymlinks(subPath)
	if err != nil {
		return err
	}

	link := ""
	if evalPath != path {
		link = evalPath
	}

	if skipResampled(path) {
		return nil
	}

	header, err := tar.FileInfoHeader(fileInfo, link)
	if err != nil {
		return err
	}
	header.Name = evalPath[len(subPath):]

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return err
	}

	return err
}

// IsDir returns whether path is an existing directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
