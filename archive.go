/* SPDX-License-Identifier: BSD-2-Clause */

package archive

/*
#cgo pkg-config: libarchive
#include <archive.h>
#include <archive_entry.h>
#include <stdlib.h>

extern ssize_t goReadCallback(struct archive*, uintptr_t, void**);
extern int goCloseCallback(struct archive*, uintptr_t);

static archive_read_callback *read_cb() {
    return (archive_read_callback *)goReadCallback;
}

static archive_close_callback *close_cb() {
    return (archive_close_callback *)goCloseCallback;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"runtime"
	"runtime/cgo"
	"time"
	"unsafe"
)

type readerState struct {
	r   io.Reader
	buf []byte
}

// Archive is an opened libarchive reader.
//
// It is NOT safe for concurrent use by multiple goroutines.
type Archive struct {
	c      *C.struct_archive
	handle cgo.Handle
	rs     *readerState // streaming state for OpenReader
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

// Skip draining data
func (e *Entry) Skip() error {
	var buf [32 * 1024]byte
	for {
		n, err := e.Read(buf[:])
		if n == 0 && err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// Close closes the archive and frees associated C resources.
// It is safe to call multiple times.
func (a *Archive) Close() error {
	if a.c == nil {
		return nil
	}
	r := C.archive_read_free(a.c)
	a.c = nil
	a.handle.Delete()
	if r == C.ARCHIVE_OK || r == C.ARCHIVE_WARN {
		return nil
	}
	return errors.New("libarchive: archive_read_free failed")
}

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

//export goReadCallback
func goReadCallback(a *C.struct_archive, clientData C.uintptr_t, buff *unsafe.Pointer) C.ssize_t {
	h := cgo.Handle(uintptr(clientData))
	ar := h.Value().(*Archive)

	n, err := ar.rs.r.Read(ar.rs.buf)
	if n > 0 {
		*buff = unsafe.Pointer(&ar.rs.buf[0])
		return C.ssize_t(n)
	}
	if err == io.EOF {
		return C.ssize_t(0)
	}
	return C.ssize_t(-1)
}

//export goCloseCallback
func goCloseCallback(a *C.struct_archive, clientData C.uintptr_t) C.int {
	return C.ARCHIVE_OK
}

func OpenReader(r io.Reader) (*Archive, error) {
	ar, err := newArchive()
	if err != nil {
		return nil, err
	}

	ar.rs = &readerState{
		r:   r,
		buf: make([]byte, 32*1024),
	}

	ar.handle = cgo.NewHandle(ar)

	if C.archive_read_open(
		ar.c,
		unsafe.Pointer(uintptr(ar.handle)),
		nil,
		C.read_cb(),
		C.close_cb(),
	) != C.ARCHIVE_OK {
		ar.handle.Delete()
		err := wrapArchiveError(ar.c, "archive_read_open")
		_ = ar.Close()
		return nil, err
	}

	return ar, nil
}
