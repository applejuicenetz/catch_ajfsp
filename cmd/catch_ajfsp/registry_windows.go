package main

import (
	"errors"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	keyRead   = 0x20019
	keyWrite  = 0x20006
	wow64Key  = 0x100 // always the 64-bit registry view, like the setups
	regSZ     = 1
	errFileNF = 2
	errPathNF = 3
	rootKey   = syscall.HKEY_CURRENT_USER
)

var (
	advapi         = syscall.NewLazyDLL("advapi32.dll")
	procCreateKey  = advapi.NewProc("RegCreateKeyExW")
	procSetValueEx = advapi.NewProc("RegSetValueExW")
)

// userStore reads and writes per-user protocol associations.
type userStore struct{}

func (userStore) Get(key, name string) (string, bool, error) {
	var h syscall.Handle
	err := syscall.RegOpenKeyEx(rootKey, syscall.StringToUTF16Ptr(key), 0, keyRead|wow64Key, &h)
	if err != nil {
		if isNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	defer syscall.RegCloseKey(h)

	n16 := syscall.StringToUTF16Ptr(name)
	var typ, size uint32
	if err := syscall.RegQueryValueEx(h, n16, nil, &typ, nil, &size); err != nil {
		if isNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	if typ != regSZ || size < 2 {
		return "", true, nil
	}
	buf := make([]uint16, size/2+1)
	size = uint32(len(buf)-1) * 2
	if err := syscall.RegQueryValueEx(h, n16, nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &size); err != nil {
		return "", false, err
	}
	n := int(size / 2)
	for n > 0 && buf[n-1] == 0 {
		n--
	}
	return string(utf16.Decode(buf[:n])), true, nil
}

func (userStore) Set(key, name, data string) error {
	var h syscall.Handle
	r, _, _ := procCreateKey.Call(uintptr(rootKey), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(key))),
		0, 0, 0, keyWrite|wow64Key, 0, uintptr(unsafe.Pointer(&h)), 0)
	if r != 0 {
		return syscall.Errno(r)
	}
	defer syscall.RegCloseKey(h)

	val := utf16.Encode([]rune(data))
	val = append(val, 0)
	r, _, _ = procSetValueEx.Call(uintptr(h), uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
		0, regSZ, uintptr(unsafe.Pointer(&val[0])), uintptr(len(val)*2))
	if r != 0 {
		return syscall.Errno(r)
	}
	return nil
}

func isNotFound(err error) bool {
	var e syscall.Errno
	return errors.As(err, &e) && (e == errFileNF || e == errPathNF)
}
