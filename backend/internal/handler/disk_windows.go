package handler

import (
	"os"

	"golang.org/x/sys/windows"
)

func diskUsage() (total, free uint64) {
	cwd, err := os.Getwd()
	if err != nil {
		return 0, 0
	}
	path, err := windows.UTF16PtrFromString(cwd)
	if err != nil {
		return 0, 0
	}
	if err := windows.GetDiskFreeSpaceEx(path, nil, &total, &free); err != nil {
		return 0, 0
	}
	return total, free
}
