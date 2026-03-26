package filestore

import "syscall"

func setBsize(st *syscall.Statfs_t, v uint64) {
	st.Bsize = int64(v)
}
