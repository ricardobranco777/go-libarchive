/* SPDX-License-Identifier: BSD-2-Clause */

package archive

/*
#cgo pkg-config: libarchive
#include <archive.h>
#include <archive_entry.h>
#include <stdlib.h>
*/
import "C"

import (
	"io"
	"io/fs"
	"os"
	"time"
	"unsafe"
)

// Entry represents a single entry (file, dir, symlink, ...) in an archive.
//
// An Entry holds a C struct allocated by archive_entry_new and freed
// by (*Entry).Close or a finalizer.
type Entry struct {
	a *Archive
	c *C.struct_archive_entry
}

var _ fs.FileInfo = (*Entry)(nil)

// Close frees the underlying archive_entry associated with e.
func (e *Entry) Close() error {
	C.archive_entry_free(e.c)
	e.c = nil
	return nil
}

// Read reads data from the current entry into p.
//
// It implements io.Reader. It returns 0, io.EOF when the entry is fully read.
func (e *Entry) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	n := C.archive_read_data(
		e.a.c,
		unsafe.Pointer(&p[0]),
		C.size_t(len(p)),
	)

	switch {
	case n > 0:
		return int(n), nil
	case n == 0:
		// End of entry data.
		return 0, io.EOF
	default:
		return 0, wrapArchiveError(e.a.c, "archive_read_data")
	}
}

// Skip entry by draining data for non-io.Seeker readers
func (e *Entry) Skip() error {
	if C.archive_read_data_skip(e.a.c) != C.ARCHIVE_OK {
		return wrapArchiveError(e.a.c, "archive_read_data_skip")
	}
	return nil
}

// Name returns the path/name of the entry inside the archive.
func (e *Entry) Name() string {
	cname := C.archive_entry_pathname(e.c)
	if cname == nil {
		return ""
	}
	return C.GoString(cname)
}

// Size returns the entry size in bytes, or -1 if unknown.
func (e *Entry) Size() int64 {
	return int64(C.archive_entry_size(e.c))
}

// Mode returns the entry's permission bits as an os.FileMode.
// (Type bits are also included.)
func (e *Entry) Mode() os.FileMode {
	return os.FileMode(C.archive_entry_mode(e.c))
}

// UID returns the numeric user ID for the entry, or -1 if unknown.
func (e *Entry) UID() int64 {
	return int64(C.archive_entry_uid(e.c))
}

// GID returns the numeric group ID for the entry, or -1 if unknown.
func (e *Entry) GID() int64 {
	return int64(C.archive_entry_gid(e.c))
}

// Linkname returns the target of a symlink entry, if any.
func (e *Entry) Linkname() string {
	cname := C.archive_entry_symlink(e.c)
	if cname == nil {
		return ""
	}
	return C.GoString(cname)
}

// ModTime returns the modification time of the entry.
// If the time is not set, it returns the zero time.
func (e *Entry) ModTime() time.Time {
	sec := int64(C.archive_entry_mtime(e.c))
	nsec := int64(C.archive_entry_mtime_nsec(e.c))
	if sec == 0 && nsec == 0 {
		return time.Time{}
	}
	return time.Unix(sec, nsec)
}

func (e *Entry) IsDir() bool {
	return e.Mode().IsDir()
}

type EntrySys struct {
	UID  uint32
	GID  uint32
	Size int64
	Mode fs.FileMode
}

func (e *Entry) Sys() any {
	st := C.archive_entry_stat(e.c)
	if st == nil {
		return nil
	}

	return &EntrySys{
		UID:  uint32(st.st_uid),
		GID:  uint32(st.st_gid),
		Size: int64(st.st_size),
		Mode: fs.FileMode(st.st_mode),
	}
}

// Ensure Entry implements fs.DirEntry
var _ fs.DirEntry = (*Entry)(nil)

func (e *Entry) Type() fs.FileMode {
	return e.Mode().Type()
}

func (e *Entry) Info() (fs.FileInfo, error) {
	return e, nil
}
