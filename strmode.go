/* SPDX-License-Identifier: BSD-2-Clause */

package archive

// StrMode converts a file mode to a string like "drwxr-xr-x".
// This is Golang implementation of BSD strmode(3)
// because io/fs's FileMode.String() sucks.
func StrMode(mode uint32) string {
	var b [11]byte

	// File type
	switch mode & S_IFMT {
	case S_IFDIR:
		b[0] = 'd'
	case S_IFCHR:
		b[0] = 'c'
	case S_IFBLK:
		b[0] = 'b'
	case S_IFREG:
		b[0] = '-'
	case S_IFLNK:
		b[0] = 'l'
	case S_IFSOCK:
		b[0] = 's'
	case S_IFIFO:
		b[0] = 'p'
	default:
		b[0] = '?'
	}

	// User permissions
	if mode&S_IRUSR != 0 {
		b[1] = 'r'
	} else {
		b[1] = '-'
	}
	if mode&S_IWUSR != 0 {
		b[2] = 'w'
	} else {
		b[2] = '-'
	}
	switch mode & (S_IXUSR | S_ISUID) {
	case 0:
		b[3] = '-'
	case S_IXUSR:
		b[3] = 'x'
	case S_ISUID:
		b[3] = 'S'
	case S_IXUSR | S_ISUID:
		b[3] = 's'
	}

	// Group permissions
	if mode&S_IRGRP != 0 {
		b[4] = 'r'
	} else {
		b[4] = '-'
	}
	if mode&S_IWGRP != 0 {
		b[5] = 'w'
	} else {
		b[5] = '-'
	}
	switch mode & (S_IXGRP | S_ISGID) {
	case 0:
		b[6] = '-'
	case S_IXGRP:
		b[6] = 'x'
	case S_ISGID:
		b[6] = 'S'
	case S_IXGRP | S_ISGID:
		b[6] = 's'
	}

	// Other permissions
	if mode&S_IROTH != 0 {
		b[7] = 'r'
	} else {
		b[7] = '-'
	}
	if mode&S_IWOTH != 0 {
		b[8] = 'w'
	} else {
		b[8] = '-'
	}
	switch mode & (S_IXOTH | S_ISVTX) {
	case 0:
		b[9] = '-'
	case S_IXOTH:
		b[9] = 'x'
	case S_ISVTX:
		b[9] = 'T'
	case S_IXOTH | S_ISVTX:
		b[9] = 't'
	}

	b[10] = ' '

	return string(b[:])
}
