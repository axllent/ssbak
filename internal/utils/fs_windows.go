//go:build windows

package utils

// HasEnoughSpace does not work on Windows
func HasEnoughSpace(path string, requiredSize int64) error {
	return nil
}
