//go:build windows

package hide_files

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	fileAttributeHidden uint32 = 0x2
	// さらに見えにくくしたいなら SYSTEM も（任意）
	// fileAttributeSystem uint32 = 0x4
)

func hideFolder(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("not a directory: %s", abs)
	}

	p, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return err
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return err
	}
	newAttrs := uint32(attrs) | fileAttributeHidden
	// SYSTEMも付けたいなら: newAttrs |= fileAttributeSystem

	if newAttrs == uint32(attrs) {
		return nil
	}
	return syscall.SetFileAttributes(p, newAttrs)
}

func unhideFolder(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	p, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return err
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return err
	}
	newAttrs := uint32(attrs) &^ fileAttributeHidden
	// SYSTEMも外すなら: newAttrs &^= fileAttributeSystem

	if newAttrs == uint32(attrs) {
		return nil
	}
	return syscall.SetFileAttributes(p, newAttrs)
}
