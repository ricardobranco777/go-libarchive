/* SPDX-License-Identifier: BSD-2-Clause */

package archive

import (
	"io/fs"
	"os"
	"time"
)

// Ensure Header implements fs.FileInfo
var _ fs.FileInfo = (*Header)(nil)

// IsDir returns true if entry is a directory.
func (h *Header) IsDir() bool {
	return h.FileMode&os.ModeDir != 0
}

// Mode returns the entry's permission bits as an os.FileMode.
// (Type bits are also included.)
func (h *Header) Mode() os.FileMode {
	return h.FileMode
}

// ModTime returns the modification time of the entry.
// If the time is not set, it returns the zero time.
func (h *Header) ModTime() time.Time {
	return h.Modified
}

// Name returns the path/name of the entry inside the archive.
func (h *Header) Name() string {
	return h.Pathname
}

// Size returns the entry size in bytes
func (h *Header) Size() int64 {
	return h.size
}

// Sys returns Header
func (h *Header) Sys() any {
	return h
}
