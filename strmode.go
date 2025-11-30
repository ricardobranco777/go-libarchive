/* SPDX-License-Identifier: BSD-2-Clause */

package archive

import (
	"io/fs"
	"syscall"
)

// Converts fs.FileMode to uint32
func FileModeToMode(m fs.FileMode) uint32 {
	var mode uint32

	// permissions
	mode |= uint32(m.Perm())

	// type
	switch {
	case m.IsDir():
		mode |= syscall.S_IFDIR
	case m&fs.ModeSymlink != 0:
		mode |= syscall.S_IFLNK
	case m&fs.ModeNamedPipe != 0:
		mode |= syscall.S_IFIFO
	case m&fs.ModeSocket != 0:
		mode |= syscall.S_IFSOCK
	case m&fs.ModeDevice != 0:
		if m&fs.ModeCharDevice != 0 {
			mode |= syscall.S_IFCHR
		} else {
			mode |= syscall.S_IFBLK
		}
	default:
		mode |= syscall.S_IFREG
	}

	// special bits
	if m&fs.ModeSetuid != 0 {
		mode |= syscall.S_ISUID
	}
	if m&fs.ModeSetgid != 0 {
		mode |= syscall.S_ISGID
	}
	if m&fs.ModeSticky != 0 {
		mode |= syscall.S_ISVTX
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
	switch mode & syscall.S_IFMT {
	case syscall.S_IFDIR:
		b[0] = 'd'
	case syscall.S_IFCHR:
		b[0] = 'c'
	case syscall.S_IFBLK:
		b[0] = 'b'
	case syscall.S_IFREG:
		b[0] = '-'
	case syscall.S_IFLNK:
		b[0] = 'l'
	case syscall.S_IFSOCK:
		b[0] = 's'
	case syscall.S_IFIFO:
		b[0] = 'p'
	default:
		b[0] = '?'
	}

	// User permissions
	if mode&syscall.S_IRUSR != 0 {
		b[1] = 'r'
	} else {
		b[1] = '-'
	}
	if mode&syscall.S_IWUSR != 0 {
		b[2] = 'w'
	} else {
		b[2] = '-'
	}
	switch mode & (syscall.S_IXUSR | syscall.S_ISUID) {
	case 0:
		b[3] = '-'
	case syscall.S_IXUSR:
		b[3] = 'x'
	case syscall.S_ISUID:
		b[3] = 'S'
	case syscall.S_IXUSR | syscall.S_ISUID:
		b[3] = 's'
	}

	// Group permissions
	if mode&syscall.S_IRGRP != 0 {
		b[4] = 'r'
	} else {
		b[4] = '-'
	}
	if mode&syscall.S_IWGRP != 0 {
		b[5] = 'w'
	} else {
		b[5] = '-'
	}
	switch mode & (syscall.S_IXGRP | syscall.S_ISGID) {
	case 0:
		b[6] = '-'
	case syscall.S_IXGRP:
		b[6] = 'x'
	case syscall.S_ISGID:
		b[6] = 'S'
	case syscall.S_IXGRP | syscall.S_ISGID:
		b[6] = 's'
	}

	// Other permissions
	if mode&syscall.S_IROTH != 0 {
		b[7] = 'r'
	} else {
		b[7] = '-'
	}
	if mode&syscall.S_IWOTH != 0 {
		b[8] = 'w'
	} else {
		b[8] = '-'
	}
	switch mode & (syscall.S_IXOTH | syscall.S_ISVTX) {
	case 0:
		b[9] = '-'
	case syscall.S_IXOTH:
		b[9] = 'x'
	case syscall.S_ISVTX:
		b[9] = 'T'
	case syscall.S_IXOTH | syscall.S_ISVTX:
		b[9] = 't'
	}

	b[10] = ' '

	return string(b[:])
}
