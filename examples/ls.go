/* SPDX-License-Identifier: BSD-2-Clause */

package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"time"

	libarchive "github.com/ricardobranco777/go-libarchive"
)

// fileTypeChar returns the leading type character used by ls -l / tar tvf.
func fileTypeChar(m fs.FileMode) byte {
	switch {
	case m.IsDir():
		return 'd'
	case m&fs.ModeSymlink != 0:
		return 'l'
	case m&fs.ModeDevice != 0:
		if m&fs.ModeCharDevice != 0 {
			return 'c'
		}
		return 'b'
	case m&fs.ModeNamedPipe != 0:
		return 'p'
	case m&fs.ModeSocket != 0:
		return 's'
	default:
		return '-' // regular file
	}
}

func formatMode(m os.FileMode) string {
	perms := []byte("---------")
	p := m.Perm()
	rwx := []int{0400, 0200, 0100, 040, 020, 010, 04, 02, 01}
	for i, bit := range rwx {
		if p&os.FileMode(bit) != 0 {
			perms[i] = "rwxrwxrwx"[i]
		}
	}

	return string(perms)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "----------"
	}
	now := time.Now()
	year := now.Year()
	layout := "Jan _2 15:04" // the tar-style time layout
	if t.Year() != year {
		layout = "Jan _2  2006"
	}
	return t.Format(layout)
}

func main() {
	ar, err := libarchive.OpenReader(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	defer ar.Close()

	for {
		e, err := ar.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		mode := e.Mode()
		nlink := e.Nlink()
		uid := e.UID()
		gid := e.GID()
		size := e.Size()
		mtime := e.ModTime()
		name := e.Name()
		link := e.Linkname()

		t := fileTypeChar(e.Type())

		fmt.Printf(
			"%c%s %3d %5d %5d %10d %s %s",
			t,
			formatMode(mode),
			nlink,
			uid,
			gid,
			size,
			formatTime(mtime),
			name,
		)

		if link != "" {
			fmt.Printf(" -> %s", link)
		}

		fmt.Println()

		// No need to drain data – libarchive auto-skips on Next()
	}
}
