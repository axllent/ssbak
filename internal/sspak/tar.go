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

		// Strip the archive's top-level directory (normally "assets/") so the
		// contents are extracted directly into directory.
		relPath, isTopLevel := assetRelPath(header.Name)
		if relPath == "" || (isTopLevel && !fileInfo.IsDir()) {
			continue
		}

		// Tar slip prevention (CWE-22): resolve the full path first, then
		// verify it sits within the extraction directory. String-matching on
		// header.Name alone misses absolute paths, mixed separators, and other
		// bypass tricks, so only the resolved path is trustworthy.
		filename := filepath.Join(directory, filepath.FromSlash(relPath))
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

		if _, err := io.Copy(f, tarReader); err != nil {
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

// assetRelPath strips the top-level directory from a tar entry name, returning
// the path relative to the assets directory. Archives normally contain a single
// top-level "assets" directory, however it may have a different name (eg: when
// the assets directory was a symlink to a differently named directory when saved).
// isTopLevel is true when the entry is the top-level entry itself, in which case
// "." is returned. An empty string is returned for entries that should be ignored.
func assetRelPath(name string) (relPath string, isTopLevel bool) {
	// cleaning against "/" removes any "./" prefixes and "../" traversal
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" {
		return "", false
	}

	i := strings.IndexByte(name, '/')
	if i == -1 {
		return ".", true
	}

	return name[i+1:], false
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
			fmt.Fprintf(os.Stderr, "error cleaning up %q: %v\n", undoDir, err)
		}
	}, nil
}

func stripTrailingSlash(p string) string {
	if len(p) > 0 && p[len(p)-1] == '/' {
		return p[:len(p)-1]
	}
	return p
}

// assetsArchiveRoot is the top-level directory name of every assets archive,
// regardless of the name of the directory the assets were read from.
const assetsArchiveRoot = "assets"

// tarAddDirectory recursively writes directory and its contents to the tar writer,
// storing them under archiveName (eg: "assets") in the archive. Symbolic links
// are followed, so linked files & directories are stored as their contents.
func tarAddDirectory(directory string, tarWriter *tar.Writer, archiveName string) error {
	return tarAddDirectoryRecursive(directory, tarWriter, archiveName, map[string]bool{})
}

// tarAddDirectoryRecursive does the work for tarAddDirectory. visited contains the
// resolved paths of the directories currently being walked, to prevent symlink loops.
func tarAddDirectoryRecursive(directory string, tarWriter *tar.Writer, archiveName string, visited map[string]bool) error {
	realDir, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return err
	}
	if visited[realDir] {
		app.Log(fmt.Sprintf("Skipping '%s': symbolic link loop", directory))
		return nil
	}
	visited[realDir] = true
	defer delete(visited, realDir)

	base, err := os.Stat(directory)
	if err != nil {
		return err
	}

	// Add the directory header to tar so we can restore permissions etc
	header, err := tar.FileInfoHeader(base, "")
	if err != nil {
		return err
	}
	header.Name = archiveName + "/"

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	files, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, file := range files {
		currentPath := filepath.Join(directory, file.Name())
		currentName := archiveName + "/" + file.Name()

		// os.Stat follows symbolic links
		fi, err := os.Stat(currentPath)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			if err := tarAddDirectoryRecursive(currentPath, tarWriter, currentName, visited); err != nil {
				return err
			}
			continue
		}

		if !fi.Mode().IsRegular() {
			app.Log(fmt.Sprintf("Skipping '%s': not a regular file", currentPath))
			continue
		}

		if err := tarAddFile(currentPath, tarWriter, fi, currentName); err != nil {
			return err
		}
	}

	return nil
}

// tarAddFile writes the file at filePath to the tar writer as archiveName.
func tarAddFile(filePath string, tarWriter *tar.Writer, fileInfo os.FileInfo, archiveName string) error {
	if skipResampled(archiveName) {
		return nil
	}

	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	header, err := tar.FileInfoHeader(fileInfo, "")
	if err != nil {
		return err
	}
	header.Name = archiveName

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)

	return err
}

// IsDir returns whether path is an existing directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
