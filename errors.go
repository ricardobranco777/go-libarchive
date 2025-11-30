/* SPDX-License-Identifier: BSD-2-Clause */

package archive

/*
#cgo CFLAGS: -I/usr/local/include -I/usr/include
#cgo LDFLAGS: -L/usr/local/lib -larchive
#include <archive.h>
#include <archive_entry.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
)

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
