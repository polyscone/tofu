package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/polyscone/tofu/errsx"
)

var (
	exedir = filepath.ToSlash(filepath.Dir(errsx.Must(os.Executable())))
	info   = errsx.MustOK(debug.ReadBuildInfo())
)

var ErrBlacklisted = errors.New("file is blacklisted")

func fileDir(skip int) string {
	_, file, _, ok := runtime.Caller(1 + skip)
	if ok {
		return path.Dir(file)
	}

	return ""
}

// RelDirFS will return an os.DirFS with the given directory relative to the
// file it's called in.
func RelDirFS(dir string) fs.FS {
	dir = filepath.ToSlash(filepath.Join(fileDir(1), dir))
	dir = strings.ReplaceAll(dir, info.Main.Path+"/", "")
	dir = strings.ReplaceAll(dir, exedir+"/", "")

	return os.DirFS(dir)
}

type RestrictedFSAllowedFunc func(name string) bool

type Restricted struct {
	fsys    fs.FS
	allowed RestrictedFSAllowedFunc
}

func NewRestricted(fsys fs.FS, allowed RestrictedFSAllowedFunc) *Restricted {
	return &Restricted{
		fsys:    fsys,
		allowed: allowed,
	}
}

func (r *Restricted) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	if !r.allowed(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: ErrBlacklisted}
	}

	return r.fsys.Open(name)
}
