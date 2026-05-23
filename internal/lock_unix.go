//go:build !windows

package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var lockFileObj *os.File

func AcquireLock() error {
	tmpDir := os.TempDir()
	lockPath := filepath.Join(tmpDir, lockFile)

	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir archivo de lock: %v", err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return errors.New("⚠️ Clipboard Monitor ya está en ejecución")
	}

	if err := f.Truncate(0); err != nil {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		return fmt.Errorf("error al truncar lock: %v", err)
	}

	pid := fmt.Sprintf("%d\n", os.Getpid())
	if _, err := f.WriteString(pid); err != nil {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		return fmt.Errorf("error al escribir PID: %v", err)
	}

	lockFileObj = f
	return nil
}

func ReleaseLock() {
	if lockFileObj != nil {
		syscall.Flock(int(lockFileObj.Fd()), syscall.LOCK_UN)
		lockFileObj.Close()
		os.Remove(filepath.Join(os.TempDir(), lockFile))
		lockFileObj = nil
	}
}
