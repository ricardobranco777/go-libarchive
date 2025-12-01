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
	return h.Mode()&os.ModeDir != 0
}

// Mode returns the entry's permission bits as an os.FileMode.
// (Type bits are also included.)
func (h *Header) Mode() os.FileMode {
	mode := os.FileMode(h.UnixMode)

	if h.UnixMode&04000 != 0 {
		mode |= os.ModeSetuid
	}
	if h.UnixMode&02000 != 0 {
		mode |= os.ModeSetgid
	}
	if h.UnixMode&01000 != 0 {
		mode |= os.ModeSticky
	}

	switch mode & S_IFMT {
	case S_IFBLK:
		mode |= fs.ModeDevice
	case S_IFCHR:
		mode |= fs.ModeDevice | fs.ModeCharDevice
	case S_IFDIR:
		mode |= fs.ModeDir
	case S_IFIFO:
		mode |= fs.ModeNamedPipe
	case S_IFLNK:
		mode |= fs.ModeSymlink
	case S_IFREG:
		break
	case S_IFSOCK:
		mode |= fs.ModeSocket
	}

	return mode
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
