package archive

import (
	"io/fs"
)

// Ensure Entry implements fs.DirEntry
var _ fs.DirEntry = (*Entry)(nil)

func (e *Entry) Type() fs.FileMode {
	return e.Mode().Type()
}

func (e *Entry) Info() (fs.FileInfo, error) {
	return e, nil
}
