package filestore

import "syscall"

// StatfsFunc allows tests to replace the syscall.Statfs implementation.
var StatfsFunc = &statfs

// FakeStatfs returns a statfs function that simulates the given free/total block counts.
func FakeStatfs(bavail uint64, bsize uint32, blocks uint64) func(string, *syscall.Statfs_t) error {
	return func(_ string, st *syscall.Statfs_t) error {
		st.Bavail = bavail
		st.Bsize = bsize
		st.Blocks = blocks
		return nil
	}
}
