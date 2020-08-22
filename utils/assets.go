package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axllent/ssbak/app"
)

// AssetsToTarGz creates a tar.gz from the assets folder
func AssetsToTarGz(assetsDir, gzipFile string) error {
	app.Log(fmt.Sprintf("Calculating size of '%s'", assetsDir))

	size, _ := DirSize(assetsDir)
	app.Log(fmt.Sprintf("Compressing '%s' (%s) to '%s'", assetsDir, size, gzipFile))

	if app.IgnoreResampled {
		app.Log("Ignoring resampled images")
	}

	err := TarGZCompress(assetsDir, gzipFile)

	outSize, _ := DirSize(gzipFile)
	app.Log(fmt.Sprintf("Compressed '%s' (%s)", gzipFile, outSize))

	return err
}

// AssetsFromTarGz extracts assets from a tar.gz. If an existing assets directory is found
// then the existing one is renamed assets.old, and then deleted after the process completes.
func AssetsFromTarGz(tmpDir, assetsBase string) error {
	assetsPath := filepath.Join(assetsBase, "assets")
	if IsDir(assetsPath) {
		app.Log(fmt.Sprintf("Renaming existing '%s' to '%s.old'", assetsPath, assetsPath))
		if err := os.Rename(assetsPath, assetsPath+".old"); err != nil {
			return err
		}
	}

	in := filepath.Join(tmpDir, "assets.tar.gz")
	if !IsFile(in) {
		return fmt.Errorf("File '%s' does not exist", in)
	}

	app.Log(fmt.Sprintf("Unpacking '%s' to '%s'", in, assetsPath))

	if app.IgnoreResampled {
		app.Log("Ignoring resampled images")
	}

	err := TarGZExtract(in, assetsBase)
	if err != nil {
		return err
	}

	// only mark assets.old for deletion after assets.tar.gz has been successfully extracted
	app.AddTempFile(assetsPath + ".old")

	outSize, _ := DirSize(assetsPath)
	app.Log(fmt.Sprintf("Restored '%s' (%s)", assetsPath, outSize))

	return nil
}
