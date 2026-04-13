package sspak

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/internal/utils"
	"github.com/klauspost/compress/zstd"
)

// AddAssets adds the assets file to the File struct, given the path to the assets directory. It returns an error if the assets file could not be created.
func (f *File) AddAssets(assetsDir string) error {
	app.Log(fmt.Sprintf("Calculating size of '%s'", assetsDir))

	size, _ := utils.CalcSize(assetsDir)

	if err := utils.HasEnoughSpace(f.TempFolder, size); err != nil {
		return err
	}

	if UseZSTD {
		f.AssetsFile = filepath.Join(f.TempFolder, "assets.tar.zst")
	} else {
		f.AssetsFile = filepath.Join(f.TempFolder, "assets.tar.gz")
	}

	app.Log(fmt.Sprintf("Compressing '%s' (%s) to '%s'", assetsDir, utils.ByteToHr(size), f.AssetsFile))

	if app.IgnoreResampled {
		app.Log("Ignoring resampled images")
	}

	var err error
	assetsDir, err = filepath.Abs(assetsDir)
	if err != nil {
		return err
	}

	// create the assets archive
	files, err := os.ReadDir(assetsDir)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return errors.New("compress: input directory is empty")
	}

	file, err := os.Create(f.AssetsFile)
	if err != nil {
		return err
	}

	var (
		tarWriter  *tar.Writer
		zstdWriter *zstd.Encoder
		gzipWriter *gzip.Writer
	)

	if UseZSTD {
		zstdWriter, err = zstd.NewWriter(file)
		if err != nil {
			_ = file.Close()
			return err
		}
		tarWriter = tar.NewWriter(zstdWriter)
	} else {
		gzipWriter = gzip.NewWriter(file)
		tarWriter = tar.NewWriter(gzipWriter)
	}

	err = tarAddDirectory(assetsDir, tarWriter, filepath.Dir(assetsDir))
	if err != nil {
		tarWriter.Close()
		if zstdWriter != nil {
			zstdWriter.Close()
		}
		if gzipWriter != nil {
			gzipWriter.Close()
		}
		file.Close()
		return err
	}

	// Close tarWriter first to ensure all data is flushed to the underlying writer before closing it.
	if err = tarWriter.Close(); err != nil {
		if zstdWriter != nil {
			_ = zstdWriter.Close()
		}
		if gzipWriter != nil {
			_ = gzipWriter.Close()
		}
		file.Close()

		return err
	}

	if zstdWriter != nil {
		if err = zstdWriter.Close(); err != nil {
			file.Close()
			return err
		}
	}
	if gzipWriter != nil {
		if err = gzipWriter.Close(); err != nil {
			file.Close()
			return err
		}
	}

	return file.Close()
}

// LoadAssets extracts the assets archive from f.AssetsFile into assetsBase.
// Any existing assets directory is renamed to assets.old and scheduled for cleanup.
// It supports both tar.gz and tar.zst formats.
func (f *File) LoadAssets(assetsBase string) error {
	inSize, _ := utils.CalcSize(f.AssetsFile)

	if assetsBase == "" {
		assetsBase = "."
	}

	if err := utils.HasEnoughSpace(assetsBase, inSize); err != nil {
		return err
	}

	assetsPath := filepath.Join(assetsBase, "assets")
	if IsDir(assetsPath) {
		app.Log(fmt.Sprintf("Renaming existing '%s' to '%s.old'", assetsPath, assetsPath))
		if err := os.Rename(assetsPath, assetsPath+".old"); err != nil {
			return err
		}
		app.AddTempFile(assetsPath + ".old")
	}

	app.Log(fmt.Sprintf("Unpacking '%s' to '%s'", f.AssetsFile, assetsPath))

	if app.IgnoreResampled {
		app.Log("Ignoring resampled images")
	}

	if err := extractAssets(f.AssetsFile, assetsBase); err != nil {
		return err
	}

	outSize, _ := utils.CalcSize(assetsPath)
	app.Log(fmt.Sprintf("Restored '%s' (%s)", assetsPath, utils.ByteToHr(outSize)))

	return nil
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
