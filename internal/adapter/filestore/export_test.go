package filestore

import "syscall"

// StatfsFunc allows tests to replace the syscall.Statfs implementation.
var StatfsFunc = &statfs

// FakeStatfs returns a statfs function that simulates the given free/total block counts.
// bsize is passed as uint64 and narrowed to the platform-specific Bsize type via
// setBsize to avoid type mismatches between darwin (uint32) and linux (int64).
func FakeStatfs(bavail uint64, bsize uint64, blocks uint64) func(string, *syscall.Statfs_t) error {
	return func(_ string, st *syscall.Statfs_t) error {
		st.Bavail = bavail
		setBsize(st, bsize)
		st.Blocks = blocks
		return nil
	}
}
