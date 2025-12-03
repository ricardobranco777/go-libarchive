/* SPDX-License-Identifier: BSD-2-Clause */

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	libarchive "github.com/ricardobranco777/go-libarchive"
	"github.com/ricardobranco777/httpseek"
)

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func main() {
	log.SetFlags(0)

	f := io.ReadCloser(os.Stdin)
	var err error

	argc := len(os.Args)
	if argc > 2 {
		log.Fatalf("usage: %s [URL]\n", os.Args[0])
	} else if argc > 1 {
		url := os.Args[1]
		// httpseek.SetLogger(httpseek.StdLogger())
		f, err = httpseek.Open(url)
		if err != nil {
			log.Fatalf("open: %v", err)
		}
		defer f.Close()
	}

	err = libarchive.Walk(f, func(h *libarchive.Header) error {
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
