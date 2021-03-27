// +build !windows

package utils

import (
	"fmt"
	"path"
	"syscall"
)

// HasEnoughSpace will return an error message if the provided location does not
// have sufficient storage space
func HasEnoughSpace(location string, requiredSize int64) error {
	location = path.Join(location)
	var stat syscall.Statfs_t

	if err := syscall.Statfs(location, &stat); err != nil {
		return err
	}

	// Available blocks * size per block = available space in bytes
	remainingBytes := stat.Bavail * uint64(stat.Bsize)

	storageExpected := uint64(requiredSize)

	if storageExpected > remainingBytes {
		return fmt.Errorf(
			"'%s' does not have enough space available (+-%s required, %s available)",
			location,
			ByteToHr(requiredSize),
			ByteToHr(int64(remainingBytes)),
		)
	}

	return nil
}
