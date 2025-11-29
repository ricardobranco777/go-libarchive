/* SPDX-License-Identifier: BSD-2-Clause */

package archive

/*
#cgo pkg-config: libarchive
#include <archive.h>
#include <archive_entry.h>
#include <stdlib.h>

extern ssize_t goReadCallback(struct archive*, void*, void*);
extern int goCloseCallback(struct archive*, void*);

static archive_read_callback *read_cb() {
    return (archive_read_callback *)goReadCallback;
}

static archive_close_callback *close_cb() {
    return (archive_close_callback *)goCloseCallback;
}
*/
import "C"

import (
	"io"
	"sync"
	"sync/atomic"
	"unsafe"
)

type readerState struct {
	r   io.Reader
	buf []byte
}

var (
	readers sync.Map
	nextKey atomic.Uintptr
)

//export goReadCallback
func goReadCallback(a *C.struct_archive, clientData unsafe.Pointer, buff unsafe.Pointer) C.ssize_t {
	key := *(*uintptr)(clientData)

	v, ok := readers.Load(key)
	if !ok {
		return -1
	}
	rs := v.(*readerState)

	n, err := rs.r.Read(rs.buf)
	if n > 0 {
		p := (*unsafe.Pointer)(buff)
		*p = unsafe.Pointer(&rs.buf[0])
		return C.ssize_t(n)
	}
	if err == io.EOF {
		return C.ssize_t(0)
	}
	return C.ssize_t(-1)
}

//export goCloseCallback
func goCloseCallback(a *C.struct_archive, clientData unsafe.Pointer) C.int {
	key := *(*uintptr)(clientData)
	readers.Delete(key)
	C.free(clientData)
	return C.ARCHIVE_OK
}

func OpenReader(r io.Reader) (*Archive, error) {
	ar, err := newArchive()
	if err != nil {
		return nil, err
	}

	key := nextKey.Add(1)

	readers.Store(key, &readerState{
		r:   r,
		buf: make([]byte, 32*1024),
	})

	// allocate C memory to hold the key
	cookie := C.malloc(C.size_t(unsafe.Sizeof(uintptr(0))))
	*(*uintptr)(cookie) = key

	if C.archive_read_open(
		ar.c,
		cookie,
		nil,
		C.read_cb(),
		C.close_cb(),
	) != C.ARCHIVE_OK {
		readers.Delete(key)
		C.free(cookie)
		err := wrapArchiveError(ar.c, "archive_read_open")
		_ = ar.Close()
		return nil, err
	}

	return ar, nil
}
