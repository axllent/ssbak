// Package utils contains various utilities
package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/axllent/ssbak/app"
)

// AssetsToTarGz creates a tar.gz from the assets folder
func AssetsToTarGz(assetsDir, gzipFile string) error {
	app.Log(fmt.Sprintf("Calculating size of '%s'", assetsDir))

	size, _ := CalcSize(assetsDir)
	app.Log(fmt.Sprintf("Compressing '%s' (%s) to '%s'", assetsDir, ByteToHr(size), gzipFile))

	if err := HasEnoughSpace(path.Dir(gzipFile), size); err != nil {
		return err
	}

	if app.IgnoreResampled {
		app.Log("Ignoring resampled images")
	}

	err := TarGZCompress(assetsDir, gzipFile)

	outSize, _ := CalcSize(gzipFile)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", gzipFile, ByteToHr(outSize)))

	return err
}

// AssetsFromTarGz extracts assets from a tar.gz. If an existing assets directory is found
// then the existing one is renamed assets.old, and then deleted after the process completes.
func AssetsFromTarGz(tmpDir, assetsBase string) error {
	in := filepath.Join(tmpDir, "assets.tar.gz")
	if !IsFile(in) {
		return fmt.Errorf("file '%s' does not exist", in)
	}

	inSize, _ := CalcSize(in)

	if assetsBase == "" {
		assetsBase = "."
	}

	// Test output directory has sufficient space. It's not entirely
	// accurate as we are using a targz value, but we do not know the output size.
	if err := HasEnoughSpace(assetsBase, inSize); err != nil {
		return err
	}

	assetsPath := filepath.Join(assetsBase, "assets")
	if IsDir(assetsPath) {
		app.Log(fmt.Sprintf("Renaming existing '%s' to '%s.old'", assetsPath, assetsPath))
		if err := os.Rename(assetsPath, assetsPath+".old"); err != nil {
			return err
		}
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

	outSize, _ := CalcSize(assetsPath)
	app.Log(fmt.Sprintf("Restored '%s' (%s)", assetsPath, ByteToHr(outSize)))

	return nil
}
