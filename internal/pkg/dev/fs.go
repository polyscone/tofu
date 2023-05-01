package dev

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

var (
	exedir = filepath.ToSlash(filepath.Dir(errors.Must(os.Executable())))
	info   = errors.MustOK(debug.ReadBuildInfo())
)

// RelDirFS will return an os.DirFS with the given directory relative to the
// file it's called in.
func RelDirFS(dir string) fs.FS {
	dir = filepath.ToSlash(filepath.Join(fileDir(1), dir))
	dir = strings.ReplaceAll(dir, info.Main.Path+"/", "")
	dir = strings.ReplaceAll(dir, exedir+"/", "")

	return os.DirFS(dir)
}

func fileDir(skip int) string {
	_, file, _, ok := runtime.Caller(1 + skip)
	if ok {
		return path.Dir(file)
	}

	return ""
}
