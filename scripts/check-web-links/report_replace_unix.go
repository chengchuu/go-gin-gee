//go:build !windows

package main

import "os"

func replaceReportFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
