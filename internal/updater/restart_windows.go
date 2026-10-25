//go:build windows

package updater

import (
	"os"
	"os/exec"
	"path/filepath"
)

// RestartSelf pada Windows tidak bisa menggantikan proses berjalan (Exec
// tidak tersedia), maka binary baru diluncurkan terpisah lalu proses lama
// keluar. Pada sukses, fungsi ini tidak pernah kembali.
func RestartSelf() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.Dir = filepath.Dir(exePath)
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
