package sspak

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axllent/ssbak/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetAppState resets the global app flags that affect sspak behaviour.
func resetAppState(t *testing.T) {
	t.Helper()
	prev := app.TempDir
	prevOnlyDB := app.OnlyDB
	prevOnlyAssets := app.OnlyAssets
	t.Cleanup(func() {
		app.TempDir = prev
		app.OnlyDB = prevOnlyDB
		app.OnlyAssets = prevOnlyAssets
	})
	app.OnlyDB = false
	app.OnlyAssets = false
}

func TestWriteAndOpen(t *testing.T) {
	resetAppState(t)
	tmpDir := t.TempDir()

	dbFile := filepath.Join(tmpDir, "database.sql.gz")
	assetsFile := filepath.Join(tmpDir, "assets.tar.gz")
	require.NoError(t, os.WriteFile(dbFile, []byte("fake sql gz"), 0644))
	require.NoError(t, os.WriteFile(assetsFile, []byte("fake assets gz"), 0644))

	f := &File{
		DatabaseFile: dbFile,
		AssetsFile:   assetsFile,
		TempFolder:   tmpDir,
	}

	sspakPath := filepath.Join(tmpDir, "test.sspak")
	require.NoError(t, f.Write(sspakPath))
	assert.FileExists(t, sspakPath)

	// Use a dedicated extraction dir so GetTempDir() doesn't collide between tests.
	app.TempDir = filepath.Join(t.TempDir(), "extracted")

	opened, err := Open(sspakPath)
	require.NoError(t, err)
	assert.NotEmpty(t, opened.DatabaseFile)
	assert.NotEmpty(t, opened.AssetsFile)
	assert.Contains(t, opened.DatabaseFile, "database.sql.gz")
	assert.Contains(t, opened.AssetsFile, "assets.tar.gz")
}

func TestWriteRequiresAtLeastOneFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := &File{TempFolder: tmpDir}

	err := f.Write(filepath.Join(tmpDir, "empty.sspak"))
	assert.Error(t, err)
}

func TestWriteOnlyDatabase(t *testing.T) {
	resetAppState(t)
	tmpDir := t.TempDir()

	dbFile := filepath.Join(tmpDir, "database.sql.gz")
	require.NoError(t, os.WriteFile(dbFile, []byte("db content"), 0644))

	f := &File{DatabaseFile: dbFile, TempFolder: tmpDir}
	sspakPath := filepath.Join(tmpDir, "db-only.sspak")

	require.NoError(t, f.Write(sspakPath))
	assert.FileExists(t, sspakPath)

	app.TempDir = filepath.Join(t.TempDir(), "ext1")
	opened, err := Open(sspakPath)
	require.NoError(t, err)
	assert.NotEmpty(t, opened.DatabaseFile)
	assert.Empty(t, opened.AssetsFile)
}

func TestExtractSSPakContents(t *testing.T) {
	resetAppState(t)
	tmpDir := t.TempDir()

	dbContent := []byte("database content")
	assetsContent := []byte("assets content")

	dbFile := filepath.Join(tmpDir, "database.sql.gz")
	assetsFile := filepath.Join(tmpDir, "assets.tar.gz")
	require.NoError(t, os.WriteFile(dbFile, dbContent, 0644))
	require.NoError(t, os.WriteFile(assetsFile, assetsContent, 0644))

	sspakPath := filepath.Join(tmpDir, "source.sspak")
	f := &File{DatabaseFile: dbFile, AssetsFile: assetsFile, TempFolder: tmpDir}
	require.NoError(t, f.Write(sspakPath))

	outDir := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	require.NoError(t, extractSSPakContents(sspakPath, outDir))

	assert.FileExists(t, filepath.Join(outDir, "database.sql.gz"))
	assert.FileExists(t, filepath.Join(outDir, "assets.tar.gz"))

	got, err := os.ReadFile(filepath.Join(outDir, "database.sql.gz"))
	require.NoError(t, err)
	assert.Equal(t, dbContent, got)
}

func TestExtractSSPakContentsOnlyDB(t *testing.T) {
	resetAppState(t)
	app.OnlyDB = true

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "database.sql.gz")
	assetsFile := filepath.Join(tmpDir, "assets.tar.gz")
	require.NoError(t, os.WriteFile(dbFile, []byte("db"), 0644))
	require.NoError(t, os.WriteFile(assetsFile, []byte("assets"), 0644))

	sspakPath := filepath.Join(tmpDir, "source.sspak")
	f := &File{DatabaseFile: dbFile, AssetsFile: assetsFile, TempFolder: tmpDir}
	require.NoError(t, f.Write(sspakPath))

	outDir := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	require.NoError(t, extractSSPakContents(sspakPath, outDir))

	assert.FileExists(t, filepath.Join(outDir, "database.sql.gz"))
	assert.NoFileExists(t, filepath.Join(outDir, "assets.tar.gz"))
}

func TestExtractSSPakContentsOnlyAssets(t *testing.T) {
	resetAppState(t)
	app.OnlyAssets = true

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "database.sql.gz")
	assetsFile := filepath.Join(tmpDir, "assets.tar.gz")
	require.NoError(t, os.WriteFile(dbFile, []byte("db"), 0644))
	require.NoError(t, os.WriteFile(assetsFile, []byte("assets"), 0644))

	sspakPath := filepath.Join(tmpDir, "source.sspak")
	f := &File{DatabaseFile: dbFile, AssetsFile: assetsFile, TempFolder: tmpDir}
	require.NoError(t, f.Write(sspakPath))

	outDir := filepath.Join(t.TempDir(), "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))

	require.NoError(t, extractSSPakContents(sspakPath, outDir))

	assert.NoFileExists(t, filepath.Join(outDir, "database.sql.gz"))
	assert.FileExists(t, filepath.Join(outDir, "assets.tar.gz"))
}
