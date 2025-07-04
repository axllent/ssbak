package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/axllent/ssbak/app"
)

// TarGZCompress creates a archive from the folder inputFilePath.
// Only adds the last directory in inputFilePath to the archive, not the whole path.
// It tries to create the directory structure outputFilePath contains if it doesn't exist.
// It returns potential errors to be checked or nil if everything works.
func TarGZCompress(inputFilePath, outputFilePath string) (err error) {
	inputFilePath = stripTrailingSlashes(inputFilePath)
	inputFilePath, outputFilePath, err = makeAbsolute(inputFilePath, outputFilePath)
	if err != nil {
		return err
	}
	undoDir, err := mkdirAll(filepath.Dir(outputFilePath), 0750)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			undoDir()
		}
	}()

	err = compress(inputFilePath, outputFilePath, filepath.Dir(inputFilePath))
	if err != nil {
		return err
	}

	return nil
}

// TarGZExtract extracts a archive from the file inputFilePath.
// It tries to create the directory structure outputFilePath contains if it doesn't exist.
// It returns potential errors to be checked or nil if everything works.
func TarGZExtract(inputFilePath, outputFilePath string) (err error) {
	outputFilePath = stripTrailingSlashes(outputFilePath)
	inputFilePath, outputFilePath, err = makeAbsolute(inputFilePath, outputFilePath)
	if err != nil {
		return err
	}
	undoDir, err := mkdirAll(outputFilePath, 0750)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			undoDir()
		}
	}()

	return extract(inputFilePath, outputFilePath)
}

// Creates all directories with os.MkdirAll and returns a function to remove the first created directory so cleanup is possible.
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

// Remove trailing slash if any.
func stripTrailingSlashes(path string) string {
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[0 : len(path)-1]
	}

	return path
}

// Make input and output paths absolute.
func makeAbsolute(inputFilePath, outputFilePath string) (string, string, error) {
	inputFilePath, err := filepath.Abs(inputFilePath)
	if err == nil {
		outputFilePath, err = filepath.Abs(outputFilePath)
	}

	return inputFilePath, outputFilePath, err
}

// The main interaction with tar and gzip. Creates a archive and recursively adds all files in the directory.
// The finished archive contains just the directory added, not any parents.
// This is possible by giving the whole path except the final directory in subPath.
func compress(inPath, outFilePath, subPath string) (err error) {
	files, err := os.ReadDir(inPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return errors.New("targz: input directory is empty")
	}

	file, err := os.Create(path.Clean(outFilePath))
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if err := os.Remove(outFilePath); err != nil {
				panic(err)
			}
		}
	}()

	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	err = writeDirectory(inPath, tarWriter, subPath)
	if err != nil {
		return err
	}

	err = tarWriter.Close()
	if err != nil {
		return err
	}

	err = gzipWriter.Close()
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	return nil
}

// Read a directory and write it to the tar writer. Recursive function that writes all sub folders.
func writeDirectory(directory string, tarWriter *tar.Writer, subPath string) error {
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
			if err := writeDirectory(currentPath, tarWriter, subPath); err != nil {
				return err
			}
		} else {
			fi, err := file.Info()
			if err != nil {
				return err
			}
			err = writeTarGz(currentPath, tarWriter, fi, subPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Write path without the prefix in subPath to tar writer.
func writeTarGz(path string, tarWriter *tar.Writer, fileInfo os.FileInfo, subPath string) error {
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

// Extract the file in filePath to directory.
func extract(filePath string, directory string) error {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	gzipReader, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		return err
	}
	defer func() { _ = gzipReader.Close() }()

	tarReader := tar.NewReader(gzipReader)

	// Post extraction directory permissions & timestamps
	type DirInfo struct {
		Path   string
		Header *tar.Header
	}

	// slice to add all extracted directory info for post-processing
	postExtraction := []DirInfo{}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileInfo := header.FileInfo()

		// skip any file that contains a `..` (eg: `../file`) - CWE-22
		if strings.Contains(fileInfo.Name(), "..") {
			continue
		}

		dir := filepath.Join(directory, filepath.Dir(header.Name))
		filename := filepath.Join(dir, path.Clean(fileInfo.Name()))

		if skipResampled(filename) {
			continue
		}

		if fileInfo.IsDir() {
			// create the directory 755 in case writing permissions prohibit writing before files added
			if err := os.MkdirAll(filename, 0750); err != nil {
				return err
			}

			// set file ownership (if allowed)
			// Chtimes() && Chmod() only set after once extraction is complete
			_ = os.Chown(filename, header.Uid, header.Gid) // #nosec

			// add directory info to slice to process afterwards
			postExtraction = append(postExtraction, DirInfo{filename, header})
			continue
		}

		// make sure parent directory exists (may not be included in tar)
		if !fileInfo.IsDir() && !IsDir(dir) {
			err = os.MkdirAll(dir, 0750)
			if err != nil {
				return err
			}
		}

		file, err := os.Create(filename) // #nosec
		if err != nil {
			return err
		}

		writer := bufio.NewWriter(file) // #nosec

		buffer := make([]byte, 4096)
		for {
			n, err := tarReader.Read(buffer)
			if err != nil && err != io.EOF {
				panic(err)
			}
			if n == 0 {
				break
			}

			_, err = writer.Write(buffer[:n])
			if err != nil {
				return err
			}
		}

		err = writer.Flush()
		if err != nil {
			return err
		}

		err = file.Close()
		if err != nil {
			return err
		}

		// set file permissions, timestamps & uid/gid
		_ = os.Chmod(filename, os.FileMode(header.Mode))            // #nosec
		_ = os.Chtimes(filename, header.AccessTime, header.ModTime) // #nosec
		_ = os.Chown(filename, header.Uid, header.Gid)              // #nosec
	}

	if len(postExtraction) > 0 {
		// update directory timestamps & permissions once extraction is complete
		app.Log(fmt.Sprintf("Setting timestamps for %d extracted directories", len(postExtraction)))

		for _, dir := range postExtraction {
			_ = os.Chtimes(dir.Path, dir.Header.AccessTime, dir.Header.ModTime) // #nosec
			_ = os.Chmod(dir.Path, dir.Header.FileInfo().Mode().Perm())         // #nosec
		}
	}

	return nil
}
