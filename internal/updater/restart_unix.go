//go:build !windows

package updater

import (
	"os"
	"path/filepath"
	"syscall"
)

// RestartSelf menggantikan proses yang sedang berjalan dengan binary yang
// baru dipasang (re-exec). Pada sukses, fungsi ini tidak pernah kembali —
// seluruh state proses digantikan oleh binary baru dengan argumen & env sama.
func RestartSelf() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}
	return syscall.Exec(exePath, os.Args, os.Environ())
}
