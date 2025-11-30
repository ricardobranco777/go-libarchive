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
	// ar, err := libarchive.OpenFile("test.tar.gz")
	ar, err := libarchive.OpenReader(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	defer ar.Close()

	for {
		e, err := ar.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Fatal(err)
		}

		fmt.Printf("Entry: %s (%d bytes)\n", e.Name(), e.Size())
		e.Close()
	}
}
