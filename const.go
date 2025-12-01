/* SPDX-License-Identifier: BSD-2-Clause */

package archive

const (
	S_IFMT   = 0xf000
	S_IFSOCK = 0xc000
	S_IFLNK  = 0xa000
	S_IFREG  = 0x8000
	S_IFBLK  = 0x6000
	S_IFDIR  = 0x4000
	S_IFCHR  = 0x2000
	S_IFIFO  = 0x1000
	S_ISUID  = 0x800
	S_ISGID  = 0x400
	S_ISVTX  = 0x200
	S_IRUSR  = 0x100
	S_IWUSR  = 0x80
	S_IXUSR  = 0x40
	S_IRGRP  = 0x20
	S_IWGRP  = 0x10
	S_IXGRP  = 0x8
	S_IROTH  = 0x4
	S_IWOTH  = 0x2
	S_IXOTH  = 0x1
)
