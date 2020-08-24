// +build !windows

package utils

import (
	"fmt"
	"syscall"
)

// HasEnoughSpace will return an error message if the provided path does not
// have sufficient storage space
func HasEnoughSpace(path string, requiredSize int64) error {
	var stat syscall.Statfs_t

	syscall.Statfs(path, &stat)

	// Available blocks * size per block = available space in bytes
	remainingBytes := stat.Bavail * uint64(stat.Bsize)

	storageExpected := uint64(requiredSize)

	if storageExpected > remainingBytes {
		return fmt.Errorf("%s does not have enough space available (+-%s required)", path, ByteToHr(requiredSize))
	}

	return nil
}
