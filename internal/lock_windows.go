//go:build windows

package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx   = kernel32.NewProc("LockFileEx")
	unlockFileEx = kernel32.NewProc("UnlockFileEx")
	writeFile    = kernel32.NewProc("WriteFile")
)

var lockHandle syscall.Handle

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 2
	LOCKFILE_FAIL_IMMEDIATELY = 1
)

func AcquireLock() error {
	tmpDir := os.TempDir()
	lockPath := filepath.Join(tmpDir, lockFile)

	pathp, err := syscall.UTF16PtrFromString(lockPath)
	if err != nil {
		return fmt.Errorf("error al convertir path: %v", err)
	}

	handle, err := syscall.CreateFile(
		pathp,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.CREATE_ALWAYS,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return fmt.Errorf("error al crear archivo de lock: %v", err)
	}

	flags := LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY
	overlapped := &syscall.Overlapped{}

	r, _, err := lockFileEx.Call(
		uintptr(handle),
		uintptr(flags),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(overlapped)),
	)

	if r == 0 {
		syscall.CloseHandle(handle)
		return errors.New("⚠️ Clipboard Monitor ya está en ejecución")
	}

	lockHandle = handle

	pid := fmt.Sprintf("%d", os.Getpid())
	var written uint32
	_, _, err = writeFile.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&[]byte(pid)[0])),
		uintptr(len(pid)),
		uintptr(unsafe.Pointer(&written)),
		0,
	)
	if err != nil && err != syscall.Errno(0) {
		ReleaseLock()
		return fmt.Errorf("error al escribir PID: %v", err)
	}

	return nil
}

func ReleaseLock() {
	if lockHandle != 0 {
		overlapped := &syscall.Overlapped{}
		_, _, err := unlockFileEx.Call(
			uintptr(lockHandle),
			0,
			1,
			0,
			uintptr(unsafe.Pointer(overlapped)),
		)
		if err != nil {
			return
		}
		err2 := syscall.CloseHandle(lockHandle)
		if err2 != nil {
			return
		}
		err3 := os.Remove(filepath.Join(os.TempDir(), lockFile))
		if err3 != nil {
			return
		}
		lockHandle = 0
	}
}
