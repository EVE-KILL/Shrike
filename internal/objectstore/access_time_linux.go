//go:build linux

package objectstore

import (
	"io/fs"
	"syscall"
	"time"
)

func fileAccessTime(info fs.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
	}
	return info.ModTime()
}
