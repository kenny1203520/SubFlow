//go:build !windows

package upgrade

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
