package sspak

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAssetsTarget(t *testing.T) {
	base := t.TempDir()
	webroot := filepath.Join(base, "release")
	public := filepath.Join(webroot, "public")
	shared := filepath.Join(base, "shared")
	require.NoError(t, os.MkdirAll(filepath.Join(public, "assets"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(shared, "assets"), 0755))

	sspakFile := filepath.Join(base, "backups", "site.sspak")
	require.NoError(t, os.MkdirAll(filepath.Dir(sspakFile), 0755))
	require.NoError(t, os.WriteFile(sspakFile, []byte("x"), 0644))

	// symlinked webroot (eg: releases/current -> releases/123)
	current := filepath.Join(base, "current")
	require.NoError(t, os.Symlink(webroot, current))

	protected := []string{webroot, public, sspakFile, ""}

	allowed := []string{
		filepath.Join(public, "assets"),
		filepath.Join(public, "test", "blaah"), // does not exist yet
		filepath.Join(shared, "assets"),
		filepath.Join(webroot, "assets"),
		filepath.Join(current, "public", "assets"),
	}
	for _, target := range allowed {
		assert.NoError(t, ValidateAssetsTarget(target, protected...), "expected %s to be allowed", target)
	}

	refused := []string{
		"/",
		base,                       // parent of webroot
		webroot,                    // webroot
		public,                     // public directory
		filepath.Join(public, "."), // public directory (unclean)
		filepath.Join(public, "assets", ".."),
		filepath.Dir(sspakFile),          // contains the .sspak
		filepath.Join(current, "public"), // public via symlinked parent
	}
	for _, target := range refused {
		assert.Error(t, ValidateAssetsTarget(target, protected...), "expected %s to be refused", target)
	}

	// the webroot given via a symlink is protected in both forms
	assert.Error(t, ValidateAssetsTarget(current, current))
	assert.Error(t, ValidateAssetsTarget(webroot, current))
	assert.Error(t, ValidateAssetsTarget(filepath.Join(current, "public"), filepath.Join(current, "public")))
}

func TestValidateAssetsTargetSymlinkedTarget(t *testing.T) {
	base := t.TempDir()
	webroot := filepath.Join(base, "release")
	require.NoError(t, os.MkdirAll(filepath.Join(webroot, "public"), 0755))

	// public/assets -> webroot: restoring there renames the link, not the webroot
	link := filepath.Join(webroot, "public", "assets")
	require.NoError(t, os.Symlink(webroot, link))
	assert.NoError(t, ValidateAssetsTarget(link, webroot))

	// .sspak located inside a symlinked target is refused (the link would be renamed)
	sspakFile := filepath.Join(link, "site.sspak")
	require.NoError(t, os.WriteFile(sspakFile, []byte("x"), 0644))
	assert.Error(t, ValidateAssetsTarget(link, sspakFile))
}

func TestContainsPath(t *testing.T) {
	assert.True(t, containsPath("/a/b", "/a/b"))
	assert.True(t, containsPath("/a/b", "/a/b/c"))
	assert.True(t, containsPath("/", "/a"))
	assert.False(t, containsPath("/a/b", "/a"))
	assert.False(t, containsPath("/a/b", "/a/bc"))
	assert.False(t, containsPath("/a/b", "/a/..b"))
	assert.True(t, containsPath("/a/b", "/a/b/..c"))
}
