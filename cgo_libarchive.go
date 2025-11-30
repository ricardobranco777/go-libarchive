//go:build !windows
// +build !windows

package archive

/*
#cgo freebsd CFLAGS: -I/usr/local/include
#cgo freebsd LDFLAGS: -L/usr/local/lib -larchive

#cgo linux pkg-config: libarchive

#cgo darwin CFLAGS: -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/opt/homebrew/lib -larchive
*/
import "C"
