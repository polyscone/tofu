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

	"github.com/polyscone/tofu/internal/errsx"
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

type RestrictedConfig struct {
	AllowOpen     func(name string) (bool, error)
	AllowDirEntry func(dir string, entry fs.DirEntry) (bool, error)
}

type Restricted struct {
	fsys   fs.FS
	config RestrictedConfig
}

func NewRestricted(fsys fs.FS, config RestrictedConfig) *Restricted {
	return &Restricted{
		fsys:   fsys,
		config: config,
	}
}

func (r *Restricted) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	if r.config.AllowOpen != nil {
		allowed, err := r.config.AllowOpen(name)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, &fs.PathError{Op: "open", Path: name, Err: ErrBlacklisted}
		}
	}

	return r.fsys.Open(name)
}

func (r *Restricted) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(r.fsys, name)
	if err != nil {
		return nil, err
	}

	if r.config.AllowDirEntry == nil {
		return entries, nil
	}

	var filtered []fs.DirEntry
	for _, entry := range entries {
		allowed, err := r.config.AllowDirEntry(name, entry)
		if err != nil {
			return nil, err
		}
		if allowed {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}
