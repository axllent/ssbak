package utils

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/axllent/ssbak/app"
)

// ExtractSSPak extracts a SSPak (tar) file
func ExtractSSPak(sspakFile, outDir string) error {
	r, err := os.Open(filepath.Clean(sspakFile))
	if err != nil {
		return err
	}

	defer func() {
		if err := r.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	if err := MkDirIfNotExists(outDir); err != nil {
		return err
	}

	inSize, _ := CalcSize(sspakFile)

	// Test tmp directory has sufficient space.
	if err := HasEnoughSpace(outDir, inSize); err != nil {
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

		target := filepath.Join(outDir, filepath.Clean(header.Name))

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
			f, err := os.OpenFile(filepath.Clean(target), os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			/* #nosec  - file is streamed from targz to file */
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; deferring would cause each file close
			// to wait until all operations have completed.
			if err := f.Close(); err != nil {
				return err
			}

			outSize, _ := CalcSize(target)
			app.Log(fmt.Sprintf("Extracted '%s' (%s)", target, ByteToHr(outSize)))
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

	outDir := path.Dir(sspakFile)
	var inSize int64
	for _, f := range files {
		size, err := CalcSize(f)
		if err != nil {
			return err
		}
		inSize = inSize + size
	}

	// Test output directory has sufficient space.
	if err := HasEnoughSpace(outDir, inSize); err != nil {
		return err
	}

	file, err := os.Create(path.Clean(sspakFile))
	if err != nil {
		return fmt.Errorf("Could not create '%s': %s", sspakFile, err.Error())
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()

	for _, file := range files {
		if err := addFileToTarWriter(filepath.Base(file), file, tarWriter); err != nil {
			return fmt.Errorf("Could not add '%s' to '%s': %s", file, sspakFile, err.Error())
		}
	}

	outSize, _ := CalcSize(sspakFile)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", sspakFile, ByteToHr(outSize)))

	return nil
}

func addFileToTarWriter(fileName, filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("Could not open '%s': %s", filePath, err.Error())
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

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
