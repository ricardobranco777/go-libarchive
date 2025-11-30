/* SPDX-License-Identifier: BSD-2-Clause */

package archive

import (
	"io/fs"

	"golang.org/x/sys/unix"
)

// Converts fs.FileMode to uint32
func FileModeToMode(m fs.FileMode) uint32 {
	var mode uint32

	// permissions
	mode |= uint32(m.Perm())

	// type
	switch {
	case m.IsDir():
		mode |= unix.S_IFDIR
	case m&fs.ModeSymlink != 0:
		mode |= unix.S_IFLNK
	case m&fs.ModeNamedPipe != 0:
		mode |= unix.S_IFIFO
	case m&fs.ModeSocket != 0:
		mode |= unix.S_IFSOCK
	case m&fs.ModeDevice != 0:
		if m&fs.ModeCharDevice != 0 {
			mode |= unix.S_IFCHR
		} else {
			mode |= unix.S_IFBLK
		}
	default:
		mode |= unix.S_IFREG
	}

	// special bits
	if m&fs.ModeSetuid != 0 {
		mode |= unix.S_ISUID
	}
	if m&fs.ModeSetgid != 0 {
		mode |= unix.S_ISGID
	}
	if m&fs.ModeSticky != 0 {
		mode |= unix.S_ISVTX
	}

	return mode
}

// StrMode converts a file mode to a string like "drwxr-xr-x".
// This is Golang implementation of BSD strmode(3)
// because io/fs's FileMode.String() sucks.
func StrMode(m fs.FileMode) string {
	var b [11]byte

	mode := FileModeToMode(m)

	// File type
	switch mode & unix.S_IFMT {
	case unix.S_IFDIR:
		b[0] = 'd'
	case unix.S_IFCHR:
		b[0] = 'c'
	case unix.S_IFBLK:
		b[0] = 'b'
	case unix.S_IFREG:
		b[0] = '-'
	case unix.S_IFLNK:
		b[0] = 'l'
	case unix.S_IFSOCK:
		b[0] = 's'
	case unix.S_IFIFO:
		b[0] = 'p'
	default:
		b[0] = '?'
	}

	// User permissions
	if mode&unix.S_IRUSR != 0 {
		b[1] = 'r'
	} else {
		b[1] = '-'
	}
	if mode&unix.S_IWUSR != 0 {
		b[2] = 'w'
	} else {
		b[2] = '-'
	}
	switch mode & (unix.S_IXUSR | unix.S_ISUID) {
	case 0:
		b[3] = '-'
	case unix.S_IXUSR:
		b[3] = 'x'
	case unix.S_ISUID:
		b[3] = 'S'
	case unix.S_IXUSR | unix.S_ISUID:
		b[3] = 's'
	}

	// Group permissions
	if mode&unix.S_IRGRP != 0 {
		b[4] = 'r'
	} else {
		b[4] = '-'
	}
	if mode&unix.S_IWGRP != 0 {
		b[5] = 'w'
	} else {
		b[5] = '-'
	}
	switch mode & (unix.S_IXGRP | unix.S_ISGID) {
	case 0:
		b[6] = '-'
	case unix.S_IXGRP:
		b[6] = 'x'
	case unix.S_ISGID:
		b[6] = 'S'
	case unix.S_IXGRP | unix.S_ISGID:
		b[6] = 's'
	}

	// Other permissions
	if mode&unix.S_IROTH != 0 {
		b[7] = 'r'
	} else {
		b[7] = '-'
	}
	if mode&unix.S_IWOTH != 0 {
		b[8] = 'w'
	} else {
		b[8] = '-'
	}
	switch mode & (unix.S_IXOTH | unix.S_ISVTX) {
	case 0:
		b[9] = '-'
	case unix.S_IXOTH:
		b[9] = 'x'
	case unix.S_ISVTX:
		b[9] = 'T'
	case unix.S_IXOTH | unix.S_ISVTX:
		b[9] = 't'
	}

	b[10] = ' '

	return string(b[:])
}
