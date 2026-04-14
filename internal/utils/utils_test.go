package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axllent/ssbak/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))

	assert.True(t, utils.IsFile(file))
	assert.False(t, utils.IsFile(dir))
	assert.False(t, utils.IsFile(filepath.Join(dir, "nonexistent.txt")))
}

func TestIsDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0644))

	assert.True(t, utils.IsDir(dir))
	assert.False(t, utils.IsDir(file))
	assert.False(t, utils.IsDir(filepath.Join(dir, "nonexistent")))
}

func TestMkDirIfNotExists(t *testing.T) {
	dir := t.TempDir()

	// Already exists — should not error
	assert.NoError(t, utils.MkDirIfNotExists(dir))

	// New single-level directory
	newDir := filepath.Join(dir, "newdir")
	assert.NoError(t, utils.MkDirIfNotExists(newDir))
	assert.True(t, utils.IsDir(newDir))

	// Nested new directory
	nested := filepath.Join(dir, "a", "b", "c")
	assert.NoError(t, utils.MkDirIfNotExists(nested))
	assert.True(t, utils.IsDir(nested))
}

func TestCalcSize(t *testing.T) {
	dir := t.TempDir()

	// Single file
	content := []byte("hello world")
	file := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(file, content, 0644))

	size, err := utils.CalcSize(file)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)

	// Directory with multiple files
	subdir := filepath.Join(dir, "sub")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "a.txt"), []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "b.txt"), []byte("bb"), 0644))

	size, err = utils.CalcSize(subdir)
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)

	// Nonexistent path returns error
	_, err = utils.CalcSize(filepath.Join(dir, "nonexistent"))
	assert.Error(t, err)
}

func TestByteToHr(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0KiB"},
		{1536, "1.5KiB"},
		{1048576, "1.0MiB"},
		{1073741824, "1.0GiB"},
		{1099511627776, "1.0TiB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, utils.ByteToHr(tt.input))
		})
	}
}
