package dev

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

// RelDirFS will return an os.DirFS with the given directory relative to the
// file it's called in.
func RelDirFS(dir string) fs.FS {
	dir = filepath.Join(fileDir(1), dir)

	return os.DirFS(dir)
}

func fileDir(skip int) string {
	_, file, _, ok := runtime.Caller(1 + skip)
	if ok {
		return path.Dir(file)
	}

	return ""
}
