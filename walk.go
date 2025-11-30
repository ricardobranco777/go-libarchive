/* SPDX-License-Identifier: BSD-2-Clause */

package archive

import (
	"io"
)

// WalkFunc is the callback used by Walk.
// h contains the header metadata
type WalkFunc func(h *Header) error

// Walk iterates through each entry in the archive.
// Returning a non-nil error stops iteration immediately.
func Walk(r io.Reader, fn WalkFunc) error {
	ar, err := OpenReader(r)
	if err != nil {
		return err
	}
	defer ar.Close()

	for {
		h, err := ar.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if err := fn(h); err != nil {
			return err
		}
		if err := h.Skip(); err != nil {
			return err
		}
	}
}
