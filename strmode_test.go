package archive

import (
	"io/fs"
	"testing"
)

func TestStrMode(t *testing.T) {
	tests := []struct {
		mode fs.FileMode
		want string
	}{
		// basic perms
		{0o0000, "---------- "},
		{0o0644, "-rw-r--r-- "},
		{0o0755, "-rwxr-xr-x "},
		{0o0777, "-rwxrwxrwx "},

		// directories
		{fs.ModeDir | 0o755, "drwxr-xr-x "},
		{fs.ModeDir | 0o777, "drwxrwxrwx "},

		// sticky bit
		{fs.ModeDir | fs.ModeSticky | 0o777, "drwxrwxrwt "},
		{fs.ModeSticky, "---------T "},

		// setuid
		{fs.ModeSetuid | 0o4755, "-rwsr-xr-x "},
		{fs.ModeSetuid | 0o0400, "-r-S------ "},

		// setgid
		{fs.ModeSetgid | 0o2755, "-rwxr-sr-x "},
		{fs.ModeSetgid, "------S--- "},

		// character & block device
		{fs.ModeDevice | fs.ModeCharDevice, "c--------- "},
		{fs.ModeDevice, "b--------- "},

		// fifo, socket, symlink
		{fs.ModeNamedPipe | 0o644, "prw-r--r-- "},
		{fs.ModeSocket | 0o777, "srwxrwxrwx "},
		{fs.ModeSymlink | 0o777, "lrwxrwxrwx "},
	}

	for _, tt := range tests {
		got := StrMode(tt.mode)
		if got != tt.want {
			t.Errorf("StrMode(%#o) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}
