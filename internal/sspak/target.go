package sspak

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateAssetsTarget checks that restoring assets to target is safe. Restoring
// renames an existing target directory and deletes it afterwards, so target must
// not be the filesystem root, and must not be (or contain) any of the protected
// paths, eg: the webroot, the public directory, the .sspak file being restored,
// or the current working directory. Empty protected paths are ignored.
func ValidateAssetsTarget(target string, protected ...string) error {
	targets, err := targetForms(target)
	if err != nil {
		return err
	}

	for _, t := range targets {
		if t == filepath.Dir(t) {
			return fmt.Errorf("refusing to restore assets to '%s': filesystem root", target)
		}
	}

	for _, p := range protected {
		if p == "" {
			continue
		}

		protectedPaths, err := protectedForms(p)
		if err != nil {
			return err
		}

		for _, t := range targets {
			for _, pp := range protectedPaths {
				if containsPath(t, pp) {
					return fmt.Errorf("refusing to restore assets to '%s': it would replace '%s'", target, p)
				}
			}
		}
	}

	return nil
}

// targetForms returns the absolute path of target, plus the same path with any
// symlinks in its parent directories resolved. The final element is never
// resolved, as that is what gets renamed (a symlink is renamed, not its target).
func targetForms(target string) ([]string, error) {
	abs, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}

	parent, err := resolveExisting(filepath.Dir(abs))
	if err != nil {
		return nil, err
	}

	return []string{abs, filepath.Join(parent, filepath.Base(abs))}, nil
}

// protectedForms returns the absolute path of p, plus the same path with all symlinks resolved.
func protectedForms(p string) ([]string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return nil, err
	}

	resolved, err := resolveExisting(abs)
	if err != nil {
		return nil, err
	}

	return []string{abs, resolved}, nil
}

// resolveExisting resolves symlinks in the longest existing prefix of the
// absolute path p, appending any remaining (non-existent) path elements.
func resolveExisting(p string) (string, error) {
	rest := []string{}

	for {
		resolved, err := filepath.EvalSymlinks(p)
		if err == nil {
			return filepath.Join(append([]string{resolved}, rest...)...), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Join(append([]string{p}, rest...)...), nil
		}
		rest = append([]string{filepath.Base(p)}, rest...)
		p = parent
	}
}

// containsPath returns whether p is dir, or is located within dir.
func containsPath(dir, p string) bool {
	rel, err := filepath.Rel(dir, p)
	if err != nil {
		return false
	}

	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
