package utils

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/axllent/ssbak/app"
)

// ExtractSSPak extracts a SSPak (tar) file
func ExtractSSPak(sspakFile, outDir string) error {
	r, err := os.Open(sspakFile)
	if err != nil {
		return err
	}

	defer r.Close()

	if err := MkDirIfNotExists(outDir); err != nil {
		return err
	}

	app.Log(fmt.Sprintf("Opening SSPak archive '%s'", sspakFile))

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break // end of archive
			}
			return err
		}

		if header.Name == "assets.tar.gz" && app.OnlyDB {
			app.Log("Skipping extraction of 'assets.tar.gz' (--only-db)")
			continue
		}
		if header.Name == "database.sql.gz" && app.OnlyAssets {
			app.Log("Skipping extraction of 'database.sql.gz' (--only-assets)")
			continue
		}

		target := filepath.Join(outDir, header.Name)

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()

			outSize, _ := DirSize(target)
			app.Log(fmt.Sprintf("Extracted '%s' (%s)", target, outSize))
		}
	}

	return err
}

// CreateSSPak creates a regular POSIX tar file from a database
// and an assets archive
func CreateSSPak(sspakFile string, files []string) error {
	if len(files) == 0 {
		return errors.New("No files to compress")
	}

	app.Log(fmt.Sprintf("Creating SSPak archive `%s`", sspakFile))
	file, err := os.Create(sspakFile)
	if err != nil {
		return fmt.Errorf("Could not create '%s': %s", sspakFile, err.Error())
	}
	defer file.Close()

	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()

	for _, file := range files {
		if err := addFileToTarWriter(filepath.Base(file), file, tarWriter); err != nil {
			return fmt.Errorf("Could not add '%s' to '%s': %s", file, sspakFile, err.Error())
		}
	}

	outSize, _ := DirSize(sspakFile)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", sspakFile, outSize))

	return nil
}

func addFileToTarWriter(fileName, filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Could not open '%s': %s", filePath, err.Error())
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("Could not get stat for '%s': %s", filePath, err.Error())
	}

	header := &tar.Header{
		Name:    fileName,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("Could not write header '%s': %s", filePath, err.Error())
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("Could not copy the file '%s' data to archive: %s", filePath, err.Error())
	}

	return nil
}
