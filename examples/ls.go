/* SPDX-License-Identifier: BSD-2-Clause */

package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	libarchive "github.com/ricardobranco777/go-libarchive"
)

func main() {
	ar, err := libarchive.OpenFile("test.tar.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer ar.Close()

	for {
		e, err := ar.Next()
		if err != nil {
			// You'll probably want to switch this to io.EOF in the impl above.
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrNotExist) {
				break
			}
			log.Fatal(err)
		}

		fmt.Printf("Entry: %s (%d bytes)\n", e.Name(), e.Size())

		// Example: drain data
		var buf [32 * 1024]byte
		for {
			n, err := e.Read(buf[:])
			if n > 0 {
				// process buf[:n]
			}
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, os.ErrNotExist) {
					break
				}
				log.Fatal(err)
			}
		}

		e.Close()
	}
}
