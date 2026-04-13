// Package sspak handles the logic for creating and extracting sspak files.
package sspak

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/internal/utils"
)

var (
	// UseZSTD indicates whether to use ZSTD compression for compression.
	// This is set using a CLI flag, and makes the output sspak file incompatible with the official sspak tool.
	UseZSTD bool
)

// File represents a .sspak file, containing paths to the database and assets files.
type File struct {
	DatabaseFile string
	AssetsFile   string
	TempFolder   string // TempFolder is used for processing the files before creating the final .sspak file.
}

// New creates a new File struct with the given name and a temporary path for processing.
func New() *File {
	tempFolder := app.GetTempDir()

	return &File{
		TempFolder: tempFolder,
	}
}

// Open extracts an sspak file to a temporary directory and returns a File struct
// with DatabaseFile and AssetsFile populated for any found entries.
// It respects app.OnlyDB and app.OnlyAssets to skip extracting unneeded files.
func Open(sspakFile string) (*File, error) {
	sspakFile = filepath.Clean(sspakFile)

	inSize, err := utils.CalcSize(sspakFile)
	if err != nil {
		return nil, err
	}

	tempFolder := app.GetTempDir()

	if err := utils.HasEnoughSpace(tempFolder, inSize); err != nil {
		return nil, err
	}

	app.Log(fmt.Sprintf("Opening SSPak archive '%s'", sspakFile))

	if err := extractSSPakContents(sspakFile, tempFolder); err != nil {
		return nil, err
	}

	f := &File{TempFolder: tempFolder}

	for _, name := range []string{"database.sql.gz", "database.sql.zst"} {
		candidate := filepath.Join(tempFolder, name)
		if utils.IsFile(candidate) {
			f.DatabaseFile = candidate
			break
		}
	}

	for _, name := range []string{"assets.tar.gz", "assets.tar.zst"} {
		candidate := filepath.Join(tempFolder, name)
		if utils.IsFile(candidate) {
			f.AssetsFile = candidate
			break
		}
	}

	return f, nil
}

// Extract extracts the raw contents of an sspak file directly into outputDir.
// It respects app.OnlyDB and app.OnlyAssets to skip extracting unneeded files.
func Extract(sspakFile, outputDir string) error {
	sspakFile = filepath.Clean(sspakFile)

	inSize, err := utils.CalcSize(sspakFile)
	if err != nil {
		return err
	}

	if err := utils.HasEnoughSpace(outputDir, inSize); err != nil {
		return err
	}

	app.Log(fmt.Sprintf("Opening SSPak archive '%s'", sspakFile))

	return extractSSPakContents(sspakFile, outputDir)
}

// extractSSPakContents extracts the outer sspak tar into outDir,
// skipping the assets or database file when the OnlyDB/OnlyAssets flags are set.
func extractSSPakContents(sspakFile, outDir string) error {
	r, err := os.Open(filepath.Clean(sspakFile))
	if err != nil {
		return err
	}

	defer func() {
		if err := r.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		isAssets := header.Name == "assets.tar.gz" || header.Name == "assets.tar.zst"
		isDatabase := header.Name == "database.sql.gz" || header.Name == "database.sql.zst"

		if isAssets && app.OnlyDB {
			app.Log(fmt.Sprintf("Skipping extraction of '%s' (--db)", header.Name))
			continue
		}
		if isDatabase && app.OnlyAssets {
			app.Log(fmt.Sprintf("Skipping extraction of '%s' (--assets)", header.Name))
			continue
		}

		target := filepath.Join(outDir, filepath.Clean(header.Name))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(filepath.Clean(target), os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			/* #nosec - file is streamed from sspak archive */
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			outSize, _ := utils.CalcSize(target)
			app.Log(fmt.Sprintf("Extracted '%s' (%s)", target, utils.ByteToHr(outSize)))
		}
	}

	return nil
}

// Write creates the .sspak file with the given name, using the database and assets files specified in the File struct.
// It returns an error if the file could not be created.
func (f *File) Write(fileName string) error {
	if f.AssetsFile == "" && f.DatabaseFile == "" {
		return fmt.Errorf("no database or assets file to include in the .sspak archive")
	}

	fileName = filepath.Clean(fileName)
	outDir := path.Dir(fileName)

	app.Log(fmt.Sprintf("Creating .sspak file '%s'", fileName))

	var inSize int64
	for _, f := range []string{f.DatabaseFile, f.AssetsFile} {
		if f == "" {
			continue
		}
		size, err := utils.CalcSize(f)
		if err != nil {
			return err
		}
		inSize = inSize + size
	}

	if !utils.IsDir(outDir) {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("could not create output directory '%s': %s", outDir, err.Error())
		}
	}

	if err := utils.HasEnoughSpace(outDir, inSize); err != nil {
		return err
	}

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("could not create '%s': %s", fileName, err.Error())
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	tarWriter := tar.NewWriter(file)

	for _, f := range []string{f.DatabaseFile, f.AssetsFile} {
		if f == "" {
			continue
		}
		if err := writeFileToSSPak(f, tarWriter); err != nil {
			_ = tarWriter.Close()
			return fmt.Errorf("could not add '%s' to '%s': %s", f, fileName, err.Error())
		}
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("could not finalise '%s': %s", fileName, err.Error())
	}

	// Size is read after tarWriter.Close() so the tar footer is included.
	outSize, _ := utils.CalcSize(fileName)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", fileName, utils.ByteToHr(outSize)))

	return nil
}

func writeFileToSSPak(fileName string, tarWriter *tar.Writer) error {
	fileName = filepath.Clean(fileName)

	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return fmt.Errorf("could not open '%s': %s", fileName, err.Error())
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("could not get stat for '%s': %s", fileName, err.Error())
	}

	header := &tar.Header{
		Name:    filepath.Base(fileName), // Use the base name of the file in the archive, not the full path.
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("could not write header '%s': %s", fileName, err.Error())
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("could not copy the file '%s' data to archive: %s", fileName, err.Error())
	}

	return nil
}
