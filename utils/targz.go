package utils

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"syscall"
)

// TarGZCompress creates a archive from the folder inputFilePath points to in the file outputFilePath points to.
// Only adds the last directory in inputFilePath to the archive, not the whole path.
// It tries to create the directory structure outputFilePath contains if it doesn't exist.
// It returns potential errors to be checked or nil if everything works.
func TarGZCompress(inputFilePath, outputFilePath string) (err error) {
	inputFilePath = stripTrailingSlashes(inputFilePath)
	inputFilePath, outputFilePath, err = makeAbsolute(inputFilePath, outputFilePath)
	if err != nil {
		return err
	}
	undoDir, err := mkdirAll(filepath.Dir(outputFilePath), 0775)
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

// TarGZExtract extracts a archive from the file inputFilePath points to in the directory outputFilePath points to.
// It tries to create the directory structure outputFilePath contains if it doesn't exist.
// It returns potential errors to be checked or nil if everything works.
func TarGZExtract(inputFilePath, outputFilePath string) (err error) {
	outputFilePath = stripTrailingSlashes(outputFilePath)
	inputFilePath, outputFilePath, err = makeAbsolute(inputFilePath, outputFilePath)
	if err != nil {
		return err
	}
	undoDir, err := mkdirAll(outputFilePath, 0775)
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

// Creates all directories with os.MakedirAll and returns a function to remove the first created directory so cleanup is possible.
func mkdirAll(dirPath string, perm os.FileMode) (func(), error) {
	var undoDir string

	for p := dirPath; ; p = path.Dir(p) {
		finfo, err := os.Stat(p)

		if err == nil {
			if finfo.IsDir() {
				break
			}

			finfo, err = os.Lstat(p)
			if err != nil {
				return nil, err
			}

			if finfo.IsDir() {
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

	return func() { os.RemoveAll(undoDir) }, nil
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

// The main interaction with tar and gzip. Creates a archive and recursivly adds all files in the directory.
// The finished archive contains just the directory added, not any parents.
// This is possible by giving the whole path exept the final directory in subPath.
func compress(inPath, outFilePath, subPath string) (err error) {
	files, err := ioutil.ReadDir(inPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return errors.New("targz: input directory is empty")
	}

	file, err := os.Create(outFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.Remove(outFilePath)
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

// Read a directy and write it to the tar writer. Recursive function that writes all sub folders.
func writeDirectory(directory string, tarWriter *tar.Writer, subPath string) error {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, file := range files {
		currentPath := filepath.Join(directory, file.Name())
		if file.IsDir() {
			err := writeDirectory(currentPath, tarWriter, subPath)
			if err != nil {
				return err
			}
		} else {
			err = writeTarGz(currentPath, tarWriter, file, subPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Write path without the prefix in subPath to tar writer.
func writeTarGz(path string, tarWriter *tar.Writer, fileInfo os.FileInfo, subPath string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	evaledPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}

	subPath, err = filepath.EvalSymlinks(subPath)
	if err != nil {
		return err
	}

	link := ""
	if evaledPath != path {
		link = evaledPath
	}

	if skipResampled(path) {
		return nil
	}

	header, err := tar.FileInfoHeader(fileInfo, link)
	if err != nil {
		return err
	}
	header.Name = evaledPath[len(subPath):]

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
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileInfo := header.FileInfo()
		dir := filepath.Join(directory, filepath.Dir(header.Name))
		filename := filepath.Join(dir, fileInfo.Name())

		if skipResampled(filename) {
			continue
		}

		err = os.MkdirAll(dir, 0775)
		if err != nil {
			return err
		}

		file, err := os.Create(filename)
		if err != nil {
			return err
		}

		writer := bufio.NewWriter(file)

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

		// set file permissions
		if err := os.Chmod(filename, fileInfo.Mode().Perm()); err != nil {
			return err
		}
	}

	return nil
}