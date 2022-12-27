package fstack

import "io/fs"

// FStack implements a simple fs.FS stack, allowing for multiple fs.FS
// implementations to be used in conjunction with each other.
type FStack struct {
	stack []fs.FS
}

// New returns a new FStack where the stack is configured using the given
// fs.FS slice.
// The first fs.FS in the slice is treated as being at the top of the stack, and
// the last fs.FS in the slice is treated as being at the bottom.
func New(stack ...fs.FS) *FStack {
	return &FStack{stack: stack}
}

// Open will attempt to open the given file path checking each fs.FS in the
// configured file system stack one at a time.
// The first file system in the stack to not return an error is the one that
// will be returned from Open.
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
