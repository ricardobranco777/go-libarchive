/* SPDX-License-Identifier: BSD-2-Clause */

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	libarchive "github.com/ricardobranco777/go-libarchive"
)

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func main() {
	err := libarchive.Walk(os.Stdin, func(h *libarchive.Header) error {
		t := ""
		if h.IsDir() {
			t = "/"
		}
		fmt.Printf("%s%s/%s %5d %s %s%s", libarchive.StrMode(h.UnixMode), h.Uname, h.Gname, h.Size(), formatTime(h.ModTime()), h.Name(), t)
		if h.Linkname != "" {
			fmt.Printf(" -> %s", h.Linkname)
		}
		fmt.Println()
		return h.Skip()
	})
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
}
