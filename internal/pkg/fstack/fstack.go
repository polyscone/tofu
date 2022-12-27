package fstack

import "io/fs"

type FStack struct {
	stack []fs.FS
}

func New(stack ...fs.FS) *FStack {
	return &FStack{stack: stack}
}

func (f *FStack) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	for _, el := range f.stack {
		if f, err := el.Open(name); err == nil {
			return f, nil
		}
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
