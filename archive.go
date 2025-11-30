/* SPDX-License-Identifier: BSD-2-Clause */

package archive

/*
#cgo pkg-config: libarchive
#include <archive.h>
#include <archive_entry.h>
#include <stdlib.h>
extern ssize_t goReadCallback(struct archive*, uintptr_t, void**);
extern int64_t goSeekCallback(struct archive*, uintptr_t, int64_t, int);
extern int goCloseCallback(struct archive*, uintptr_t);

static archive_read_callback *read_cb() {
	return (archive_read_callback *)goReadCallback;
}
static archive_seek_callback *seek_cb() {
	return (archive_seek_callback *)goSeekCallback;
}
static archive_close_callback *close_cb() {
	return (archive_close_callback *)goCloseCallback;
}
*/
import "C"

import (
	"errors"
	"io"
	"runtime"
	"runtime/cgo"
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
			C.archive_entry_free(e.c)
		})
		return e, nil

	default:
		C.archive_entry_free(entry)
		return nil, wrapArchiveError(a.c, "archive_read_next_header2")
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
	if a.handle != 0 {
		a.handle.Delete()
		a.handle = 0
	}
	if r == C.ARCHIVE_OK || r == C.ARCHIVE_WARN {
		return nil
	}
	return errors.New("libarchive: archive_read_free failed")
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

//export goSeekCallback
func goSeekCallback(a *C.struct_archive, clientData C.uintptr_t, offset C.int64_t, whence C.int) C.int64_t {
	h := cgo.Handle(uintptr(clientData))
	ar := h.Value().(*Archive)

	if s, ok := ar.rs.r.(io.Seeker); ok {
		n, err := s.Seek(int64(offset), int(whence))
		if err == nil {
			return C.int64_t(n)
		}
	}
	return C.int64_t(-1)
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

	// If the underlying reader also implements io.Seeker,
	// tell libarchive it can try to seek.
	if _, ok := r.(io.Seeker); ok {
		if C.archive_read_set_seek_callback(ar.c, C.seek_cb()) != C.ARCHIVE_OK {
			_ = ar.Close()
			return nil, wrapArchiveError(ar.c, "archive_read_set_seek_callback")
		}
	}

	if C.archive_read_open(
		ar.c,
		unsafe.Pointer(uintptr(ar.handle)),
		nil,
		C.read_cb(),
		C.close_cb(),
	) != C.ARCHIVE_OK {
		err := wrapArchiveError(ar.c, "archive_read_open")
		_ = ar.Close()
		return nil, err
	}

	return ar, nil
}
