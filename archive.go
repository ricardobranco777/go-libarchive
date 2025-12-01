/* SPDX-License-Identifier: BSD-2-Clause */

package archive

/*
#include <archive.h>
#include <archive_entry.h>
#include <stdlib.h>
extern ssize_t goReadCallback(struct archive*, uintptr_t, void**);
extern int64_t goSeekCallback(struct archive*, uintptr_t, int64_t, int);
extern int goCloseCallback(struct archive*, uintptr_t);

static ssize_t my_libarchive_open(struct archive *a, uintptr_t client_data) {
	return archive_read_open(a, (void *)client_data, NULL,
		(archive_read_callback *)goReadCallback,
		(archive_close_callback *)goCloseCallback);
}

static archive_seek_callback *seek_cb() {
	return (archive_seek_callback *)goSeekCallback;
}
*/
import "C"

import (
	"errors"
	"io"
	"os"
	"runtime"
	"runtime/cgo"
	"strings"
	"time"
	"unsafe"
)

// Header represents a single entry (file, dir, symlink, ...) in an archive.
type Header struct {
	Pathname string    // Name of file entry
	Linkname string    // Target name of link
	Uid      int       // User ID of owner
	Gid      int       // Group ID of owner
	Uname    string    // User name of owner
	Gname    string    // Group name of owner
	Modified time.Time // Modification time
	UnixMode uint32    // Permission and mode bits
	a        *Archive
	size     int64 // Logical file size in bytes
}

// Read reads data from the current entry into p.
//
// It implements io.Reader. It returns 0, io.EOF when the entry is fully read.
func (h *Header) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	n := C.archive_read_data(h.a.c, unsafe.Pointer(&p[0]), C.size_t(len(p)))

	switch {
	case n > 0:
		return int(n), nil
	case n == 0:
		// End of entry data.
		return 0, io.EOF
	default:
		return 0, wrapArchiveError(h.a.c, "archive_read_data")
	}
}

// Skip entry by draining data for non-io.Seeker readers
func (h *Header) Skip() error {
	if C.archive_read_data_skip(h.a.c) != C.ARCHIVE_OK {
		return wrapArchiveError(h.a.c, "archive_read_data_skip")
	}
	return nil
}

type readerState struct {
	r   io.Reader
	buf []byte
}

// Archive is an opened libarchive reader.
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
func (a *Archive) Next() (*Header, error) {
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
		st := C.archive_entry_stat(entry)

		// We can't use stat info as st_mtime / st_mtime_nsec may not exist
		sec := int64(C.archive_entry_mtime(entry))
		nsec := int64(C.archive_entry_mtime_nsec(entry))

		symlink := ""
		if C.archive_entry_filetype(entry) == C.AE_IFLNK {
			symlink = C.GoString(C.archive_entry_symlink(entry))
		}

		pathname := strings.TrimRight(C.GoString(C.archive_entry_pathname(entry)), string(os.PathSeparator))

		h := &Header{
			Pathname: pathname,
			Linkname: symlink,
			UnixMode: uint32(st.st_mode),
			Uid:      int(st.st_uid),
			Gid:      int(st.st_gid),
			Uname:    C.GoString(C.archive_entry_uname(entry)),
			Gname:    C.GoString(C.archive_entry_gname(entry)),
			Modified: time.Unix(sec, nsec),
			a:        a,
			size:     int64(st.st_size),
		}
		C.archive_entry_free(entry)
		return h, nil

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
func goReadCallback(a *C.struct_archive, clientData C.uintptr_t, buf *unsafe.Pointer) C.ssize_t {
	h := cgo.Handle(uintptr(clientData))
	ar := h.Value().(*Archive)

	n, err := ar.rs.r.Read(ar.rs.buf)
	if n > 0 {
		*buf = unsafe.Pointer(&ar.rs.buf[0])
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

// Returns an Archive for r or error
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
			err := wrapArchiveError(ar.c, "archive_read_set_seek_callback")
			_ = ar.Close()
			return nil, err
		}
	}

	if C.my_libarchive_open(ar.c, C.uintptr_t(ar.handle)) != C.ARCHIVE_OK {
		err := wrapArchiveError(ar.c, "archive_read_open")
		_ = ar.Close()
		return nil, err
	}

	return ar, nil
}
