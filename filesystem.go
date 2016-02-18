package mailout

import "os"

// IsDir returns true if path is a directory
func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	return fileInfo != nil && fileInfo.IsDir() && err == nil
}

// fileExists returns true if file exists
func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return !os.IsNotExist(err) && fi.Size() > 0
}

// tempDir returns temporary directory ending with a path separatoe
func tempDir() string {
	dir := os.TempDir()
	const ps = string(os.PathSeparator)
	if len(dir) > 0 && dir[len(dir)-1:] != ps {
		dir = dir + ps
	}
	return dir
}
