package helpers

import "os"

func MakeSureDirectoryPathExists(path string) error {
	return os.MkdirAll(path, 0755)
}
