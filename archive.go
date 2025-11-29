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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"time"
	"unsafe"
)

// Archive is an opened libarchive reader.
//
// It is NOT safe for concurrent use by multiple goroutines.
type Archive struct {
	c  *C.struct_archive
	rs *readerState // streaming state for OpenReader
}

// Entry represents a single entry (file, dir, symlink, ...) in an archive.
//
// An Entry holds a C struct allocated by archive_entry_new and freed
// by (*Entry).Close or a finalizer.
type Entry struct {
	a *Archive
	c *C.struct_archive_entry
}

var _ fs.FileInfo = (*Entry)(nil)

// Error wraps libarchive's error state for an archive.
type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Msg != "" {
		return e.Msg
	}
	return fmt.Sprintf("libarchive error code %d", e.Code)
}

// FileType describes the type of a filesystem entry.
type FileType int

const (
	TypeUnknown FileType = iota
	TypeRegular
	TypeDirectory
	TypeSymlink
	TypeChar
	TypeBlock
	TypeFIFO
	TypeSocket
)

// newArchive creates a new libarchive reader and enables all filters/formats.
func newArchive() (*Archive, error) {
	a := C.archive_read_new()
	if a == nil {
		return nil, errors.New("libarchive: archive_read_new failed")
	}

	// Enable all filters and formats. You can restrict this later if desired.
	if C.archive_read_support_filter_all(a) != C.ARCHIVE_OK {
		err := wrapArchiveError(a, "archive_read_support_filter_all")
		C.archive_read_free(a)
		return nil, err
	}
	if C.archive_read_support_format_all(a) != C.ARCHIVE_OK {
		err := wrapArchiveError(a, "archive_read_support_format_all")
		C.archive_read_free(a)
		return nil, err
	}

	ar := &Archive{c: a}
	runtime.SetFinalizer(ar, func(ar *Archive) {
		_ = ar.Close()
	})
	return ar, nil
}

// OpenFile opens an archive from a filesystem path.
func OpenFile(path string) (*Archive, error) {
	ar, err := newArchive()
	if err != nil {
		return nil, err
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	// 10240 is the recommended default block size in examples.
	if C.archive_read_open_filename(ar.c, cpath, 10240) != C.ARCHIVE_OK {
		err := wrapArchiveError(ar.c, "archive_read_open_filename")
		_ = ar.Close()
		return nil, err
	}

	return ar, nil
}

// Close closes the archive and frees associated C resources.
// It's safe to call multiple times.
func (a *Archive) Close() error {
	// archive_read_free returns ARCHIVE_OK or ARCHIVE_WARN on success.
	r := C.archive_read_free(a.c)
	a.c = nil
	if r == C.ARCHIVE_OK || r == C.ARCHIVE_WARN {
		return nil
	}
	return errors.New("libarchive: archive_read_free failed")
}

// Next advances to the next entry in the archive.
// It returns io.EOF when there are no more entries.
func (a *Archive) Next() (*Entry, error) {
	entry := C.archive_entry_new()
	if entry == nil {
		return nil, errors.New("libarchive: archive_entry_new failed")
	}

	r := C.archive_read_next_header2(a.c, entry)
	switch r {
	case C.ARCHIVE_EOF:
		C.archive_entry_free(entry)
		return nil, io.EOF
	case C.ARCHIVE_OK:
		e := &Entry{a: a, c: entry}
		runtime.SetFinalizer(e, func(e *Entry) {
			_ = e.Close()
		})
		return e, nil
	default:
		err := wrapArchiveError(a.c, "archive_read_next_header2")
		C.archive_entry_free(entry)
		return nil, err
	}
}

// Close frees the underlying archive_entry associated with e.
// It is safe to call multiple times.
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
	UID      int64
	GID      int64
	Dev      uint64
	Linkname string
}

func (e *Entry) Sys() any {
	return &EntrySys{
		UID:      e.UID(),
		GID:      e.GID(),
		Dev:      uint64(C.archive_entry_dev(e.c)),
		Linkname: e.Linkname(),
	}
}

// wrapArchiveError reads errno and error string from the archive
// and returns a Go error.
func wrapArchiveError(a *C.struct_archive, message string) error {
	if a == nil {
		return &Error{
			Code: 0,
			Msg:  message,
		}
	}
	code := int(C.archive_errno(a))
	msg := C.archive_error_string(a)
	if msg == nil {
		return &Error{
			Code: code,
			Msg:  message,
		}
	}
	return &Error{
		Code: code,
		Msg:  fmt.Sprintf("%s: %s", message, C.GoString(msg)),
	}
}
