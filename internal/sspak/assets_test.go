package sspak

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axllent/ssbak/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkipResampled(t *testing.T) {
	// When IgnoreResampled is false nothing is ever skipped.
	app.IgnoreResampled = false

	for _, path := range []string{
		"/assets/photo__CropWzEwMCwxMDBd.jpg",
		"/assets/photo__FillWzgwLDgwXQ.jpg",
		"/assets/_resampled/CMSThumbnail/photo.jpg",
		"/assets/regular_image.jpg",
	} {
		assert.False(t, skipResampled(path), "IgnoreResampled=false: expected false for %s", path)
	}

	app.IgnoreResampled = true
	defer func() { app.IgnoreResampled = false }()

	shouldSkip := []string{
		// SilverStripe 4/5 variants
		"/assets/photo__CropWzEwMCwxMDBd.jpg",
		"/assets/photo__FillWzgwLDgwXQ.jpg",
		"/assets/photo__FitWzgwLDgwXQ.png",
		"/assets/photo__FocusWzEwMCwxMDBd.jpg",
		"/assets/photo__PadWzEwMCwxMDBd.jpg",
		"/assets/photo__QualityWzgwXQ.jpg",
		"/assets/photo__ResampledWzgwLDgwXQ.jpg",
		"/assets/photo__ScaleWidthWzEwMFd.jpg",
		"/assets/photo__ExtRewriteWzgwXQ.jpg",
		// SilverStripe 3
		"/assets/_resampled/CMSThumbnail/photo.jpg",
		"/assets/_resampled/PadWzEwMCwxMDBd/photo.jpg",
		"/assets/_resampled/FillWzgwLDgwXQ/photo.jpg",
		"/assets/_resampled/CroppedImage100x100/photo.jpg",
	}

	for _, path := range shouldSkip {
		assert.True(t, skipResampled(path), "expected true for %s", path)
	}

	shouldNotSkip := []string{
		// Regular uploads
		"/assets/uploads/photo.jpg",
		"/assets/hero-banner.png",
		// SS5 CMS preview thumbnails are explicitly excluded from the skip rule
		"/assets/photo__FitMaxWzM1MiwyNjRd.jpg",
		"/assets/photo__FitMaxWzM1MiwyNjRd.png",
	}

	for _, path := range shouldNotSkip {
		assert.False(t, skipResampled(path), "expected false for %s", path)
	}
}

func TestAddAssetsAndLoadAssets(t *testing.T) {
	app.IgnoreResampled = false
	UseZSTD = false

	// Source: a directory named "assets" with a few files
	srcBase := t.TempDir()
	assetsDir := filepath.Join(srcBase, "assets")
	require.NoError(t, os.MkdirAll(assetsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "photo.jpg"), []byte("jpeg data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "doc.pdf"), []byte("pdf data"), 0644))

	// AddAssets — store the archive in a temp folder
	archiveDir := t.TempDir()
	f := &File{TempFolder: archiveDir}
	require.NoError(t, f.AddAssets(assetsDir))
	assert.FileExists(t, f.AssetsFile)

	// LoadAssets — extract into a fresh destination
	destBase := t.TempDir()
	require.NoError(t, f.LoadAssets(destBase))

	assert.FileExists(t, filepath.Join(destBase, "assets", "photo.jpg"))
	assert.FileExists(t, filepath.Join(destBase, "assets", "doc.pdf"))

	// Verify content survived the roundtrip
	got, err := os.ReadFile(filepath.Join(destBase, "assets", "photo.jpg"))
	require.NoError(t, err)
	assert.Equal(t, []byte("jpeg data"), got)
}

func TestAddAssetsSkipsResampled(t *testing.T) {
	app.IgnoreResampled = true
	UseZSTD = false
	defer func() { app.IgnoreResampled = false }()

	srcBase := t.TempDir()
	assetsDir := filepath.Join(srcBase, "assets")
	require.NoError(t, os.MkdirAll(assetsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "photo.jpg"), []byte("real"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "photo__FillWzgwLDgwXQ.jpg"), []byte("thumb"), 0644))

	archiveDir := t.TempDir()
	f := &File{TempFolder: archiveDir}
	require.NoError(t, f.AddAssets(assetsDir))

	destBase := t.TempDir()
	require.NoError(t, f.LoadAssets(destBase))

	assert.FileExists(t, filepath.Join(destBase, "assets", "photo.jpg"))
	assert.NoFileExists(t, filepath.Join(destBase, "assets", "photo__FillWzgwLDgwXQ.jpg"))
}

func TestAddAssetsEmptyDirectory(t *testing.T) {
	emptyDir := t.TempDir()
	f := &File{TempFolder: t.TempDir()}

	err := f.AddAssets(emptyDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestAddAssetsZSTD(t *testing.T) {
	app.IgnoreResampled = false
	UseZSTD = true
	defer func() { UseZSTD = false }()

	srcBase := t.TempDir()
	assetsDir := filepath.Join(srcBase, "assets")
	require.NoError(t, os.MkdirAll(assetsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "file.txt"), []byte("zstd content"), 0644))

	archiveDir := t.TempDir()
	f := &File{TempFolder: archiveDir}
	require.NoError(t, f.AddAssets(assetsDir))
	assert.FileExists(t, f.AssetsFile)
	assert.Contains(t, f.AssetsFile, ".tar.zst")

	destBase := t.TempDir()
	require.NoError(t, f.LoadAssets(destBase))

	assert.FileExists(t, filepath.Join(destBase, "assets", "file.txt"))
	got, err := os.ReadFile(filepath.Join(destBase, "assets", "file.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("zstd content"), got)
}
