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

		t := ""
		if e.IsDir() {
			t = "/"
		}
		fmt.Printf("%s%s/%s %5d %s %s%s", libarchive.StrMode(e.Mode()), e.Uname, e.Gname, e.Size(), formatTime(e.ModTime()), e.Name(), t)

		if e.Linkname != "" {
			fmt.Printf(" -> %s", e.Linkname)
		}
		fmt.Println()
	}
}
