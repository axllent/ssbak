package sspak

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/axllent/ssbak/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripTrailingSlash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/foo/bar", "/foo/bar"},
		{"/foo/bar/", "/foo/bar"},
		{"foo/", "foo"},
		{"/", ""},
		{"//", "/"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, stripTrailingSlash(tt.input), "input: %q", tt.input)
	}
}

func TestIsDirSspak(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("test"), 0644))

	assert.True(t, IsDir(dir))
	assert.False(t, IsDir(file))
	assert.False(t, IsDir(filepath.Join(dir, "nonexistent")))
}

func TestMkdirAll(t *testing.T) {
	dir := t.TempDir()

	// Existing directory — undo should be a no-op
	undo, err := mkdirAll(dir, 0750)
	require.NoError(t, err)
	undo()
	assert.True(t, IsDir(dir), "existing dir should still exist after undo no-op")

	// New directory — undo should remove it
	newDir := filepath.Join(dir, "new")
	undo, err = mkdirAll(newDir, 0750)
	require.NoError(t, err)
	assert.True(t, IsDir(newDir))

	undo()
	assert.False(t, IsDir(newDir), "undo should remove created directory")

	// Nested new directories — undo removes the first created ancestor
	nested := filepath.Join(dir, "a", "b", "c")
	undo, err = mkdirAll(nested, 0750)
	require.NoError(t, err)
	assert.True(t, IsDir(nested))

	undo()
	assert.False(t, IsDir(filepath.Join(dir, "a")), "undo should remove first created ancestor")
}

func TestTarAddDirectoryAndExtract(t *testing.T) {
	srcDir := t.TempDir()

	// Build a small directory tree
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("hello"), 0644))
	subDir := filepath.Join(srcDir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("world"), 0644))

	// Write to a tar.gz
	tarPath := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(tarPath)
	require.NoError(t, err)
	gzW := gzip.NewWriter(f)
	tarW := tar.NewWriter(gzW)

	app.IgnoreResampled = false
	require.NoError(t, tarAddDirectory(srcDir, tarW, assetsArchiveRoot))
	require.NoError(t, tarW.Close())
	require.NoError(t, gzW.Close())
	require.NoError(t, f.Close())

	// Extract and verify
	outDir := t.TempDir()
	require.NoError(t, extractAssets(tarPath, outDir))

	// The top-level directory is stripped on extraction
	assert.NoDirExists(t, filepath.Join(outDir, assetsArchiveRoot))
	assert.NoDirExists(t, filepath.Join(outDir, filepath.Base(srcDir)))
	assert.FileExists(t, filepath.Join(outDir, "file1.txt"))
	assert.FileExists(t, filepath.Join(outDir, "sub", "file2.txt"))

	// Verify file content survived the roundtrip
	got, err := os.ReadFile(filepath.Join(outDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), got)
}

func TestExtractAssetsGzip(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "image.png"), []byte("fakepng"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "doc.pdf"), []byte("fakepdf"), 0644))

	tarPath := filepath.Join(t.TempDir(), "assets.tar.gz")
	f, err := os.Create(tarPath)
	require.NoError(t, err)
	gzW := gzip.NewWriter(f)
	tarW := tar.NewWriter(gzW)

	app.IgnoreResampled = false
	require.NoError(t, tarAddDirectory(srcDir, tarW, assetsArchiveRoot))
	require.NoError(t, tarW.Close())
	require.NoError(t, gzW.Close())
	require.NoError(t, f.Close())

	outDir := t.TempDir()
	require.NoError(t, extractAssets(tarPath, outDir))

	assert.FileExists(t, filepath.Join(outDir, "image.png"))
	assert.FileExists(t, filepath.Join(outDir, "doc.pdf"))
}

func TestExtractAssetsTarSlipPrevention(t *testing.T) {
	// Build a malicious tar.gz that contains a path traversal entry (../../evil.txt)
	tarPath := filepath.Join(t.TempDir(), "evil.tar.gz")
	f, err := os.Create(tarPath)
	require.NoError(t, err)

	gzW := gzip.NewWriter(f)
	tarW := tar.NewWriter(gzW)

	// Attempt path traversal
	_ = tarW.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "../../evil.txt",
		Size:     int64(len("evil")),
		Mode:     0644,
	})
	_, _ = tarW.Write([]byte("evil"))

	require.NoError(t, tarW.Close())
	require.NoError(t, gzW.Close())
	require.NoError(t, f.Close())

	outDir := t.TempDir()
	require.NoError(t, extractAssets(tarPath, outDir))

	// The traversal file must NOT have been written above outDir
	parent := filepath.Dir(outDir)
	assert.NoFileExists(t, filepath.Join(parent, "evil.txt"))
	assert.NoFileExists(t, filepath.Join(filepath.Dir(parent), "evil.txt"))
}

func TestAssetRelPath(t *testing.T) {
	tests := []struct {
		name     string
		relPath  string
		topLevel bool
	}{
		{"assets", ".", true},
		{"assets/", ".", true},
		{"/assets", ".", true},
		{"./assets/", ".", true},
		{"assets/file.txt", "file.txt", false},
		{"/assets/sub/file.txt", "sub/file.txt", false},
		{"./assets/sub/", "sub", false},
		{"uploads/file.txt", "file.txt", false},
		{"../../evil.txt", ".", true},
		{"assets/../../evil.txt", ".", true},
		{"", "", false},
		{"/", "", false},
		{".", "", false},
	}

	for _, tt := range tests {
		relPath, topLevel := assetRelPath(tt.name)
		assert.Equal(t, tt.relPath, relPath, "name: %q", tt.name)
		assert.Equal(t, tt.topLevel, topLevel, "name: %q", tt.name)
	}
}

// tarEntryNames returns the entry names of a tar.gz file
func tarEntryNames(t *testing.T, tarPath string) []string {
	t.Helper()

	f, err := os.Open(tarPath)
	require.NoError(t, err)
	defer f.Close()

	gzR, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gzR.Close()

	names := []string{}
	tarR := tar.NewReader(gzR)
	for {
		h, err := tarR.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, h.Name)
	}

	return names
}

func TestTarAddDirectoryNamesAndSymlinks(t *testing.T) {
	app.IgnoreResampled = false
	base := t.TempDir()

	// Source directory deliberately not named "assets"
	srcDir := filepath.Join(base, "public", "test", "blaah")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("world"), 0644))

	// Symlinked directory & file pointing outside the source directory
	outside := filepath.Join(base, "outside")
	require.NoError(t, os.MkdirAll(outside, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "linked.txt"), []byte("linked"), 0644))
	require.NoError(t, os.Symlink(outside, filepath.Join(srcDir, "linkeddir")))
	require.NoError(t, os.Symlink(filepath.Join(outside, "linked.txt"), filepath.Join(srcDir, "linkedfile.txt")))

	// Symlink loop back to the source directory
	require.NoError(t, os.Symlink(srcDir, filepath.Join(srcDir, "sub", "loop")))

	// Archive via a symlink to the source directory
	link := filepath.Join(base, "assets-link")
	require.NoError(t, os.Symlink(srcDir, link))

	tarPath := filepath.Join(t.TempDir(), "assets.tar.gz")
	f, err := os.Create(tarPath)
	require.NoError(t, err)
	gzW := gzip.NewWriter(f)
	tarW := tar.NewWriter(gzW)
	require.NoError(t, tarAddDirectory(link, tarW, assetsArchiveRoot))
	require.NoError(t, tarW.Close())
	require.NoError(t, gzW.Close())
	require.NoError(t, f.Close())

	assert.ElementsMatch(t, []string{
		"assets/",
		"assets/file1.txt",
		"assets/linkeddir/",
		"assets/linkeddir/linked.txt",
		"assets/linkedfile.txt",
		"assets/sub/",
		"assets/sub/file2.txt",
	}, tarEntryNames(t, tarPath))

	// Linked files are stored as regular files with their contents
	outDir := t.TempDir()
	require.NoError(t, extractAssets(tarPath, outDir))
	got, err := os.ReadFile(filepath.Join(outDir, "linkedfile.txt"))
	require.NoError(t, err)
	assert.Equal(t, []byte("linked"), got)
	fi, err := os.Lstat(filepath.Join(outDir, "linkedfile.txt"))
	require.NoError(t, err)
	assert.True(t, fi.Mode().IsRegular())
}
