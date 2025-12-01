package archive

import (
	"testing"
)

func TestStrMode(t *testing.T) {
	tests := []struct {
		mode uint32
		want string
	}{
		// basic perms
		{S_IFREG, "---------- "},
		{S_IFREG | 0o0644, "-rw-r--r-- "},
		{S_IFREG | 0o0755, "-rwxr-xr-x "},
		{S_IFREG | 0o0777, "-rwxrwxrwx "},

		// directories
		{S_IFDIR | 0o755, "drwxr-xr-x "},
		{S_IFDIR | 0o777, "drwxrwxrwx "},

		// sticky bit
		{S_IFDIR | S_ISVTX | 0o777, "drwxrwxrwt "},
		{S_IFREG | S_ISVTX, "---------T "},

		// setuid
		{S_IFREG | S_ISUID | 0o4755, "-rwsr-xr-x "},
		{S_IFREG | S_ISUID | 0o0400, "-r-S------ "},

		// setgid
		{S_IFREG | S_ISGID | 0o2755, "-rwxr-sr-x "},
		{S_IFREG | S_ISGID, "------S--- "},

		// character & block device
		{S_IFCHR, "c--------- "},
		{S_IFBLK, "b--------- "},

		// fifo, socket, symlink
		{S_IFIFO | 0o644, "prw-r--r-- "},
		{S_IFSOCK | 0o777, "srwxrwxrwx "},
		{S_IFLNK | 0o777, "lrwxrwxrwx "},
	}

	for _, tt := range tests {
		got := StrMode(tt.mode)
		if got != tt.want {
			t.Errorf("StrMode(%#o) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}
